<template>
    <form @submit.prevent="confirmEnableTwoFA">
        <div ref="modal" class="modal fade" tabindex="-1" data-bs-backdrop="static">
            <div class="modal-dialog">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">
                            {{ $t("Setup 2FA") }}
                            <span v-if="twoFAStatus === true" class="badge bg-success ms-2">{{ $t("Active") }}</span>
                            <span v-if="twoFAStatus === false" class="badge bg-secondary ms-2">{{ $t("Inactive") }}</span>
                        </h5>
                        <button :disabled="processing" type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close" />
                    </div>

                    <div class="modal-body">
                        <p class="text-muted small">
                            Use um aplicativo autenticador compatível com TOTP. Alterações de 2FA invalidam as sessões existentes e exigem novo login.
                        </p>

                        <div class="mb-3">
                            <label for="twofa-current-password" class="form-label">{{ $t("Current Password") }}</label>
                            <input
                                id="twofa-current-password"
                                v-model="currentPassword"
                                type="password"
                                class="form-control"
                                autocomplete="current-password"
                                required
                            />
                        </div>

                        <template v-if="twoFAStatus === false">
                            <button v-if="!uri" class="btn btn-primary" type="button" :disabled="processing || !currentPassword" @click="prepare2FA">
                                {{ $t("Enable 2FA") }}
                            </button>

                            <template v-if="uri">
                                <div class="mx-auto text-center" style="width: 220px;">
                                    <vue-qrcode :key="uri" :value="uri" type="image/png" :quality="1" :color="{ light: '#ffffffff' }" />
                                    <button v-show="!showURI" type="button" class="btn btn-outline-secondary btn-sm mt-2" @click="showURI = true">
                                        {{ $t("Show URI") }}
                                    </button>
                                </div>
                                <p v-if="showURI" class="text-break mt-2 small">{{ uri }}</p>

                                <div class="mt-3">
                                    <label for="twofa-token-enable" class="form-label">{{ $t("twoFAVerifyLabel") }}</label>
                                    <div class="input-group">
                                        <input
                                            id="twofa-token-enable"
                                            v-model="token"
                                            type="text"
                                            inputmode="numeric"
                                            pattern="[0-9]*"
                                            maxlength="6"
                                            class="form-control"
                                            autocomplete="one-time-code"
                                            required
                                            @input="tokenValid = false"
                                        >
                                        <button class="btn btn-outline-primary" type="button" :disabled="processing || token.length !== 6" @click="verifyToken">
                                            {{ $t("Verify Token") }}
                                        </button>
                                    </div>
                                    <p v-show="tokenValid" class="mt-2 text-success">{{ $t("tokenValidSettingsMsg") }}</p>
                                </div>
                            </template>
                        </template>

                        <template v-else-if="twoFAStatus === true">
                            <div class="alert alert-success py-2">
                                Autenticação em dois fatores está ativa nesta conta.
                            </div>
                            <div class="mb-3">
                                <label for="twofa-token-disable" class="form-label">Código atual do autenticador</label>
                                <input
                                    id="twofa-token-disable"
                                    v-model="token"
                                    type="text"
                                    inputmode="numeric"
                                    pattern="[0-9]*"
                                    maxlength="6"
                                    class="form-control"
                                    autocomplete="one-time-code"
                                    required
                                >
                            </div>
                            <button class="btn btn-danger" type="button" :disabled="processing || !currentPassword || token.length !== 6" @click="confirmDisableTwoFA">
                                {{ $t("Disable 2FA") }}
                            </button>
                        </template>
                    </div>

                    <div v-if="uri && twoFAStatus === false" class="modal-footer">
                        <button type="submit" class="btn btn-primary" :disabled="processing || !tokenValid">
                            <span v-if="processing" class="spinner-border spinner-border-sm me-1"></span>
                            {{ $t("Save") }}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </form>

    <Confirm ref="confirmEnableTwoFA" btn-style="btn-primary" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="save2FA">
        {{ $t("confirmEnableTwoFAMsg") }}
    </Confirm>

    <Confirm ref="confirmDisableTwoFA" btn-style="btn-danger" :yes-text="$t('Yes')" :no-text="$t('No')" @yes="disable2FA">
        {{ $t("confirmDisableTwoFAMsg") }}
    </Confirm>
</template>

<script lang="ts">
import { Modal } from "bootstrap";
import Confirm from "./Confirm.vue";
import VueQrcode from "vue-qrcode";
import { useToast } from "vue-toastification";
const toast = useToast();

export default {
    components: {
        Confirm,
        VueQrcode,
    },
    data() {
        return {
            currentPassword: "",
            processing: false,
            uri: null,
            tokenValid: false,
            twoFAStatus: null,
            token: "",
            showURI: false,
        };
    },
    mounted() {
        this.modal = new Modal(this.$refs.modal);
        this.getStatus();
    },
    methods: {
        show() {
            this.currentPassword = "";
            this.token = "";
            this.uri = null;
            this.tokenValid = false;
            this.showURI = false;
            this.getStatus();
            this.modal.show();
        },

        confirmEnableTwoFA() {
            if (!this.tokenValid) return;
            this.$refs.confirmEnableTwoFA.show();
        },

        confirmDisableTwoFA() {
            this.$refs.confirmDisableTwoFA.show();
        },

        prepare2FA() {
            this.processing = true;
            this.$root.getSocket().emit("prepare2FA", this.currentPassword, (res) => {
                this.processing = false;
                if (res.ok) {
                    this.uri = res.uri;
                    this.token = "";
                    this.tokenValid = false;
                } else {
                    toast.error(res.msg);
                }
            });
        },

        save2FA() {
            this.processing = true;
            this.$root.getSocket().emit("save2FA", this.currentPassword, this.token, (res) => {
                this.processing = false;
                if (res.ok) {
                    this.$root.toastRes(res);
                    this.modal.hide();
                    this.finishSecurityChange(res);
                } else {
                    toast.error(res.msg);
                }
            });
        },

        disable2FA() {
            this.processing = true;
            this.$root.getSocket().emit("disable2FA", this.currentPassword, this.token, (res) => {
                this.processing = false;
                if (res.ok) {
                    this.$root.toastRes(res);
                    this.modal.hide();
                    this.finishSecurityChange(res);
                } else {
                    toast.error(res.msg);
                }
            });
        },

        verifyToken() {
            this.processing = true;
            this.$root.getSocket().emit("verifyToken", this.token, this.currentPassword, (res) => {
                this.processing = false;
                if (res.ok) {
                    this.tokenValid = Boolean(res.valid);
                    if (!res.valid) toast.error("Código TOTP inválido.");
                } else {
                    toast.error(res.msg);
                }
            });
        },

        getStatus() {
            this.$root.getSocket().emit("twoFAStatus", (res) => {
                if (res.ok) {
                    this.twoFAStatus = Boolean(res.status);
                } else {
                    toast.error(res.msg);
                }
            });
        },

        finishSecurityChange(res) {
            if (res.reauthRequired) {
                this.$root.storage().removeItem("token");
                window.setTimeout(() => window.location.reload(), 400);
            } else {
                this.getStatus();
            }
        },
    },
};
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.dark {
    .modal-dialog .form-text,
    .modal-dialog p {
        color: $dark-font-color;
    }
}
</style>
