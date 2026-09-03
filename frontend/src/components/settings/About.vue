<template>
    <div class="d-flex justify-content-center align-items-center">
        <div class="logo d-flex flex-column justify-content-center align-items-center">
            <object class="my-4" width="200" height="200" data="/icon.svg" />
            <div class="fs-4 fw-bold">Dockge</div>
            <div class="project-edition">API-first Independent Edition</div>
            <div>{{ $t("Version") }}: {{ $root.info.version }}</div>
            <div class="frontend-version">{{ $t("Frontend Version") }}: {{ $root.frontendVersion }}</div>

            <div v-if="!$root.isFrontendBackendVersionMatched" class="alert alert-warning mt-4" role="alert">
                ⚠️ {{ $t("Frontend Version do not match backend version!") }}
            </div>

            <div class="my-3 update-link">
                <a href="https://github.com/wkarts/dockge/releases" target="_blank" rel="noopener">{{ $t("Check Update On GitHub") }}</a>
            </div>

            <div class="mt-1">
                <div class="form-check">
                    <label><input v-model="settings.checkUpdate" type="checkbox" @change="saveSettings()" /> {{ $t("Show update if available") }}</label>
                </div>
            </div>

            <div class="mt-4 project-links">
                <a href="https://github.com/wkarts/dockge" target="_blank" rel="noopener">wkarts/dockge</a>
                <span>·</span>
                <a href="https://github.com/wkarts/dockge/security/policy" target="_blank" rel="noopener">Security</a>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    computed: {
        settings() {
            return this.$parent.$parent.$parent.settings;
        },
        saveSettings() {
            return this.$parent.$parent.$parent.saveSettings;
        },
        settingsLoaded() {
            return this.$parent.$parent.$parent.settingsLoaded;
        },
    },

    watch: {

    }
};
</script>

<style lang="scss" scoped>
.logo {
    margin: 4em 1em;
}

.project-edition {
    margin-bottom: 0.5rem;
    font-size: 0.8rem;
    opacity: 0.7;
}

.update-link,
.project-links {
    font-size: 0.8em;
}

.project-links {
    display: flex;
    gap: 0.5rem;
    align-items: center;
}

.frontend-version {
    font-size: 0.9em;
    color: #777777;
}
</style>
