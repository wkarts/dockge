import { log } from "./log";
import compareVersions from "compare-versions";
import packageJSON from "../package.json";
import { Settings } from "./settings";

// How much time in ms to wait between update checks
const UPDATE_CHECKER_INTERVAL_MS = 1000 * 60 * 60 * 48;
const CHECK_URL = process.env.DOCKGE_UPDATE_CHECK_URL || "https://api.github.com/repos/wkarts/dockge/releases/latest";

class CheckVersion {
    version = packageJSON.version;
    latestVersion? : string;
    interval? : NodeJS.Timeout;

    async startInterval() {
        const check = async () => {
            if (await Settings.get("checkUpdate") === false) {
                return;
            }

            log.debug("update-checker", "Retrieving latest stable release");

            try {
                const res = await fetch(CHECK_URL, {
                    headers: {
                        Accept: "application/vnd.github+json",
                        "User-Agent": "wkarts-dockge-update-checker",
                    },
                });

                // A repository without releases is a valid state during bootstrap.
                if (res.status === 404) {
                    log.debug("update-checker", "No stable release published yet");
                    return;
                }

                if (!res.ok) {
                    throw new Error(`Update check returned HTTP ${res.status}`);
                }

                const data = await res.json() as { tag_name?: string };
                const latest = data.tag_name?.replace(/^v/, "");

                if (!latest) {
                    return;
                }

                // For debug
                if (process.env.TEST_CHECK_VERSION === "1") {
                    this.latestVersion = "1000.0.0";
                    return;
                }

                if (compareVersions.validate(latest)) {
                    this.latestVersion = latest;
                }
            } catch (error) {
                log.info("update-checker", "Failed to check for new versions");
                log.debug("update-checker", error);
            }
        };

        await check();
        this.interval = setInterval(check, UPDATE_CHECKER_INTERVAL_MS);
    }
}

const checkVersion = new CheckVersion();
export default checkVersion;
