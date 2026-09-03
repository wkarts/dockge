<template>
    <div class="api-access py-3">
        <div class="d-flex flex-wrap justify-content-between align-items-start gap-3 mb-4">
            <div>
                <h5 class="mb-1">API Access</h5>
                <p class="text-muted mb-0">
                    Credenciais máquina-a-máquina para Control Planes, Agents e integrações. O segredo é exibido somente no momento da criação ou rotação.
                </p>
            </div>
            <button class="btn btn-outline-secondary" type="button" :disabled="loading" @click="loadTokens">
                <font-awesome-icon icon="arrows-rotate" /> Atualizar
            </button>
        </div>

        <div v-if="oneTimeSecret" class="alert alert-warning border mb-4">
            <div class="d-flex justify-content-between align-items-start gap-3 flex-wrap">
                <div>
                    <strong>Salve esta credencial agora.</strong>
                    <div class="small mt-1">Ela não poderá ser recuperada novamente; o servidor preserva somente o hash SHA-256.</div>
                </div>
                <button type="button" class="btn btn-sm btn-dark" @click="copySecret">Copiar</button>
            </div>
            <input class="form-control font-monospace mt-3" :value="oneTimeSecret" readonly @focus="$event.target.select()">
            <button type="button" class="btn btn-sm btn-outline-secondary mt-2" @click="oneTimeSecret = ''">Ocultar segredo</button>
        </div>

        <div class="card border-0 shadow-sm mb-4">
            <div class="card-body">
                <h6 class="mb-3">Nova credencial</h6>
                <form @submit.prevent="createToken">
                    <div class="row g-3">
                        <div class="col-lg-6">
                            <label class="form-label">Nome</label>
                            <input v-model.trim="form.name" class="form-control" maxlength="80" placeholder="Ex.: PIGE360 Control Plane" required>
                        </div>
                        <div class="col-lg-6">
                            <label class="form-label">Expiração opcional</label>
                            <input v-model="form.expiresAt" type="datetime-local" class="form-control">
                        </div>
                        <div class="col-12">
                            <label class="form-label d-block">Scopes</label>
                            <div class="scope-grid">
                                <label v-for="scope in scopes" :key="scope" class="form-check scope-option">
                                    <input v-model="form.scopes" class="form-check-input" type="checkbox" :value="scope">
                                    <span class="form-check-label font-monospace">{{ scope }}</span>
                                </label>
                            </div>
                        </div>
                        <div class="col-12">
                            <label class="form-label">Namespaces/prefixos de stacks</label>
                            <input v-model.trim="form.prefixes" class="form-control font-monospace" placeholder="pige360-, mailcow-, connect-api-">
                            <div class="form-text">Obrigatório quando houver qualquer scope <code>stacks:*</code>. Separe múltiplos prefixos por vírgula.</div>
                        </div>
                        <div class="col-lg-6">
                            <label class="form-label">Senha atual</label>
                            <input v-model="form.password" type="password" autocomplete="current-password" class="form-control" required>
                        </div>
                        <div class="col-12">
                            <button class="btn btn-primary" type="submit" :disabled="saving || form.scopes.length === 0">
                                <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>
                                Criar credencial
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </div>

        <div class="card border-0 shadow-sm">
            <div class="card-body p-0">
                <div class="p-3 border-bottom d-flex justify-content-between align-items-center">
                    <h6 class="mb-0">Credenciais cadastradas</h6>
                    <span class="badge bg-secondary">{{ tokens.length }}</span>
                </div>
                <div class="table-responsive">
                    <table class="table align-middle mb-0">
                        <thead>
                            <tr>
                                <th>Nome</th>
                                <th>Status</th>
                                <th>Scopes / namespaces</th>
                                <th>Expiração</th>
                                <th>Fingerprint</th>
                                <th class="text-end">Ações</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="token in tokens" :key="token.id || token.name">
                                <td>
                                    <strong>{{ token.name }}</strong>
                                    <div class="small text-muted">{{ token.id || 'credencial legada' }}</div>
                                </td>
                                <td>
                                    <span v-if="token.disabled" class="badge bg-danger">Revogada</span>
                                    <span v-else-if="isExpired(token)" class="badge bg-warning text-dark">Expirada</span>
                                    <span v-else class="badge bg-success">Ativa</span>
                                </td>
                                <td>
                                    <div class="small font-monospace">{{ token.scopes.join(', ') }}</div>
                                    <div class="small text-muted font-monospace">{{ token.stackPrefixes.length ? token.stackPrefixes.join(', ') : 'sem namespace de stack' }}</div>
                                </td>
                                <td class="small">{{ formatDate(token.expiresAt) }}</td>
                                <td><code>{{ token.fingerprint }}</code></td>
                                <td class="text-end text-nowrap">
                                    <button class="btn btn-sm btn-outline-primary me-1" type="button" :disabled="token.disabled || !token.id" @click="openRotate(token)">Rotacionar</button>
                                    <button class="btn btn-sm btn-outline-danger" type="button" :disabled="token.disabled || !token.id" @click="openRevoke(token)">Revogar</button>
                                </td>
                            </tr>
                            <tr v-if="!loading && tokens.length === 0">
                                <td colspan="6" class="text-center text-muted py-5">Nenhuma credencial API cadastrada.</td>
                            </tr>
                            <tr v-if="loading">
                                <td colspan="6" class="text-center py-5"><span class="spinner-border spinner-border-sm me-2"></span>Carregando...</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>

        <Confirm ref="rotateConfirm" btn-style="btn-primary" yes-text="Rotacionar" no-text="Cancelar" @yes="rotateToken">
            <p>Rotacionar <strong>{{ selected?.name }}</strong> invalida imediatamente o segredo anterior.</p>
            <label class="form-label">Senha atual</label>
            <input v-model="actionPassword" type="password" autocomplete="current-password" class="form-control" required>
        </Confirm>

        <Confirm ref="revokeConfirm" btn-style="btn-danger" yes-text="Revogar" no-text="Cancelar" @yes="revokeToken">
            <p>Revogar <strong>{{ selected?.name }}</strong>? Integrações que utilizam esta credencial perderão acesso imediatamente.</p>
            <label class="form-label">Senha atual</label>
            <input v-model="actionPassword" type="password" autocomplete="current-password" class="form-control" required>
        </Confirm>
    </div>
</template>

<script>
import Confirm from "../Confirm.vue";
import { useToast } from "vue-toastification";
const toast = useToast();

export default {
    components: { Confirm },
    data() {
        return {
            loading: false,
            saving: false,
            tokens: [],
            scopes: [],
            oneTimeSecret: "",
            selected: null,
            actionPassword: "",
            form: {
                name: "",
                expiresAt: "",
                scopes: [ "server:read" ],
                prefixes: "",
                password: "",
            },
        };
    },
    mounted() {
        this.loadTokens();
    },
    methods: {
        loadTokens() {
            this.loading = true;
            this.$root.getSocket().emit("apiTokenList", (res) => {
                this.loading = false;
                if (!res.ok) {
                    toast.error(res.msg);
                    return;
                }
                this.tokens = res.tokens || [];
                this.scopes = res.scopes || [];
            });
        },
        prefixes() {
            return this.form.prefixes.split(",").map((value) => value.trim()).filter(Boolean);
        },
        createToken() {
            this.saving = true;
            const request = {
                name: this.form.name,
                scopes: this.form.scopes,
                stackPrefixes: this.prefixes(),
                expiresAt: this.form.expiresAt ? new Date(this.form.expiresAt).toISOString() : null,
            };
            this.$root.getSocket().emit("apiTokenCreate", request, this.form.password, (res) => {
                this.saving = false;
                if (!res.ok) {
                    toast.error(res.msg);
                    return;
                }
                this.oneTimeSecret = res.token;
                this.form.name = "";
                this.form.expiresAt = "";
                this.form.scopes = [ "server:read" ];
                this.form.prefixes = "";
                this.form.password = "";
                toast.success("Credencial criada. Salve o segredo exibido agora.");
                this.loadTokens();
            });
        },
        openRotate(token) {
            this.selected = token;
            this.actionPassword = "";
            this.$refs.rotateConfirm.show();
        },
        openRevoke(token) {
            this.selected = token;
            this.actionPassword = "";
            this.$refs.revokeConfirm.show();
        },
        rotateToken() {
            if (!this.selected?.id || !this.actionPassword) {
                toast.error("Informe a senha atual.");
                return;
            }
            this.$root.getSocket().emit("apiTokenRotate", this.selected.id, this.actionPassword, (res) => {
                if (!res.ok) {
                    toast.error(res.msg);
                    return;
                }
                this.oneTimeSecret = res.token;
                this.actionPassword = "";
                toast.success("Credencial rotacionada. O segredo anterior já foi invalidado.");
                this.loadTokens();
            });
        },
        revokeToken() {
            if (!this.selected?.id || !this.actionPassword) {
                toast.error("Informe a senha atual.");
                return;
            }
            this.$root.getSocket().emit("apiTokenRevoke", this.selected.id, this.actionPassword, (res) => {
                if (!res.ok) {
                    toast.error(res.msg);
                    return;
                }
                this.actionPassword = "";
                toast.success("Credencial revogada.");
                this.loadTokens();
            });
        },
        async copySecret() {
            try {
                await navigator.clipboard.writeText(this.oneTimeSecret);
                toast.success("Credencial copiada.");
            } catch {
                toast.error("Não foi possível acessar a área de transferência.");
            }
        },
        isExpired(token) {
            return token.expiresAt && new Date(token.expiresAt).getTime() <= Date.now();
        },
        formatDate(value) {
            if (!value) return "Sem expiração";
            try { return new Date(value).toLocaleString(); } catch { return value; }
        },
    },
};
</script>

<style scoped>
.api-access {
    max-width: 1180px;
}
.scope-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
    gap: 0.5rem 1rem;
}
.scope-option {
    border: 1px solid var(--bs-border-color);
    border-radius: 0.5rem;
    padding: 0.65rem 0.75rem 0.65rem 2.25rem;
}
</style>
