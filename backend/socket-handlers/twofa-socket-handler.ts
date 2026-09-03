import { SocketHandler } from "../socket-handler";
import { DockgeServer } from "../dockge-server";
import { R } from "redbean-node";
import { User } from "../models/user";
import { checkLogin, DockgeSocket, doubleCheckPassword } from "../util-server";
import { twoFaRateLimiter } from "../rate-limiter";
import { createTotpUri, generateTotpSecret, verifyTotp } from "../totp";

function cleanToken(value: unknown): string {
    return typeof value === "string" ? value.replace(/\s+/g, "") : "";
}

function error(callback: (value: unknown) => void, err: unknown) {
    callback({
        ok: false,
        msg: err instanceof Error ? err.message : "Unknown 2FA error",
    });
}

async function bumpAuthRevision(userID: number): Promise<void> {
    await R.exec("UPDATE `user` SET auth_revision = COALESCE(auth_revision, 1) + 1 WHERE id = ?", [ userID ]);
}

export function registerTwoFAHandlers(socket: DockgeSocket, server: DockgeServer) {
    socket.on("twoFAStatus", async (callback) => {
        try {
            checkLogin(socket);
            const user = await R.findOne("user", " id = ? AND active = 1 ", [ socket.userID ]) as User;
            callback({ ok: true, status: Boolean(user?.twofa_status) });
        } catch (err) {
            error(callback, err);
        }
    });

    socket.on("prepare2FA", async (currentPassword, callback) => {
        try {
            checkLogin(socket);
            if (!await twoFaRateLimiter.pass(callback)) return;
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            if (Boolean(user.twofa_status)) {
                throw new Error("2FA is already enabled for this user");
            }

            const secret = generateTotpSecret();
            await R.exec("UPDATE `user` SET twofa_secret = ?, twofa_last_token = NULL WHERE id = ?", [
                secret,
                user.id,
            ]);

            const issuer = process.env.DOCKGE_TOTP_ISSUER || "Dockge";
            callback({
                ok: true,
                uri: createTotpUri(secret, String(user.username), issuer),
            });
        } catch (err) {
            error(callback, err);
        }
    });

    socket.on("verifyToken", async (token, currentPassword, callback) => {
        try {
            checkLogin(socket);
            if (!await twoFaRateLimiter.pass(callback)) return;
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            if (!user.twofa_secret) throw new Error("2FA setup has not been prepared");
            callback({
                ok: true,
                valid: verifyTotp(String(user.twofa_secret), cleanToken(token)),
            });
        } catch (err) {
            error(callback, err);
        }
    });

    socket.on("save2FA", async (currentPassword, token, callback) => {
        try {
            checkLogin(socket);
            if (!await twoFaRateLimiter.pass(callback)) return;
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            const clean = cleanToken(token);
            if (!user.twofa_secret || !verifyTotp(String(user.twofa_secret), clean)) {
                throw new Error("Invalid 2FA token");
            }

            await R.exec("UPDATE `user` SET twofa_status = 1, twofa_last_token = ? WHERE id = ?", [
                clean,
                user.id,
            ]);
            await bumpAuthRevision(user.id);

            callback({
                ok: true,
                msg: "Two-factor authentication enabled. Sign in again to confirm the new security policy.",
                reauthRequired: true,
            });

            setTimeout(() => server.disconnectAllSocketClients(user.id), 250);
        } catch (err) {
            error(callback, err);
        }
    });

    socket.on("disable2FA", async (currentPassword, token, callback) => {
        try {
            checkLogin(socket);
            if (!await twoFaRateLimiter.pass(callback)) return;
            const user = await doubleCheckPassword(socket, currentPassword) as User;
            if (!Boolean(user.twofa_status) || !user.twofa_secret) {
                throw new Error("2FA is not enabled for this user");
            }

            const clean = cleanToken(token);
            if (!verifyTotp(String(user.twofa_secret), clean)) {
                throw new Error("Invalid 2FA token");
            }
            if (String(user.twofa_last_token || "") === clean) {
                throw new Error("This 2FA token was already used. Wait for the next code and try again.");
            }

            await R.exec("UPDATE `user` SET twofa_status = 0, twofa_secret = NULL, twofa_last_token = NULL WHERE id = ?", [
                user.id,
            ]);
            await bumpAuthRevision(user.id);

            callback({
                ok: true,
                msg: "Two-factor authentication disabled. Sign in again.",
                reauthRequired: true,
            });

            setTimeout(() => server.disconnectAllSocketClients(user.id), 250);
        } catch (err) {
            error(callback, err);
        }
    });
}

export class TwoFASocketHandler extends SocketHandler {
    create(socket: DockgeSocket, server: DockgeServer) {
        registerTwoFAHandlers(socket, server);
    }
}
