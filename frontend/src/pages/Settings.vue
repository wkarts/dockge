<template>
    <div>
        <h1 v-show="show" class="mb-3">{{ $t("Settings") }}</h1>

        <div class="shadow-box shadow-box-settings">
            <div class="row">
                <div v-if="showSubMenu" class="settings-menu col-lg-3 col-md-5">
                    <router-link v-for="(item, key) in subMenus" :key="key" :to="`/settings/${key}`">
                        <div class="menu-item">
                            <font-awesome-icon v-if="item.icon" :icon="item.icon" class="me-2" />
                            {{ item.title }}
                        </div>
                    </router-link>

                    <a v-if="$root.isMobile && $root.loggedIn && $root.socket.token !== 'autoLogin'" class="logout" @click.prevent="$root.logout">
                        <div class="menu-item">
                            <font-awesome-icon icon="sign-out-alt" />
                            {{ $t("Logout") }}
                        </div>
                    </a>
                </div>
                <div class="settings-content col-lg-9 col-md-7">
                    <div v-if="currentPage && subMenus[currentPage]" class="settings-content-header">
                        {{ subMenus[currentPage].title }}
                    </div>
                    <div class="mx-3">
                        <router-view v-slot="{ Component }">
                            <transition name="slide-fade" appear>
                                <component :is="Component" />
                            </transition>
                        </router-view>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { useRoute } from "vue-router";

export default {
    data() {
        return {
            show: true,
            settings: {},
            settingsLoaded: false,
        };
    },

    computed: {
        currentPage() {
            const pathSplit = useRoute().path.split("/");
            const pathEnd = pathSplit[pathSplit.length - 1];
            if (!pathEnd || pathEnd === "settings") return null;
            return pathEnd;
        },

        showSubMenu() {
            return this.$root.isMobile ? !this.currentPage : true;
        },

        subMenus() {
            return {
                general: {
                    title: this.$t("general"),
                    icon: "sliders",
                },
                appearance: {
                    title: this.$t("Appearance"),
                    icon: "palette",
                },
                security: {
                    title: this.$t("Security"),
                    icon: "shield-halved",
                },
                apiAccess: {
                    title: "API Access",
                    icon: "key",
                },
                globalEnv: {
                    title: this.$t("GlobalEnv"),
                    icon: "file-code",
                },
                about: {
                    title: this.$t("About"),
                    icon: "circle-info",
                },
            };
        },
    },

    watch: {
        "$root.isMobile"() {
            this.loadGeneralPage();
        }
    },

    mounted() {
        this.loadSettings();
        this.loadGeneralPage();
    },

    methods: {
        loadGeneralPage() {
            if (!this.currentPage && !this.$root.isMobile) {
                this.$router.push("/settings/appearance");
            }
        },

        loadSettings() {
            this.$root.getSocket().emit("getSettings", (res) => {
                if (!res?.ok) {
                    this.$root.toastRes(res);
                    return;
                }
                this.settings = res.data;
                if (this.settings.checkUpdate === undefined) {
                    this.settings.checkUpdate = true;
                }
                this.settingsLoaded = true;
            });
        },

        saveSettings(callback, currentPassword) {
            const valid = this.validateSettings();
            if (!valid.success) {
                this.$root.toastError(valid.msg);
                return;
            }

            this.$root.getSocket().emit("setSettings", this.settings, currentPassword, (res) => {
                this.$root.toastRes(res);
                if (!res?.ok) return;
                this.loadSettings();
                if (callback) callback();
            });
        },

        validateSettings() {
            if (this.settings.keepDataPeriodDays < 0) {
                return {
                    success: false,
                    msg: this.$t("dataRetentionTimeError"),
                };
            }
            return { success: true, msg: "" };
        },
    }
};
</script>

<style lang="scss" scoped>
@import "../styles/vars.scss";

.shadow-box-settings {
    padding: 20px;
    min-height: calc(100vh - 155px);
}

.settings-menu {
    a {
        text-decoration: none !important;
    }

    .menu-item {
        border-radius: 10px;
        margin: 0.5em;
        padding: 0.7em 1em;
        cursor: pointer;
        border-left-width: 0;
        transition: all ease-in-out 0.1s;
    }

    .menu-item:hover {
        background: $highlight-white;

        .dark & {
            background: $dark-header-bg;
        }
    }

    .active .menu-item {
        background: $highlight-white;
        border-left: 4px solid $primary;
        border-top-left-radius: 0;
        border-bottom-left-radius: 0;

        .dark & {
            background: $dark-header-bg;
        }
    }
}

.settings-content {
    .settings-content-header {
        width: calc(100% + 20px);
        border-bottom: 1px solid #dee2e6;
        border-radius: 0 10px 0 0;
        margin-top: -20px;
        margin-right: -20px;
        padding: 12.5px 1em;
        font-size: 26px;

        .dark & {
            background: $dark-header-bg;
            border-bottom: 0;
        }

        .mobile & {
            padding: 15px 0 0 0;

            .dark & {
                background-color: transparent;
            }
        }
    }
}

.logout {
    color: $danger !important;
}
</style>
