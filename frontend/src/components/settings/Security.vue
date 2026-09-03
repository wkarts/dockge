<template>
    <div>
        <div v-if="settingsLoaded" class="my-4">
            <template v-if="!settings.disableAuth">
                <p>
                    {{ $t("Current User") }}: <strong>{{ $root.username }}</strong>
                    <button id="logout-btn" class="btn btn-outline-danger ms-4 me-2 mb-2" @click="$root.logout">{{ $t("Logout") }}</button>
                </p>

                <h5 class="my-4 settings-subheading">{{ $t("Change Password") }}</h5>
                <form class="mb-3" @submit.prevent="savePassword">
                    <div class="mb-3">
                        <label for="current-password" class="form-label">{{ $t("Current Password") }}</label>
                        <input
                            id="current-password"
                            v-model="password.currentPassword"
                            type="password"
                            class="form-control"
                            autocomplete="current-password"
                            required
                        />
                    </div>

                    <div class="mb-3">
                        <label for="new-password" class="form-label">{{ $t("New Password") }}</label>
                        <input
                            id="new-password"
                            v-model="password.newPassword"
                            type="password"
                            class="form-control"
                            autocomplete="new-password"
                            required
                        />
                    </div>

                    <div class="mb-3">
                        <label for="repeat-new-password" class="form-label">{{ $t("Repeat New Password") }}</label>
                        <input
                            id="repeat-new-password"
                            v-model="password.repeatNewPassword"
                            type="password"
                            class="form-control"
                            :class="{ 'is-invalid': invalidPassword }"
                            autocomplete="new-password"
                            required
                        />
                        <div class="invalid-feedback">{{ $t("passwordNotMatchMsg") }}</div>
                    </div>

                    <button class="btn btn-primary" type="submit">{{ $t("Update Password") }}</button>
                </form>

                <div class="mt-5 mb-3">
                    <h5 class="my-3 settings-subheading">{{ $t("Two Factor Authentication") }}</h5>
                    <p class="text-muted">
                        Proteja o acesso humano ao Dockge com TOTP. O Agent e as APIs usam credenciais máquina-a-máquina próprias e não dependem do código TOTP humano.
                    </p>
                    <button class="btn btn-primary me-2" type="button" @click="$refs.TwoFADialog.show()">
                        {{ $t("2FA Settings") }}
                    </button>
                </div>
            </template>

            <div v-else class="alert alert-danger">
                <h5>Autenticação web desativada</h5>
                <p class="mb-3">
                    Este estado é legado e não é recomendado para uma plataforma de administração de infraestrutura. Reative a autenticação para habilitar 2FA e proteção de sessão.
                </p>
                <button id="enableAuth-btn" class="btn btn-primary" @click="enableAuth">{{ $t("Enable Auth") }}</button>
            </div>

            <div class="my-5">
                <h5 class="my-3 settings-subheading">Segurança da sessão</h5>
                <div class="alert alert-light border">
                    Senha e alterações de 2FA invalidam sessões anteriores. Tokens web possuem validade limitada e são vinculados à revisão de segurança da conta.
                </div>
            </div>
        </div>

        <TwoFADialog ref="TwoFADialog" />
    </div>
</template>

<script>
import TwoFADialog from "../../components/TwoFADialog.vue";

export default {
    components: {
        TwoFADialog
    },

    data() {
        return {
            invalidPassword: false,
            password: {
                currentPassword: "",
                newPassword: "",
                repeatNewPassword: "",
            }
        };
    },

    computed: {
        settings() {
            return this.$parent.$parent.$parent.settings;
        },
        saveSettings() {
            return this.$parent.$parent.$parent.saveSettings;
        },
        settingsLoaded() {
            return this.$parent.$parent.$parent.settingsLoaded;
        }
    },

    watch: {
        "password.repeatNewPassword"() {
            this.invalidPassword = false;
        },
    },

    methods: {
        savePassword() {
            if (this.password.newPassword !== this.password.repeatNewPassword) {
                this.invalidPassword = true;
                return;
            }

            this.$root.getSocket().emit("changePassword", this.password, (res) => {
                this.$root.toastRes(res);
                if (res.ok) {
                    this.password.currentPassword = "";
                    this.password.newPassword = "";
                    this.password.repeatNewPassword = "";
                    if (res.reauthRequired) {
                        this.$root.storage().removeItem("token");
                        window.setTimeout(() => window.location.reload(), 400);
                    }
                }
            });
        },

        enableAuth() {
            this.settings.disableAuth = false;
            this.saveSettings();
            this.$root.storage().removeItem("token");
            window.location.reload();
        },
    },
};
</script>
