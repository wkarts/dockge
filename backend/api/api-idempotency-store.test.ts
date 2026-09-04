import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { beginIdempotentMutation, completeIdempotentMutation } from "./api-idempotency-store";

function withStore(t: test.TestContext): string {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "dockge-idempotency-"));
    const file = path.join(root, "api-idempotency.json");
    process.env.DOCKGE_API_IDEMPOTENCY_FILE = file;
    t.after(() => {
        delete process.env.DOCKGE_API_IDEMPOTENCY_FILE;
        fs.rmSync(root, { recursive: true, force: true });
    });
    return file;
}

test("mutation without Idempotency-Key remains backward compatible", (t) => {
    withStore(t);
    assert.deepEqual(
        beginIdempotentMutation("token-1", undefined, "POST", "/stack/a/up", {}),
        { mode: "disabled" },
    );
});

test("completed mutation is replayed without a second execution", (t) => {
    withStore(t);
    const first = beginIdempotentMutation("token-1", "act-001", "POST", "/stack/a/up", {});
    assert.equal(first.mode, "execute");
    if (first.mode !== "execute") return;

    completeIdempotentMutation(first.session, 200, { ok: true, name: "a" });

    const replay = beginIdempotentMutation("token-1", "act-001", "POST", "/stack/a/up", {});
    assert.equal(replay.mode, "replay");
    if (replay.mode === "replay") {
        assert.equal(replay.response.status, 200);
        assert.deepEqual(replay.response.body, { ok: true, name: "a" });
    }
});

test("reserved mutation is fail-closed after an indeterminate crash window", (t) => {
    withStore(t);
    const first = beginIdempotentMutation("token-1", "act-002", "DELETE", "/stack/a", null);
    assert.equal(first.mode, "execute");

    const retry = beginIdempotentMutation("token-1", "act-002", "DELETE", "/stack/a", null);
    assert.deepEqual(retry, { mode: "in_doubt" });
});

test("same key cannot be reused for another mutation", (t) => {
    withStore(t);
    const first = beginIdempotentMutation("token-1", "act-003", "POST", "/stack/a/up", {});
    assert.equal(first.mode, "execute");

    const conflict = beginIdempotentMutation("token-1", "act-003", "POST", "/stack/a/restart", {});
    assert.deepEqual(conflict, { mode: "conflict" });
});

test("idempotency namespace is isolated by API principal", (t) => {
    withStore(t);
    const first = beginIdempotentMutation("token-pige", "act-001", "POST", "/stack/a/up", {});
    const second = beginIdempotentMutation("token-connect", "act-001", "POST", "/stack/a/up", {});
    assert.equal(first.mode, "execute");
    assert.equal(second.mode, "execute");
});

test("store does not persist plaintext principal or action key", (t) => {
    const file = withStore(t);
    const first = beginIdempotentMutation("sensitive-principal", "act-secret-001", "POST", "/stack/a/up", {});
    assert.equal(first.mode, "execute");

    const raw = fs.readFileSync(file, "utf8");
    assert.equal(raw.includes("sensitive-principal"), false);
    assert.equal(raw.includes("act-secret-001"), false);
});

test("invalid idempotency key is rejected", (t) => {
    withStore(t);
    assert.throws(
        () => beginIdempotentMutation("token-1", "bad key with spaces", "POST", "/stack/a/up", {}),
        /invalid_idempotency_key/,
    );
});
