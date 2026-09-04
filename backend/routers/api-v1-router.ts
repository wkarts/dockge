import express, { Express, Request, Response, Router as ExpressRouter } from "express";
import fs from "node:fs";
import path from "node:path";
import childProcessAsync from "promisify-child-process";
import { Router } from "../router";
import { DockgeServer } from "../dockge-server";
import { Stack } from "../stack";
import { apiAuth, apiPrincipal, assertStackAllowed, audit } from "../api/api-token-auth";
import { resolveComposeEnv, writePrivateFileAtomic } from "../api/stack-file-store";

interface StackBody {
    compose_yaml?: string;
    compose_env?: string;
    owner?: string;
    adopt?: boolean;
}

const stackNamePattern = /^[a-z0-9][a-z0-9_-]{0,127}$/;
const base = "/api/v1/automation";

function requireValidStackName(name: string): void {
    if (!stackNamePattern.test(name)) throw Object.assign(new Error("invalid stack name"), { statusCode: 400 });
}
function markerPath(server: DockgeServer, name: string): string { return path.join(server.stacksDir, name, ".dockge-managed.json"); }
function isApiManaged(server: DockgeServer, name: string): boolean { return fs.existsSync(markerPath(server, name)); }
function errorResponse(error: unknown, res: Response) {
    const err = error as Error & { statusCode?: number };
    res.status(err.statusCode || 500).json({ error: err.message || "internal_error", request_id: res.locals.requestId });
}
function trimOutput(value: string): string { return value.length > 262144 ? value.slice(-262144) : value; }

async function runCompose(server: DockgeServer, name: string, args: string[]) {
    const cwd = path.join(server.stacksDir, name);
    if (!fs.existsSync(cwd)) throw Object.assign(new Error("stack not found"), { statusCode: 404 });
    const proc = childProcessAsync.spawn("docker", ["compose", ...args], { cwd, encoding: "utf-8" });
    const result = await proc;
    const stdout = trimOutput(result.stdout?.toString() || "");
    const stderr = trimOutput(result.stderr?.toString() || "");
    if ((result as { exitCode?: number }).exitCode && (result as { exitCode?: number }).exitCode !== 0) {
        throw Object.assign(new Error(stderr || stdout || "docker compose failed"), { statusCode: 502 });
    }
    return { stdout, stderr };
}

export class ApiV1Router extends Router {
    create(_app: Express, server: DockgeServer): ExpressRouter {
        const router = express.Router();
        router.use(base, express.json({ limit: "4mb" }));

        router.get(`${base}/health`, apiAuth("server:read"), (_req, res) => {
            res.json({ ok: true, service: "dockge", api: "v1", automation: true, version: server.packageJSON.version });
        });

        router.get(`${base}/info`, apiAuth("server:read"), (_req, res) => {
            res.json({ service: "dockge", version: server.packageJSON.version, stacks_dir: server.stacksDir, api: "v1" });
        });

        router.get(`${base}/stacks`, apiAuth("stacks:read"), async (_req, res) => {
            try {
                const principal = apiPrincipal(res);
                if (!fs.existsSync(server.stacksDir)) return res.json({ stacks: [] });
                const names = fs.readdirSync(server.stacksDir, { withFileTypes: true })
                    .filter((entry) => entry.isDirectory())
                    .map((entry) => entry.name)
                    .filter((name) => stackNamePattern.test(name))
                    .filter((name) => principal.stackPrefixes.some((prefix) => name.startsWith(prefix)));
                const stacks = [];
                for (const name of names) {
                    if (!(await Stack.composeFileExists(server.stacksDir, name))) continue;
                    const stack = new Stack(server, name);
                    stacks.push({ ...(await stack.toJSON("")), api_managed: isApiManaged(server, name) });
                }
                res.json({ stacks });
            } catch (error) { errorResponse(error, res); }
        });

        router.get(`${base}/stacks/:name`, apiAuth("stacks:read"), async (req, res) => {
            try {
                requireValidStackName(req.params.name); assertStackAllowed(res, req.params.name);
                if (!(await Stack.composeFileExists(server.stacksDir, req.params.name))) return res.status(404).json({ error: "stack_not_found" });
                const stack = new Stack(server, req.params.name);
                res.json({ ...(await stack.toJSON("")), api_managed: isApiManaged(server, req.params.name) });
            } catch (error) { errorResponse(error, res); }
        });

        router.put(`${base}/stacks/:name`, apiAuth("stacks:write"), async (req: Request, res: Response) => {
            const name = req.params.name;
            try {
                requireValidStackName(name); assertStackAllowed(res, name);
                const body = req.body as StackBody;
                if (!body.compose_yaml || typeof body.compose_yaml !== "string") return res.status(400).json({ error: "compose_yaml_required" });

                const exists = await Stack.composeFileExists(server.stacksDir, name);
                if (exists && !isApiManaged(server, name)) {
                    const principal = apiPrincipal(res);
                    if (!body.adopt || !principal.scopes.has("stacks:adopt")) return res.status(409).json({ error: "external_stack_requires_explicit_adoption" });
                }

                // Em updates, compose_env ausente preserva o .env atual. Uma
                // string vazia explícita continua sendo uma solicitação válida
                // para limpar o arquivo.
                const composeEnv = resolveComposeEnv(server.stacksDir, name, body.compose_env, exists);
                const stack = new Stack(server, name, body.compose_yaml, composeEnv);
                await stack.save(!exists);
                writePrivateFileAtomic(path.join(server.stacksDir, name, ".env"), composeEnv);

                const marker = markerPath(server, name);
                let createdAt = new Date().toISOString();
                if (fs.existsSync(marker)) {
                    try { createdAt = JSON.parse(fs.readFileSync(marker, "utf8")).created_at || createdAt; } catch { /* ignore malformed historical marker */ }
                }
                writePrivateFileAtomic(marker, JSON.stringify({
                    owner: body.owner || apiPrincipal(res).name,
                    managed_by: "dockge-api-v1",
                    created_at: createdAt,
                    updated_at: new Date().toISOString(),
                }, null, 2) + "\n");

                audit(res, exists ? "stack.update" : "stack.create", name, "succeeded", { adopted: exists && !!body.adopt });
                res.status(exists ? 200 : 201).json({ ok: true, name, adopted: exists && !!body.adopt });
            } catch (error) { audit(res, "stack.apply", name, "failed"); errorResponse(error, res); }
        });

        router.delete(`${base}/stacks/:name`, apiAuth("stacks:delete"), async (req, res) => {
            const name = req.params.name;
            try {
                requireValidStackName(name); assertStackAllowed(res, name);
                if (!isApiManaged(server, name)) return res.status(409).json({ error: "external_stack_not_managed_by_api" });
                await runCompose(server, name, ["down", "--remove-orphans"]);
                await fs.promises.rm(path.join(server.stacksDir, name), { recursive: true, force: true });
                audit(res, "stack.delete", name, "succeeded");
                res.json({ ok: true, name, volumes_removed: false });
            } catch (error) { audit(res, "stack.delete", name, "failed"); errorResponse(error, res); }
        });

        router.post(`${base}/stacks/:name/actions/:action`, apiAuth("stacks:operate"), async (req, res) => {
            const name = req.params.name; const action = req.params.action;
            try {
                requireValidStackName(name); assertStackAllowed(res, name);
                if (!isApiManaged(server, name)) return res.status(409).json({ error: "external_stack_not_managed_by_api" });
                const actions: Record<string, string[]> = {
                    pull: ["pull"], up: ["up", "-d", "--remove-orphans"], down: ["down", "--remove-orphans"],
                    restart: ["restart"], start: ["start"], stop: ["stop"],
                };
                const args = actions[action];
                if (!args) return res.status(400).json({ error: "unsupported_action", supported: Object.keys(actions) });
                const output = await runCompose(server, name, args);
                audit(res, `stack.${action}`, name, "succeeded");
                res.json({ ok: true, name, action, ...output });
            } catch (error) { audit(res, `stack.${action}`, name, "failed"); errorResponse(error, res); }
        });

        router.get(`${base}/stacks/:name/ps`, apiAuth("stacks:read"), async (req, res) => {
            try {
                requireValidStackName(req.params.name); assertStackAllowed(res, req.params.name);
                if (!(await Stack.composeFileExists(server.stacksDir, req.params.name))) return res.status(404).json({ error: "stack_not_found" });
                const stack = new Stack(server, req.params.name);
                res.json({ name: req.params.name, containers: await stack.ps() });
            } catch (error) { errorResponse(error, res); }
        });

        router.get(`${base}/stacks/:name/logs`, apiAuth("stacks:read"), async (req, res) => {
            try {
                requireValidStackName(req.params.name); assertStackAllowed(res, req.params.name);
                if (!(await Stack.composeFileExists(server.stacksDir, req.params.name))) return res.status(404).json({ error: "stack_not_found" });
                const tailValue = Number(req.query.tail || 200);
                const tail = Number.isFinite(tailValue) ? Math.min(Math.max(Math.trunc(tailValue), 1), 2000) : 200;
                const output = await runCompose(server, req.params.name, ["logs", "--no-color", "--tail", String(tail)]);
                res.json({ name: req.params.name, tail, ...output });
            } catch (error) { errorResponse(error, res); }
        });

        return router;
    }
}
