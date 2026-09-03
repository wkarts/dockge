import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

export const API_SCOPES = [
    "server:read",
    "stacks:read",
    "stacks:write",
    "stacks:delete",
    "stacks:operate",
    "stacks:adopt",
] as const;

export type ApiScope = typeof API_SCOPES[number];

export interface ApiTokenRecord {
    id?: string;
    name: string;
    sha256: string;
    scopes: ApiScope[];
    stackPrefixes?: string[];
    expiresAt?: string;
    disabled?: boolean;
    createdAt?: string;
    rotatedAt?: string;
    revokedAt?: string;
}

interface TokenFile {
    version?: number;
    tokens: ApiTokenRecord[];
}

export function tokenFilePath(): string {
    return process.env.DOCKGE_API_TOKENS_FILE || "./data/api-tokens.json";
}

export function hashToken(token: string): string {
    return crypto.createHash("sha256").update(token, "utf8").digest("hex");
}

export function generateApiToken(): string {
    return `dkg_${crypto.randomBytes(32).toString("base64url")}`;
}

export function normalizeHash(value: string): string {
    return value.trim().toLowerCase().replace(/^sha256:/, "");
}

export function readTokenFile(): TokenFile {
    const file = tokenFilePath();
    if (!fs.existsSync(file)) return { version: 1, tokens: [] };
    const parsed = JSON.parse(fs.readFileSync(file, "utf8")) as TokenFile;
    if (!Array.isArray(parsed.tokens)) throw new Error("DOCKGE API token file must contain a tokens array");
    return { version: Number(parsed.version || 1), tokens: parsed.tokens };
}

export function writeTokenFile(value: TokenFile): void {
    const file = tokenFilePath();
    fs.mkdirSync(path.dirname(file), { recursive: true });
    const temp = `${file}.${process.pid}.${crypto.randomUUID()}.tmp`;
    const body = JSON.stringify({ version: 1, tokens: value.tokens }, null, 2) + "\n";
    fs.writeFileSync(temp, body, { mode: 0o600 });
    try { fs.chmodSync(temp, 0o600); } catch { /* best effort */ }
    fs.renameSync(temp, file);
    try { fs.chmodSync(file, 0o600); } catch { /* best effort */ }
}

export function publicTokenRecord(record: ApiTokenRecord) {
    return {
        id: record.id || null,
        name: record.name,
        scopes: record.scopes || [],
        stackPrefixes: record.stackPrefixes || [],
        expiresAt: record.expiresAt || null,
        disabled: Boolean(record.disabled),
        createdAt: record.createdAt || null,
        rotatedAt: record.rotatedAt || null,
        revokedAt: record.revokedAt || null,
        fingerprint: normalizeHash(record.sha256).slice(0, 12),
    };
}
