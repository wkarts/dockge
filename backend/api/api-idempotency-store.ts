import crypto from "node:crypto";
import fs from "node:fs";
import { writePrivateFileAtomic } from "./stack-file-store";

const MAX_COMPLETED_ENTRIES = 2048;
const MAX_KEY_LENGTH = 200;
const keyPattern = /^[A-Za-z0-9][A-Za-z0-9._:@/-]*$/;

interface StoredResponse {
    status: number;
    body: unknown;
}

interface Entry {
    id: string;
    fingerprint: string;
    state: "started" | "completed";
    createdAt: string;
    updatedAt: string;
    response?: StoredResponse;
}

interface FileFormat {
    version: number;
    entries: Entry[];
}

export interface IdempotencySession {
    id: string;
    fingerprint: string;
}

export type BeginIdempotencyResult =
    | { mode: "disabled" }
    | { mode: "execute"; session: IdempotencySession }
    | { mode: "replay"; response: StoredResponse }
    | { mode: "conflict" }
    | { mode: "in_doubt" };

function storePath(): string {
    return process.env.DOCKGE_API_IDEMPOTENCY_FILE || "./data/api-idempotency.json";
}

function hash(value: string): string {
    return crypto.createHash("sha256").update(value, "utf8").digest("hex");
}

function readStore(): FileFormat {
    const file = storePath();
    if (!fs.existsSync(file)) return { version: 1, entries: [] };
    const parsed = JSON.parse(fs.readFileSync(file, "utf8")) as Partial<FileFormat>;
    if (!Array.isArray(parsed.entries)) throw new Error("Dockge API idempotency store is invalid");
    return { version: 1, entries: parsed.entries as Entry[] };
}

function writeStore(value: FileFormat): void {
    // Nunca podar uma reserva sem resultado: removê-la permitiria repetir uma
    // mutação cujo desfecho pode ser desconhecido. Somente resultados já
    // concluídos são compactados para limitar crescimento do arquivo.
    const started = value.entries
        .filter((entry) => entry.state === "started")
        .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
    const completed = value.entries
        .filter((entry) => entry.state === "completed")
        .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt))
        .slice(0, MAX_COMPLETED_ENTRIES);
    const entries = [...started, ...completed];

    writePrivateFileAtomic(storePath(), JSON.stringify({ version: 1, entries }, null, 2) + "\n");
}

function normalizeKey(value: string | undefined): string | undefined {
    if (value === undefined) return undefined;
    const key = value.trim();
    if (key.length === 0 || key.length > MAX_KEY_LENGTH || !keyPattern.test(key)) {
        throw Object.assign(new Error("invalid_idempotency_key"), { statusCode: 400 });
    }
    return key;
}

function requestFingerprint(method: string, target: string, body: unknown): string {
    return hash(`${method.toUpperCase()}\n${target}\n${JSON.stringify(body ?? null)}`);
}

/**
 * Reserva a chave antes da mutação. Se o processo cair depois da reserva e
 * antes de registrar a resposta, a próxima tentativa retorna in_doubt e não
 * repete automaticamente a operação. Isso privilegia segurança operacional
 * sobre uma duplicação potencial de up/restart/delete.
 */
export function beginIdempotentMutation(
    principalID: string,
    keyValue: string | undefined,
    method: string,
    target: string,
    body: unknown,
): BeginIdempotencyResult {
    const key = normalizeKey(keyValue);
    if (!key) return { mode: "disabled" };

    const id = hash(`${principalID}\u0000${key}`);
    const fingerprint = requestFingerprint(method, target, body);
    const store = readStore();
    const existing = store.entries.find((entry) => entry.id === id);

    if (existing) {
        if (existing.fingerprint !== fingerprint) return { mode: "conflict" };
        if (existing.state === "completed" && existing.response) {
            return { mode: "replay", response: existing.response };
        }
        return { mode: "in_doubt" };
    }

    const now = new Date().toISOString();
    store.entries.push({
        id,
        fingerprint,
        state: "started",
        createdAt: now,
        updatedAt: now,
    });
    writeStore(store);
    return { mode: "execute", session: { id, fingerprint } };
}

export function completeIdempotentMutation(session: IdempotencySession | undefined, status: number, body: unknown): void {
    if (!session) return;
    const store = readStore();
    const entry = store.entries.find((item) => item.id === session.id);
    if (!entry || entry.fingerprint !== session.fingerprint) {
        throw new Error("idempotency session no longer matches persisted reservation");
    }
    entry.state = "completed";
    entry.updatedAt = new Date().toISOString();
    entry.response = { status, body };
    writeStore(store);
}
