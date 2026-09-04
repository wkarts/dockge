import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { NextFunction, Request, Response } from "express";
import {
    ApiScope,
    ApiTokenRecord,
    hashToken,
    normalizeHash,
    readTokenFile,
    tokenFilePath,
    tokenRecordID,
} from "./api-token-store";

export type { ApiScope, ApiTokenRecord } from "./api-token-store";
export { hashToken, tokenFilePath } from "./api-token-store";

export interface ApiPrincipal {
    id: string;
    name: string;
    scopes: Set<ApiScope>;
    stackPrefixes: string[];
}

function safeHashEqual(a: string, b: string): boolean {
    const left = Buffer.from(normalizeHash(a), "hex");
    const right = Buffer.from(normalizeHash(b), "hex");
    return left.length === 32 && right.length === 32 && crypto.timingSafeEqual(left, right);
}

function usable(record: ApiTokenRecord): boolean {
    if (record.disabled) return false;
    if (!record.expiresAt) return true;
    const expires = Date.parse(record.expiresAt);
    return Number.isFinite(expires) && expires > Date.now();
}

export function apiAuth(requiredScope?: ApiScope) {
    return (req: Request, res: Response, next: NextFunction) => {
        try {
            const header = req.header("authorization") || "";
            const match = header.match(/^Bearer\s+(.+)$/i);
            if (!match) return res.status(401).json({ error: "missing_bearer_token" });

            const candidateHash = hashToken(match[1]);
            const record = readTokenFile().tokens.find((item) => usable(item) && safeHashEqual(item.sha256, candidateHash));
            if (!record) return res.status(401).json({ error: "invalid_or_expired_token" });

            const principal: ApiPrincipal = {
                id: tokenRecordID(record),
                name: record.name,
                scopes: new Set(record.scopes || []),
                stackPrefixes: [...new Set(record.stackPrefixes || [])],
            };
            if (requiredScope && !principal.scopes.has(requiredScope)) {
                return res.status(403).json({ error: "insufficient_scope", required_scope: requiredScope });
            }
            res.locals.apiPrincipal = principal;
            res.locals.requestId = req.header("x-request-id") || crypto.randomUUID();
            res.setHeader("x-request-id", res.locals.requestId);
            next();
        } catch (error) {
            next(error);
        }
    };
}

export function apiPrincipal(res: Response): ApiPrincipal {
    const principal = res.locals.apiPrincipal as ApiPrincipal | undefined;
    if (!principal) throw new Error("API principal not initialized");
    return principal;
}

export function assertStackAllowed(res: Response, stackName: string): void {
    const principal = apiPrincipal(res);
    if (principal.stackPrefixes.length === 0) {
        throw Object.assign(new Error("token has no stack namespace"), { statusCode: 403 });
    }
    if (!principal.stackPrefixes.some((prefix) => stackName.startsWith(prefix))) {
        throw Object.assign(new Error("stack is outside token namespace"), { statusCode: 403 });
    }
}

export function audit(res: Response, action: string, target: string, outcome: string, details: Record<string, unknown> = {}): void {
    try {
        const principal = apiPrincipal(res);
        const file = process.env.DOCKGE_API_AUDIT_FILE || "./data/api-audit.jsonl";
        fs.mkdirSync(path.dirname(file), { recursive: true });
        fs.appendFileSync(file, JSON.stringify({
            at: new Date().toISOString(),
            request_id: res.locals.requestId,
            principal: principal.name,
            principal_id: principal.id,
            action,
            target,
            outcome,
            ...details,
        }) + "\n", { mode: 0o600 });
        try { fs.chmodSync(file, 0o600); } catch { /* best effort */ }
    } catch {
        // Audit failure must not disclose credentials or crash a successful action.
    }
}
