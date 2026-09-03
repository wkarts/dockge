import crypto from "node:crypto";

const prefix = "enc:v1:";

function keyFromInstanceSecret(instanceSecret: string): Buffer {
    if (!instanceSecret) throw new Error("instance secret is required to protect TOTP secret");
    return crypto.createHash("sha256").update(instanceSecret, "utf8").digest();
}

export function protectTotpSecret(secret: string, instanceSecret: string): string {
    const iv = crypto.randomBytes(12);
    const cipher = crypto.createCipheriv("aes-256-gcm", keyFromInstanceSecret(instanceSecret), iv);
    const encrypted = Buffer.concat([
        cipher.update(secret, "utf8"),
        cipher.final(),
    ]);
    const tag = cipher.getAuthTag();
    return [
        "enc",
        "v1",
        iv.toString("base64url"),
        tag.toString("base64url"),
        encrypted.toString("base64url"),
    ].join(":");
}

export function revealTotpSecret(stored: string, instanceSecret: string): string {
    if (!stored.startsWith(prefix)) {
        // Compatibility with secrets persisted by the historical code path.
        return stored;
    }

    const parts = stored.split(":");
    if (parts.length !== 5 || parts[0] !== "enc" || parts[1] !== "v1") {
        throw new Error("unsupported encrypted TOTP secret format");
    }

    const iv = Buffer.from(parts[2], "base64url");
    const tag = Buffer.from(parts[3], "base64url");
    const encrypted = Buffer.from(parts[4], "base64url");
    if (iv.length !== 12 || tag.length !== 16 || encrypted.length === 0) {
        throw new Error("invalid encrypted TOTP secret");
    }

    const decipher = crypto.createDecipheriv("aes-256-gcm", keyFromInstanceSecret(instanceSecret), iv);
    decipher.setAuthTag(tag);
    return Buffer.concat([
        decipher.update(encrypted),
        decipher.final(),
    ]).toString("utf8");
}
