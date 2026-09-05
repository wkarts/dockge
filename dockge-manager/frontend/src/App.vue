<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  api,
  getToken,
  login,
  setToken,
  type Application,
  type AuditEvent,
  type Deployment,
  type Operation,
  type Stack,
  type Target
} from "./api";

type Tab = "infrastructure" | "deployments" | "activity";

const authenticated = ref(Boolean(getToken()));
const email = ref("");
const password = ref("");
const error = ref("");
const notice = ref("");
const busy = ref(false);
const tab = ref<Tab>("infrastructure");

const targets = ref<Target[]>([]);
const selectedTargetId = ref("");
const stacks = ref<Stack[]>([]);
const applications = ref<Application[]>([]);
const deployments = ref<Deployment[]>([]);
const operations = ref<Operation[]>([]);
const auditEvents = ref<AuditEvent[]>([]);
const logs = ref("");
const logsTitle = ref("");
const pendingDeleteTarget = ref<Target | null>(null);

const newTarget = ref({ name: "", base_url: "", token: "", verify_tls: true });
const newApplication = ref({ name: "", description: "" });
const newDeployment = ref({
  application_id: "",
  target_id: "",
  stack_name: "",
  compose_yaml: "",
  compose_env: "",
  adopt_external: false
});

const selectedTarget = computed(() => targets.value.find((t) => t.id === selectedTargetId.value));
const healthyDeployments = computed(() => deployments.value.filter((d) => d.status === "HEALTHY").length);
const failedDeployments = computed(() => deployments.value.filter((d) => ["FAILED", "ROLLBACK_FAILED"].includes(d.status)).length);
const runningStacks = computed(() => stacks.value.length);

function applicationName(id: string): string {
  return applications.value.find((item) => item.id === id)?.name || id.slice(0, 8);
}

function targetName(id: string): string {
  return targets.value.find((item) => item.id === id)?.name || id.slice(0, 8);
}

function statusClass(status: string): string {
  if (["HEALTHY", "ROLLED_BACK", "SUCCEEDED"].includes(status)) return "status-ok";
  if (["FAILED", "ROLLBACK_FAILED"].includes(status)) return "status-error";
  return "status-pending";
}

async function run(task: () => Promise<void>, success = "") {
  error.value = "";
  notice.value = "";
  busy.value = true;
  try {
    await task();
    if (success) notice.value = success;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    busy.value = false;
  }
}

async function loadTargets() {
  targets.value = await api.targets();
  if (selectedTargetId.value && !targets.value.some((t) => t.id === selectedTargetId.value)) {
    selectedTargetId.value = "";
  }
  if (!selectedTargetId.value && targets.value.length) selectedTargetId.value = targets.value[0].id;
  if (!newDeployment.value.target_id && targets.value.length) newDeployment.value.target_id = targets.value[0].id;
  if (selectedTargetId.value) await loadStacks();
}

async function loadStacks() {
  if (!selectedTargetId.value) {
    stacks.value = [];
    return;
  }
  stacks.value = (await api.stacks(selectedTargetId.value)).stacks || [];
}

async function loadDeployments() {
  applications.value = await api.applications();
  deployments.value = await api.deployments();
  if (!newDeployment.value.application_id && applications.value.length) {
    newDeployment.value.application_id = applications.value[0].id;
  }
}

async function loadActivity() {
  [operations.value, auditEvents.value] = await Promise.all([api.operations(), api.audit()]);
}

async function loadAll() {
  await loadTargets();
  await loadDeployments();
  await loadActivity();
}

async function doLogin() {
  await run(async () => {
    await login(email.value, password.value);
    authenticated.value = true;
    await loadAll();
  });
}

function logout() {
  setToken("");
  authenticated.value = false;
  targets.value = [];
  stacks.value = [];
  applications.value = [];
  deployments.value = [];
  operations.value = [];
  auditEvents.value = [];
}

async function addTarget() {
  await run(async () => {
    await api.createTarget(newTarget.value);
    newTarget.value = { name: "", base_url: "", token: "", verify_tls: true };
    await loadTargets();
  }, "Dockge Target cadastrado.");
}

async function testTarget(target: Target) {
  await run(async () => {
    await api.testTarget(target.id);
    await loadTargets();
  }, `Conexão com ${target.name} validada.`);
}

async function deleteTarget(target: Target) {
  await run(async () => {
    await api.deleteTarget(target.id);
    pendingDeleteTarget.value = null;
    selectedTargetId.value = "";
    await loadTargets();
    await loadDeployments();
  }, "Target removido somente do Manager. Nenhuma stack remota foi removida.");
}

async function stackAction(stack: Stack, action: string) {
  if (!selectedTargetId.value) return;
  await run(async () => {
    await api.action(selectedTargetId.value, stack.name, action);
    await loadStacks();
    await loadActivity();
  }, `${stack.name}: ${action} concluído.`);
}

async function showLogs(stack: Stack) {
  if (!selectedTargetId.value) return;
  await run(async () => {
    const result = await api.logs(selectedTargetId.value, stack.name);
    logsTitle.value = `${selectedTarget.value?.name || "Dockge"} / ${stack.name}`;
    logs.value = [result.stdout, result.stderr].filter(Boolean).join("\n") || "Sem saída de log.";
  });
}

async function addApplication() {
  await run(async () => {
    const created = await api.createApplication(newApplication.value);
    newApplication.value = { name: "", description: "" };
    await loadDeployments();
    newDeployment.value.application_id = created.id;
  }, "Aplicação cadastrada.");
}

async function addDeployment() {
  await run(async () => {
    await api.createDeployment(newDeployment.value);
    const appId = newDeployment.value.application_id;
    const targetId = newDeployment.value.target_id;
    newDeployment.value = {
      application_id: appId,
      target_id: targetId,
      stack_name: "",
      compose_yaml: "",
      compose_env: "",
      adopt_external: false
    };
    await loadDeployments();
  }, "Deployment criado em estado DRAFT. Nada foi executado no Dockge ainda.");
}

async function deploy(item: Deployment) {
  await run(async () => {
    await api.deploy(item.id);
    await loadDeployments();
    await loadActivity();
    if (selectedTargetId.value === item.target_id) await loadStacks();
  }, `${item.stack_name}: deployment verificado como saudável.`);
}

async function rollback(item: Deployment) {
  await run(async () => {
    await api.rollback(item.id);
    await loadDeployments();
    await loadActivity();
    if (selectedTargetId.value === item.target_id) await loadStacks();
  }, `${item.stack_name}: rollback concluído e verificado.`);
}

onMounted(async () => {
  if (!authenticated.value) return;
  await run(async () => {
    try {
      await api.me();
      await loadAll();
    } catch {
      logout();
    }
  });
});
</script>

<template>
  <main class="page">
    <header class="topbar">
      <div>
        <div class="eyebrow">DOCKGE ECOSYSTEM</div>
        <h1>Dockge Manager</h1>
        <p>Management plane API-first para uma ou várias instalações Dockge.</p>
      </div>
      <button v-if="authenticated" class="secondary" @click="logout">Sair</button>
    </header>

    <div v-if="error" class="alert">{{ error }}</div>
    <div v-if="notice" class="notice">{{ notice }}</div>

    <section v-if="!authenticated" class="card login-card">
      <h2>Entrar</h2>
      <label>E-mail<input v-model="email" type="email" autocomplete="username" /></label>
      <label>Senha<input v-model="password" type="password" autocomplete="current-password" @keyup.enter="doLogin" /></label>
      <button :disabled="busy" @click="doLogin">Entrar</button>
    </section>

    <template v-else>
      <nav class="tabs" aria-label="Navegação principal">
        <button :class="{ active: tab === 'infrastructure' }" @click="tab = 'infrastructure'">Infraestrutura</button>
        <button :class="{ active: tab === 'deployments' }" @click="tab = 'deployments'">Deployments</button>
        <button :class="{ active: tab === 'activity' }" @click="tab = 'activity'; loadActivity()">Atividade</button>
      </nav>

      <section class="stats-grid">
        <div class="stat"><span>Dockges</span><strong>{{ targets.length }}</strong><small>{{ targets.filter(t => t.last_seen_at).length }} verificados</small></div>
        <div class="stat"><span>Stacks visíveis</span><strong>{{ runningStacks }}</strong><small>{{ selectedTarget?.name || 'nenhum Target' }}</small></div>
        <div class="stat"><span>Deployments saudáveis</span><strong>{{ healthyDeployments }}</strong><small>{{ deployments.length }} cadastrados</small></div>
        <div class="stat"><span>Falhas</span><strong>{{ failedDeployments }}</strong><small>rollback automático quando aplicável</small></div>
      </section>

      <template v-if="tab === 'infrastructure'">
        <section class="grid">
          <div class="card">
            <div class="section-title"><h2>Dockge Targets</h2><button class="secondary" :disabled="busy" @click="loadTargets">Atualizar</button></div>
            <div class="target-list">
              <button
                v-for="target in targets"
                :key="target.id"
                class="target-row"
                :class="{ selected: selectedTargetId === target.id }"
                @click="selectedTargetId = target.id; loadStacks()"
              >
                <strong>{{ target.name }}</strong>
                <span>{{ target.base_url }}</span>
                <small>{{ target.last_version ? `v${target.last_version}` : 'não verificado' }}</small>
              </button>
              <p v-if="!targets.length" class="muted">Nenhum Dockge cadastrado.</p>
            </div>
            <div v-if="selectedTarget" class="row-actions">
              <button class="secondary" @click="testTarget(selectedTarget)">Testar conexão</button>
              <button class="danger" @click="pendingDeleteTarget = selectedTarget">Remover do Manager</button>
            </div>
            <div v-if="pendingDeleteTarget" class="inline-confirm">
              <strong>Remover {{ pendingDeleteTarget.name }} do Manager?</strong>
              <p>Isso remove apenas cadastro e credencial local. Stacks, containers e volumes remotos não são tocados.</p>
              <div class="row-actions">
                <button class="danger" :disabled="busy" @click="deleteTarget(pendingDeleteTarget)">Confirmar remoção</button>
                <button class="secondary" @click="pendingDeleteTarget = null">Cancelar</button>
              </div>
            </div>
          </div>

          <div class="card">
            <h2>Novo Target</h2>
            <label>Nome<input v-model="newTarget.name" placeholder="produção-01" /></label>
            <label>URL<input v-model="newTarget.base_url" placeholder="https://dockge.exemplo.com" /></label>
            <label>Bearer token<input v-model="newTarget.token" type="password" placeholder="token da Automation API" /></label>
            <label class="checkbox"><input v-model="newTarget.verify_tls" type="checkbox" /> Validar TLS</label>
            <button :disabled="busy || !newTarget.name || !newTarget.base_url || !newTarget.token" @click="addTarget">Cadastrar</button>
          </div>
        </section>

        <section class="card">
          <div class="section-title">
            <div><h2>Stacks</h2><p>{{ selectedTarget?.name || 'Selecione um Target' }}</p></div>
            <button class="secondary" :disabled="busy || !selectedTargetId" @click="loadStacks">Recarregar</button>
          </div>
          <div class="table-wrap">
            <table>
              <thead><tr><th>Stack</th><th>Origem</th><th>Ações</th></tr></thead>
              <tbody>
                <tr v-for="stack in stacks" :key="stack.name">
                  <td><strong>{{ stack.name }}</strong></td>
                  <td><span class="badge">{{ stack.api_managed ? 'Automation API' : 'existente' }}</span></td>
                  <td class="actions">
                    <button @click="stackAction(stack, 'start')">Start</button>
                    <button class="secondary" @click="stackAction(stack, 'restart')">Restart</button>
                    <button class="secondary" @click="stackAction(stack, 'stop')">Stop</button>
                    <button class="secondary" @click="showLogs(stack)">Logs</button>
                  </td>
                </tr>
                <tr v-if="!stacks.length"><td colspan="3">Nenhuma stack retornada.</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <section v-if="logs" class="card">
          <div class="section-title"><div><h2>Logs</h2><p>{{ logsTitle }}</p></div><button class="secondary" @click="logs = ''">Fechar</button></div>
          <pre>{{ logs }}</pre>
        </section>
      </template>

      <template v-else-if="tab === 'deployments'">
        <section class="grid">
          <div class="card">
            <h2>Aplicações</h2>
            <div class="simple-list">
              <div v-for="item in applications" :key="item.id"><strong>{{ item.name }}</strong><small>{{ item.description || 'Sem descrição' }}</small></div>
              <p v-if="!applications.length" class="muted">Cadastre a primeira aplicação lógica.</p>
            </div>
            <hr />
            <label>Nome<input v-model="newApplication.name" placeholder="connect-api" /></label>
            <label>Descrição<input v-model="newApplication.description" placeholder="Aplicação / serviço" /></label>
            <button :disabled="busy || !newApplication.name" @click="addApplication">Cadastrar aplicação</button>
          </div>

          <div class="card deployment-form">
            <h2>Novo Deployment</h2>
            <label>Aplicação
              <select v-model="newDeployment.application_id"><option value="">Selecione</option><option v-for="item in applications" :key="item.id" :value="item.id">{{ item.name }}</option></select>
            </label>
            <label>Dockge Target
              <select v-model="newDeployment.target_id"><option value="">Selecione</option><option v-for="target in targets" :key="target.id" :value="target.id">{{ target.name }}</option></select>
            </label>
            <label>Nome da stack<input v-model="newDeployment.stack_name" placeholder="connect-api-prod" /></label>
            <label>compose.yaml<textarea v-model="newDeployment.compose_yaml" rows="10" spellcheck="false" placeholder="services:\n  app:\n    image: ..." /></label>
            <label>.env<textarea v-model="newDeployment.compose_env" rows="5" spellcheck="false" placeholder="APP_ENV=production" /></label>
            <label class="checkbox"><input v-model="newDeployment.adopt_external" type="checkbox" /> Adotar explicitamente uma stack existente com o mesmo nome</label>
            <p class="muted">Criar o deployment não altera o Dockge. A execução acontece somente ao clicar em Deploy.</p>
            <button :disabled="busy || !newDeployment.application_id || !newDeployment.target_id || !newDeployment.stack_name || !newDeployment.compose_yaml" @click="addDeployment">Criar DRAFT</button>
          </div>
        </section>

        <section class="card">
          <div class="section-title"><div><h2>Deployments</h2><p>Snapshot → Apply → Up → Verify → Commit ou rollback automático</p></div><button class="secondary" @click="loadDeployments">Atualizar</button></div>
          <div class="deployment-list">
            <article v-for="item in deployments" :key="item.id" class="deployment-row">
              <div>
                <div class="deployment-heading">
                  <strong>{{ applicationName(item.application_id) }}</strong>
                  <span :class="['status', statusClass(item.status)]">{{ item.status }}</span>
                </div>
                <p>{{ targetName(item.target_id) }} / <code>{{ item.stack_name }}</code></p>
                <small>revisão desejada {{ item.current_revision }} · ativa {{ item.active_revision || 'nenhuma' }}</small>
                <p v-if="item.last_error" class="error-text">{{ item.last_error }}</p>
              </div>
              <div class="actions">
                <button :disabled="busy" @click="deploy(item)">Deploy</button>
                <button class="secondary" :disabled="busy || item.active_revision <= 1" @click="rollback(item)">Rollback</button>
              </div>
            </article>
            <p v-if="!deployments.length" class="muted">Nenhum deployment cadastrado.</p>
          </div>
        </section>
      </template>

      <template v-else>
        <section class="grid">
          <div class="card">
            <div class="section-title"><h2>Operações</h2><button class="secondary" @click="loadActivity">Atualizar</button></div>
            <div class="activity-list">
              <div v-for="item in operations" :key="item.id">
                <span :class="['status', statusClass(item.status)]">{{ item.status }}</span>
                <strong>{{ item.action }}</strong>
                <code>{{ item.stack_name }}</code>
                <small>{{ targetName(item.target_id) }} · {{ new Date(item.created_at).toLocaleString() }}</small>
              </div>
              <p v-if="!operations.length" class="muted">Nenhuma operação registrada.</p>
            </div>
          </div>

          <div class="card">
            <h2>Auditoria</h2>
            <div class="activity-list">
              <div v-for="item in auditEvents" :key="item.id">
                <strong>{{ item.event_type }}</strong>
                <code>{{ item.resource || '—' }}</code>
                <small>{{ item.actor }} · {{ new Date(item.created_at).toLocaleString() }}</small>
              </div>
              <p v-if="!auditEvents.length" class="muted">Nenhum evento registrado.</p>
            </div>
          </div>
        </section>
      </template>
    </template>
  </main>
</template>
