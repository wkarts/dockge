import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { resolveComposeEnv, writePrivateFileAtomic } from "./stack-file-store";

function tempStacksDir(): string {
    return fs.mkdtempSync(path.join(os.tmpdir(), "dockge-stack-files-"));
}

test("resolveComposeEnv preserves existing .env when update omits compose_env", () => {
    const root = tempStacksDir();
    try {
        const stackDir = path.join(root, "pige360-app");
        fs.mkdirSync(stackDir, { recursive: true });
        fs.writeFileSync(path.join(stackDir, ".env"), "SECRET=keep-me\nPORT=8080\n");

        assert.equal(
            resolveComposeEnv(root, "pige360-app", undefined, true),
            "SECRET=keep-me\nPORT=8080\n",
        );
    } finally {
        fs.rmSync(root, { recursive: true, force: true });
    }
});

test("resolveComposeEnv accepts explicit empty string to clear .env", () => {
    const root = tempStacksDir();
    try {
        const stackDir = path.join(root, "pige360-app");
        fs.mkdirSync(stackDir, { recursive: true });
        fs.writeFileSync(path.join(stackDir, ".env"), "SECRET=old\n");

        assert.equal(resolveComposeEnv(root, "pige360-app", "", true), "");
    } finally {
        fs.rmSync(root, { recursive: true, force: true });
    }
});

test("resolveComposeEnv defaults to empty .env for new stack", () => {
    const root = tempStacksDir();
    try {
        assert.equal(resolveComposeEnv(root, "connect-api-app", undefined, false), "");
    } finally {
        fs.rmSync(root, { recursive: true, force: true });
    }
});

test("resolveComposeEnv rejects non-string compose_env", () => {
    const root = tempStacksDir();
    try {
        assert.throws(
            () => resolveComposeEnv(root, "pige360-app", { secret: true }, true),
            /compose_env_must_be_string/,
        );
    } finally {
        fs.rmSync(root, { recursive: true, force: true });
    }
});

test("writePrivateFileAtomic replaces content without leaving temp files", () => {
    const root = tempStacksDir();
    try {
        const file = path.join(root, "pige360-app", ".env");
        writePrivateFileAtomic(file, "TOKEN=first\n");
        writePrivateFileAtomic(file, "TOKEN=second\n");

        assert.equal(fs.readFileSync(file, "utf8"), "TOKEN=second\n");
        const siblings = fs.readdirSync(path.dirname(file));
        assert.deepEqual(siblings, [".env"]);

        if (process.platform !== "win32") {
            assert.equal(fs.statSync(file).mode & 0o777, 0o600);
        }
    } finally {
        fs.rmSync(root, { recursive: true, force: true });
    }
});
