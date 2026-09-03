import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {
    generateApiToken,
    hashToken,
    publicTokenRecord,
    readTokenFile,
    tokenRecordID,
    writeTokenFile,
} from "./api-token-store";

test("API token generation stores only a hashable one-time secret", () => {
    const token = generateApiToken();
    assert.match(token, /^dkg_[A-Za-z0-9_-]+$/);
    assert.ok(token.length >= 40);

    const digest = hashToken(token);
    assert.match(digest, /^[a-f0-9]{64}$/);
    assert.notEqual(digest, token);

    const publicRecord = publicTokenRecord({
        id: "token-id",
        name: "control-plane",
        sha256: digest,
        scopes: [ "server:read" ],
    });
    assert.equal(publicRecord.id, "token-id");
    assert.equal(publicRecord.fingerprint, digest.slice(0, 12));
    assert.equal("sha256" in publicRecord, false);
});

test("token file is persisted and read back without plaintext secret", () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "dockge-token-test-"));
    const file = path.join(root, "api-tokens.json");
    const previous = process.env.DOCKGE_API_TOKENS_FILE;
    process.env.DOCKGE_API_TOKENS_FILE = file;

    try {
        const secret = generateApiToken();
        writeTokenFile({
            version: 1,
            tokens: [
                {
                    id: "id-1",
                    name: "agent",
                    sha256: hashToken(secret),
                    scopes: [ "server:read" ],
                    createdAt: "2026-09-03T00:00:00.000Z",
                },
            ],
        });

        const raw = fs.readFileSync(file, "utf8");
        assert.equal(raw.includes(secret), false);
        assert.equal(fs.existsSync(file + ".tmp"), false);

        const loaded = readTokenFile();
        assert.equal(loaded.tokens.length, 1);
        assert.equal(loaded.tokens[0].name, "agent");
        assert.equal(loaded.tokens[0].sha256, hashToken(secret));
    } finally {
        if (previous === undefined) delete process.env.DOCKGE_API_TOKENS_FILE;
        else process.env.DOCKGE_API_TOKENS_FILE = previous;
        fs.rmSync(root, { recursive: true, force: true });
    }
});

test("legacy token receives a deterministic administrative id", () => {
    const record = {
        name: "legacy",
        sha256: "a".repeat(64),
        scopes: [ "server:read" as const ],
    };
    assert.equal(tokenRecordID(record), `legacy-${"a".repeat(24)}`);
    assert.equal(publicTokenRecord(record).legacy, true);
});
