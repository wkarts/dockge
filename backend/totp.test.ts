import assert from "node:assert/strict";
import test from "node:test";
import { createTotpUri, generateTotpSecret, verifyTotp } from "./totp";
import { protectTotpSecret, revealTotpSecret } from "./totp-secret";

const RFC_SECRET = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ";

test("TOTP validates RFC 6238 SHA1 vector", () => {
    assert.equal(verifyTotp(RFC_SECRET, "94287082", {
        digits: 8,
        period: 30,
        window: 0,
        now: 59_000,
    }), true);

    assert.equal(verifyTotp(RFC_SECRET, "94287081", {
        digits: 8,
        period: 30,
        window: 0,
        now: 59_000,
    }), false);
});

test("generated secret and otpauth URI are usable", () => {
    const secret = generateTotpSecret();
    assert.match(secret, /^[A-Z2-7]+$/);
    assert.ok(secret.length >= 32);

    const uri = createTotpUri(secret, "admin@example.test", "Dockge Test");
    assert.ok(uri.startsWith("otpauth://totp/"));
    assert.ok(uri.includes(`secret=${secret}`));
    assert.ok(uri.includes("issuer=Dockge+Test"));
});

test("TOTP secret is encrypted at rest and decrypts with instance secret", () => {
    const encrypted = protectTotpSecret(RFC_SECRET, "instance-secret-A");
    assert.match(encrypted, /^enc:v1:/);
    assert.equal(encrypted.includes(RFC_SECRET), false);
    assert.equal(revealTotpSecret(encrypted, "instance-secret-A"), RFC_SECRET);
    assert.throws(() => revealTotpSecret(encrypted, "instance-secret-B"));
});

test("historical plaintext TOTP secret remains readable for migration", () => {
    assert.equal(revealTotpSecret(RFC_SECRET, "instance-secret-A"), RFC_SECRET);
});
