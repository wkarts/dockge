import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { User } from "../models/user";
import { checkLogin, DockgeSocket, doubleCheckPassword } from "../util-server";
import {
    API_SCOPES,
    ApiScope,
    ApiTokenRecord,
    generateApiToken,
    hashToken,
    publicTokenRecord,
    readTokenFile,
    tokenRecordID,
    writeTokenFile,
} from "../api/api-token-store";

const stackScopes = new Set<ApiScope>([
    "stacks:read",
    "stacks:write",
    "stacks:delete",
    "stacks:operate",
    "stacks:adopt",
]);

function callbackError(callback: (value: unknown) => void, err: unknown) {
    callback({
        ok: false,
        msg: err instanceof Error ? err.message : "Unknown API token error",
    });
}

function normalizeName(value: unknown): string {
    const name = typeof value === "string" ? value.trim() : "";
    if (name.length < 3 || name.length > 80 || !/^[A-Za-z0-9][A-Za-z0-9 ._:@/-]*$/.test(name)) {
        throw new Error("Token name must contain 3-80 safe characters");
    }
    return name;
}

function normalizeScopes(value: unknown): ApiScope[] {
    if (!Array.isArray(value)) throw new Error("At least one API scope is required");
    const allowed = new Set<string>(API_SCOPES);
    const scopes = [...new Set(value.map((item) => String(item)))] as ApiScope[];
    if (scopes.length === 0 || scopes.some((scope) => !allowed.has(scope))) {
        throw new Error("Invalid or empty API scopes");
    }
    return scopes;
}

function normalizePrefixes(value: unknown, scopes: ApiScope[]): string[] {
    const prefixes = Array.isArray(value)
        ? [...new Set(value.map((item) => String(item).trim()).filter(Boolean))]
        : [];
    for (const prefix of prefixes) {
        if (prefix.length > 80 || !/^[a-z0-9][a-z0-9_-]*$/.test(prefix)) {
            throw new Error(`Invalid stack namespace prefix: ${prefix}`);
        }
    }
    if (scopes.some((scope) => stackScopes.has(scope)) && prefixes.length === 0) {
        throw new Error("Stack scopes require at least one stack namespace prefix");
    }
    return prefixes;
}

function normalizeExpiry(value: unknown): string | undefined {
    if (value === undefined || value === null || value === "") return undefined;
    const timestamp = Date.parse(String(value));
    if (!Number.isFinite(timestamp) || timestamp <= Date.now()) {
        throw new Error("Token expiration must be a future date/time");
    }
    return new Date(timestamp).toISOString();
}

function managementAudit(user: User, action: string, record: ApiTokenRecord) {
    try {
        const file = process.env.DOCKGE_API_AUDIT_FILE || "./data/api-audit.jsonl";
        fs.mkdirSync(path.dirname(file), { recursive: true });
        fs.appendFileSync(file, JSON.stringify({
            at: new Date().toISOString(),
            principal: String(user.username),
            principal_type: "web-user",
            action,
            target: tokenRecordID(record),
            token_name: record.name,
            outcome: "succeeded",
        }) + "\n", { mode: 0o600 });
        try { fs.chmodSync(file, 0o600); } catch { /* best effort */ }
    } catch {
        // Token lifecycle must not leak or fail merely because audit storage is unavailable.
    }
}

function findToken(file: ReturnType<typeof readTokenFile>, id: unknown): ApiTokenRecord {
    const requestedID = String(id || "");
    const record = file.tokens.find((item) => tokenRecordID(item) === requestedID);
    if (!record) throw new Error("API token not found");
    return record;
}

function materializeStableID(record: ApiTokenRecord): void {
    if (!record.id) record.id = crypto.randomUUID();
}

export function registerApiTokenHandlers(socket: DockgeSocket) {
    socket.on("apiTokenList", async (callback) => {
        try {
            checkLogin(socket);
            const tokens = readTokenFile().tokens
                .map(publicTokenRecord)
                .sort((a, b) => String(a.name).localeCompare(String(b.name)));
            callback({ ok: true, tokens, scopes: API_SCOPES });
        } catch (err) {
            callbackError(callback, err);
        }
    });

    socket.on("apiTokenCreate", async (request, currentPassword, callback) => {
        try {
            checkLogin(socket);
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            const name = normalizeName(request?.name);
            const scopes = normalizeScopes(request?.scopes);
            const stackPrefixes = normalizePrefixes(request?.stackPrefixes, scopes);
            const expiresAt = normalizeExpiry(request?.expiresAt);
            const file = readTokenFile();

            if (file.tokens.some((record) => record.name.toLowerCase() === name.toLowerCase() && !record.disabled)) {
                throw new Error("An active API token with this name already exists");
            }

            const token = generateApiToken();
            const record: ApiTokenRecord = {
                id: crypto.randomUUID(),
                name,
                sha256: hashToken(token),
                scopes,
                stackPrefixes,
                expiresAt,
                disabled: false,
                createdAt: new Date().toISOString(),
            };
            file.tokens.push(record);
            writeTokenFile(file);
            managementAudit(user, "api-token.create", record);

            callback({
                ok: true,
                token,
                record: publicTokenRecord(record),
                oneTimeSecret: true,
            });
        } catch (err) {
            callbackError(callback, err);
        }
    });

    socket.on("apiTokenRotate", async (id, currentPassword, callback) => {
        try {
            checkLogin(socket);
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            const file = readTokenFile();
            const record = findToken(file, id);
            if (record.disabled) throw new Error("Revoked API token cannot be rotated");

            materializeStableID(record);
            const token = generateApiToken();
            record.sha256 = hashToken(token);
            record.rotatedAt = new Date().toISOString();
            writeTokenFile(file);
            managementAudit(user, "api-token.rotate", record);

            callback({
                ok: true,
                token,
                record: publicTokenRecord(record),
                oneTimeSecret: true,
            });
        } catch (err) {
            callbackError(callback, err);
        }
    });

    socket.on("apiTokenRevoke", async (id, currentPassword, callback) => {
        try {
            checkLogin(socket);
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            const file = readTokenFile();
            const record = findToken(file, id);
            materializeStableID(record);
            if (!record.disabled) {
                record.disabled = true;
                record.revokedAt = new Date().toISOString();
                writeTokenFile(file);
                managementAudit(user, "api-token.revoke", record);
            }
            callback({ ok: true, record: publicTokenRecord(record) });
        } catch (err) {
            callbackError(callback, err);
        }
    });
}
