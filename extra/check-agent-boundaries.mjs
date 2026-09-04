import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import { extname, join, relative } from "node:path";

const nativeAgentPaths = [
    "backend/agent-manager.ts",
    "backend/agent-socket-handler.ts",
    "backend/agent-socket-handlers/docker-socket-handler.ts",
    "backend/agent-socket-handlers/terminal-socket-handler.ts",
    "backend/models/agent.ts",
    "backend/socket-handlers/agent-proxy-socket-handler.ts",
    "backend/socket-handlers/manage-agent-socket-handler.ts",
    "common/agent-socket.ts",
];

const missing = nativeAgentPaths.filter((path) => !existsSync(path));
if (missing.length > 0) {
    console.error("Native Agent boundary violation: required Dockge Agent files are missing:");
    for (const path of missing) {
        console.error(` - ${path}`);
    }
    process.exit(31);
}

const scanRoots = ["backend", "common", "frontend/src"];
const sourceExtensions = new Set([".ts", ".vue"]);
const forbiddenCoupling = [
    /infrastructure-agent/i,
    /Generic Infrastructure Agent/i,
];

const violations = [];

function scan(path) {
    const stat = statSync(path);
    if (stat.isDirectory()) {
        for (const entry of readdirSync(path)) {
            scan(join(path, entry));
        }
        return;
    }

    if (!sourceExtensions.has(extname(path))) {
        return;
    }

    const content = readFileSync(path, "utf8");
    if (forbiddenCoupling.some((pattern) => pattern.test(content))) {
        violations.push(relative(process.cwd(), path));
    }
}

for (const root of scanRoots) {
    if (existsSync(root)) {
        scan(root);
    }
}

if (violations.length > 0) {
    console.error("Agent boundary violation: Dockge Core source is coupled to the retired external infrastructure-agent concept:");
    for (const path of violations) {
        console.error(` - ${path}`);
    }
    process.exit(32);
}

console.log(`Native Agent boundary OK: ${nativeAgentPaths.length} protected files present and no external-agent coupling in Core source.`);
