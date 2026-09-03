import { Database } from "../backend/database";
import { R } from "redbean-node";
import readline from "node:readline";
import { DockgeServer } from "../backend/dockge-server";
import { log } from "../backend/log";

console.log("╔══════════════════════════════════════════════╗");
console.log("║         Dockge · Recuperação local 2FA      ║");
console.log("╚══════════════════════════════════════════════╝");
console.log("Este comando exige acesso ao host/container e não cria um bypass remoto.\n");

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
});

const server = new DockgeServer();

function question(prompt: string): Promise<string> {
    return new Promise((resolve) => rl.question(prompt, resolve));
}

export async function main() {
    console.log("Conectando ao banco de dados...");
    try {
        await Database.init(server);
    } catch (error) {
        if (error instanceof Error) {
            log.error("security", "Falha ao abrir banco: " + error.message);
        }
        process.exitCode = 1;
        rl.close();
        return;
    }

    try {
        const user = await R.findOne("user", " active = 1 ");
        if (!user) throw new Error("Nenhum usuário ativo encontrado.");

        console.log(`Usuário encontrado: ${user.username}`);
        console.log(`2FA atual: ${Boolean(user.twofa_status) ? "ATIVO" : "inativo"}`);

        if (!Boolean(user.twofa_status) && !user.twofa_secret) {
            console.log("Nenhuma configuração 2FA precisa ser removida.");
            return;
        }

        const confirmation = await question(`Digite exatamente '${user.username}' para confirmar a recuperação: `);
        if (confirmation !== String(user.username)) {
            console.log("Confirmação não corresponde. Nenhuma alteração foi feita.");
            return;
        }

        await R.exec(
            "UPDATE `user` SET twofa_status = 0, twofa_secret = NULL, twofa_last_token = NULL, auth_revision = COALESCE(auth_revision, 1) + 1 WHERE id = ?",
            [ user.id ],
        );

        // auth_revision invalidates this user's existing JWTs without rotating
        // the instance key used to protect encrypted TOTP secrets belonging to
        // other users/future accounts.
        console.log("\n✓ 2FA removido localmente.");
        console.log("✓ Sessões existentes desta conta invalidadas.");
        console.log("Reinicie/recarregue a instância e entre novamente com usuário e senha.");
    } catch (error) {
        console.error("Erro: " + (error instanceof Error ? error.message : String(error)));
        process.exitCode = 1;
    } finally {
        await Database.close();
        rl.close();
    }
}

if (!process.env.TEST_BACKEND) {
    void main();
}
