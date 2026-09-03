import { R } from "redbean-node";
import { log } from "./log";
import { LooseObject } from "../common/util-common";

export class Settings {

    static cacheList : LooseObject = {};

    static cacheCleaner? : NodeJS.Timeout;

    static async get(key : string) {
        if (!Settings.cacheCleaner) {
            Settings.cacheCleaner = setInterval(() => {
                log.debug("settings", "Cache Cleaner is just started.");
                for (key in Settings.cacheList) {
                    if (Date.now() - Settings.cacheList[key].timestamp > 60 * 1000) {
                        log.debug("settings", "Cache Cleaner deleted: " + key);
                        delete Settings.cacheList[key];
                    }
                }
            }, 60 * 1000);
        }

        if (key in Settings.cacheList) {
            const v = Settings.cacheList[key].value;
            log.debug("settings", `Get Setting (cache): ${key}: ${v}`);
            return v;
        }

        const value = await R.getCell("SELECT `value` FROM setting WHERE `key` = ? ", [ key ]);

        try {
            const v = JSON.parse(value);
            log.debug("settings", `Get Setting: ${key}: ${v}`);
            Settings.cacheList[key] = {
                value: v,
                timestamp: Date.now()
            };
            return v;
        } catch (e) {
            return value;
        }
    }

    static assertAllowed(key: string, value: unknown) {
        if (key === "disableAuth" && value === true && process.env.DOCKGE_ALLOW_DISABLE_AUTH !== "true") {
            throw new Error("Disabling authentication is blocked by deployment policy. Set DOCKGE_ALLOW_DISABLE_AUTH=true only for explicitly isolated/local installations.");
        }
    }

    static async set(key : string, value : object | string | number | boolean, type : string | null = null) {
        Settings.assertAllowed(key, value);

        let bean = await R.findOne("setting", " `key` = ? ", [ key ]);
        if (!bean) {
            bean = R.dispense("setting");
            bean.key = key;
        }
        bean.type = type;
        bean.value = JSON.stringify(value);
        await R.store(bean);

        Settings.deleteCache([ key ]);
    }

    static async getSettings(type : string) {
        const list = await R.getAll("SELECT `key`, `value` FROM setting WHERE `type` = ? ", [ type ]);
        const result : LooseObject = {};

        for (const row of list) {
            try {
                result[row.key] = JSON.parse(row.value);
            } catch (e) {
                result[row.key] = row.value;
            }
        }

        return result;
    }

    static async setSettings(type : string, data : LooseObject) {
        const keyList = Object.keys(data);
        const promiseList = [];

        for (const key of keyList) {
            Settings.assertAllowed(key, data[key]);

            let bean = await R.findOne("setting", " `key` = ? ", [ key ]);

            if (bean == null) {
                bean = R.dispense("setting");
                bean.type = type;
                bean.key = key;
            }

            if (bean.type === type) {
                bean.value = JSON.stringify(data[key]);
                promiseList.push(R.store(bean));
            }
        }

        await Promise.all(promiseList);
        Settings.deleteCache(keyList);
    }

    static deleteCache(keyList : string[]) {
        for (const key of keyList) {
            delete Settings.cacheList[key];
        }
    }

    static stopCacheCleaner() {
        if (Settings.cacheCleaner) {
            clearInterval(Settings.cacheCleaner);
            Settings.cacheCleaner = undefined;
        }
    }
}
