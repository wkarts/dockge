import crypto from "node:crypto";

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

export interface VerifyTotpOptions {
    window?: number;
    period?: number;
    digits?: number;
    now?: number;
}

function encodeBase32(input: Buffer): string {
    let bits = "";
    for (const byte of input) bits += byte.toString(2).padStart(8, "0");
    let out = "";
    for (let i = 0; i < bits.length; i += 5) {
        const chunk = bits.slice(i, i + 5).padEnd(5, "0");
        out += alphabet[Number.parseInt(chunk, 2)];
    }
    return out;
}

function decodeBase32(value: string): Buffer {
    const normalized = value.toUpperCase().replace(/=+$/g, "").replace(/\s+/g, "");
    let bits = "";
    for (const char of normalized) {
        const index = alphabet.indexOf(char);
        if (index < 0) throw new Error("invalid base32 secret");
        bits += index.toString(2).padStart(5, "0");
    }
    const bytes: number[] = [];
    for (let i = 0; i + 8 <= bits.length; i += 8) {
        bytes.push(Number.parseInt(bits.slice(i, i + 8), 2));
    }
    return Buffer.from(bytes);
}

function hotp(secret: string, counter: number, digits = 6): string {
    if (!Number.isSafeInteger(counter) || counter < 0) throw new Error("invalid HOTP counter");
    const key = decodeBase32(secret);
    const msg = Buffer.alloc(8);
    msg.writeBigUInt64BE(BigInt(counter));
    const digest = crypto.createHmac("sha1", key).update(msg).digest();
    const offset = digest[digest.length - 1] & 0x0f;
    const binary = ((digest[offset] & 0x7f) << 24)
        | ((digest[offset + 1] & 0xff) << 16)
        | ((digest[offset + 2] & 0xff) << 8)
        | (digest[offset + 3] & 0xff);
    return String(binary % (10 ** digits)).padStart(digits, "0");
}

function safeEqual(left: string, right: string): boolean {
    const a = Buffer.from(left, "utf8");
    const b = Buffer.from(right, "utf8");
    return a.length === b.length && crypto.timingSafeEqual(a, b);
}

export function generateTotpSecret(bytes = 20): string {
    return encodeBase32(crypto.randomBytes(bytes));
}

export function verifyTotp(secret: string, token: string, options: VerifyTotpOptions = {}): boolean {
    const digits = options.digits ?? 6;
    const period = options.period ?? 30;
    const window = options.window ?? 1;
    const clean = String(token || "").replace(/\s+/g, "");
    if (!Number.isInteger(digits) || digits < 6 || digits > 10) return false;
    if (!Number.isFinite(period) || period <= 0 || !Number.isInteger(window) || window < 0 || window > 10) return false;
    if (!new RegExp(`^\\d{${digits}}$`).test(clean)) return false;
    const now = options.now ?? Date.now();
    const counter = Math.floor(now / 1000 / period);
    if (counter < 0) return false;
    for (let delta = -window; delta <= window; delta += 1) {
        const candidate = counter + delta;
        if (candidate < 0) continue;
        if (safeEqual(hotp(secret, candidate, digits), clean)) return true;
    }
    return false;
}

export function createTotpUri(secret: string, account: string, issuer = "Dockge"): string {
    const label = encodeURIComponent(`${issuer}:${account}`);
    const query = new URLSearchParams({
        secret,
        issuer,
        algorithm: "SHA1",
        digits: "6",
        period: "30",
    });
    return `otpauth://totp/${label}?${query.toString()}`;
}
