import crypto from "node:crypto";
import {
    API_SCOPES,
    ApiScope,
    generateApiToken,
    hashToken,
    publicTokenRecord,
    readTokenFile,
    writeTokenFile,
} from "../backend/api/api-token-store";

function parseArgs(values: string[]): Record<string, string | boolean> {
    const out: Record<string, string | boolean> = {};
    for (let i = 0; i < values.length; i += 1) {
        const value = values[i];
        if (!value.startsWith("--")) continue;
        const key = value.slice(2);
        const next = values[i + 1];
        if (!next || next.startsWith("--")) {
            out[key] = true;
        } else {
            out[key] = next;
            i += 1;
        }
    }
    return out;
}

function csv(value: string | boolean | undefined): string[] {
    if (typeof value !== "string") return [];
    return [...new Set(value.split(",").map((item) => item.trim()).filter(Boolean))];
}

const args = parseArgs(process.argv.slice(2));
const name = typeof args.name === "string" ? args.name.trim() : "automation";
if (name.length < 3 || name.length > 80 || !/^[A-Za-z0-9][A-Za-z0-9 ._:@/-]*$/.test(name)) {
    throw new Error("--name must contain 3-80 safe characters");
}

const requestedScopes = csv(args.scopes);
const scopes = (requestedScopes.length ? requestedScopes : [
    "server:read",
    "stacks:read",
    "stacks:write",
    "stacks:operate",
]) as ApiScope[];
const allowedScopes = new Set<string>(API_SCOPES);
if (scopes.some((scope) => !allowedScopes.has(scope))) {
    throw new Error(`invalid scope; allowed: ${API_SCOPES.join(",")}`);
}

const stackPrefixes = csv(args.prefixes);
if (scopes.some((scope) => scope.startsWith("stacks:")) && stackPrefixes.length === 0) {
    throw new Error("stack scopes require --prefixes");
}
for (const prefix of stackPrefixes) {
    if (prefix.length > 80 || !/^[a-z0-9][a-z0-9_-]*$/.test(prefix)) {
        throw new Error(`invalid stack prefix: ${prefix}`);
    }
}

let expiresAt: string | undefined;
if (typeof args.expires === "string" && args.expires.trim() !== "") {
    const timestamp = Date.parse(args.expires);
    if (!Number.isFinite(timestamp) || timestamp <= Date.now()) throw new Error("--expires must be a future date/time");
    expiresAt = new Date(timestamp).toISOString();
}

const file = readTokenFile();
const activeSameName = file.tokens.filter((record) => record.name.toLowerCase() === name.toLowerCase() && !record.disabled);
const replace = args.replace === true;
if (activeSameName.length > 0 && !replace) {
    throw new Error(`active token name already exists: ${name}; use --replace only when intentional rotation is required`);
}

const now = new Date().toISOString();
if (replace) {
    for (const record of activeSameName) {
        record.disabled = true;
        record.revokedAt = now;
    }
}

const token = generateApiToken();
const record = {
    id: crypto.randomUUID(),
    name,
    sha256: hashToken(token),
    scopes,
    stackPrefixes,
    expiresAt,
    disabled: false,
    createdAt: now,
};
file.tokens.push(record);
writeTokenFile(file);

if (args["token-only"] === true) {
    process.stdout.write(token + "\n");
} else if (args.json === true) {
    process.stdout.write(JSON.stringify({
        token,
        record: publicTokenRecord(record),
        replaced: activeSameName.map(publicTokenRecord),
        oneTimeSecret: true,
    }, null, 2) + "\n");
} else {
    console.log(replace && activeSameName.length > 0 ? "Dockge API credential rotated." : "Dockge API credential created.");
    console.log("Copy this value now; only its SHA-256 is persisted:");
    console.log(token);
    console.log(`Name: ${name}`);
    console.log(`Scopes: ${scopes.join(",")}`);
    console.log(`Prefixes: ${stackPrefixes.join(",") || "(none)"}`);
    console.log(`Expires: ${expiresAt || "never"}`);
    if (activeSameName.length > 0) console.log(`Replaced credentials: ${activeSameName.length}`);
}
