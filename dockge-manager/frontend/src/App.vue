<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api, getToken, login, setToken, type Stack, type Target } from "./api";

const authenticated = ref(Boolean(getToken()));
const email = ref("");
const password = ref("");
const error = ref("");
const busy = ref(false);
const targets = ref<Target[]>([]);
const selectedTargetId = ref("");
const stacks = ref<Stack[]>([]);
const logs = ref("");
const newTarget = ref({ name: "", base_url: "", token: "", verify_tls: true });

const selectedTarget = computed(() => targets.value.find((t) => t.id === selectedTargetId.value));

async function run(task: () => Promise<void>) {
  error.value = "";
  busy.value = true;
  try { await task(); } catch (e) { error.value = e instanceof Error ? e.message : String(e); }
  finally { busy.value = false; }
}

async function loadTargets() {
  targets.value = await api.targets();
  if (!selectedTargetId.value && targets.value.length) selectedTargetId.value = targets.value[0].id;
  if (selectedTargetId.value) await loadStacks();
}

async function loadStacks() {
  if (!selectedTargetId.value) { stacks.value = []; return; }
  stacks.value = (await api.stacks(selectedTargetId.value)).stacks || [];
}

async function doLogin() {
  await run(async () => {
    await login(email.value, password.value);
    authenticated.value = true;
    await loadTargets();
  });
}

function logout() {
  setToken("");
  authenticated.value = false;
  targets.value = [];
  stacks.value = [];
}

async function addTarget() {
  await run(async () => {
    await api.createTarget(newTarget.value);
    newTarget.value = { name: "", base_url: "", token: "", verify_tls: true };
    await loadTargets();
  });
}

async function testTarget(target: Target) {
  await run(async () => { await api.testTarget(target.id); await loadTargets(); });
}

async function deleteTarget(target: Target) {
  if (!confirm(`Remover o alvo ${target.name}?`)) return;
  await run(async () => { await api.deleteTarget(target.id); selectedTargetId.value = ""; await loadTargets(); });
}

async function stackAction(stack: Stack, action: string) {
  if (!selectedTargetId.value) return;
  await run(async () => { await api.action(selectedTargetId.value, stack.name, action); await loadStacks(); });
}

async function showLogs(stack: Stack) {
  if (!selectedTargetId.value) return;
  await run(async () => {
    const result = await api.logs(selectedTargetId.value, stack.name);
    logs.value = [result.stdout, result.stderr].filter(Boolean).join("\n");
  });
}

onMounted(async () => {
  if (!authenticated.value) return;
  await run(async () => {
    try { await api.me(); await loadTargets(); }
    catch { logout(); }
  });
});
</script>

<template>
  <main class="page">
    <header class="topbar">
      <div>
        <h1>Dockge Manager</h1>
        <p>Management plane API-first para uma ou várias instalações Dockge.</p>
      </div>
      <button v-if="authenticated" class="secondary" @click="logout">Sair</button>
    </header>

    <div v-if="error" class="alert">{{ error }}</div>

    <section v-if="!authenticated" class="card login-card">
      <h2>Entrar</h2>
      <label>E-mail<input v-model="email" type="email" autocomplete="username" /></label>
      <label>Senha<input v-model="password" type="password" autocomplete="current-password" @keyup.enter="doLogin" /></label>
      <button :disabled="busy" @click="doLogin">Entrar</button>
    </section>

    <template v-else>
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
              <small>{{ target.last_version ? `v${target.last_version}` : "não verificado" }}</small>
            </button>
          </div>
          <div v-if="selectedTarget" class="row-actions">
            <button class="secondary" @click="testTarget(selectedTarget)">Testar conexão</button>
            <button class="danger" @click="deleteTarget(selectedTarget)">Remover</button>
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
          <div><h2>Stacks</h2><p>{{ selectedTarget?.name || "Selecione um Target" }}</p></div>
          <button class="secondary" :disabled="busy || !selectedTargetId" @click="loadStacks">Recarregar</button>
        </div>
        <div class="table-wrap">
          <table>
            <thead><tr><th>Stack</th><th>Origem</th><th>Ações</th></tr></thead>
            <tbody>
              <tr v-for="stack in stacks" :key="stack.name">
                <td><strong>{{ stack.name }}</strong></td>
                <td>{{ stack.api_managed ? "Automation API" : "existente" }}</td>
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
        <div class="section-title"><h2>Logs</h2><button class="secondary" @click="logs = ''">Fechar</button></div>
        <pre>{{ logs }}</pre>
      </section>
    </template>
  </main>
</template>
