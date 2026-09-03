import assert from "node:assert/strict";
import test from "node:test";
import { createTotpUri, generateTotpSecret, verifyTotp } from "./totp";

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
