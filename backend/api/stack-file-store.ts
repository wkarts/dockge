import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

/**
 * Resolve o conteúdo de .env para um apply de stack.
 *
 * Em atualizações, a ausência de compose_env significa "preservar o arquivo
 * existente". Uma string vazia explícita continua significando "limpar .env".
 */
export function resolveComposeEnv(stacksDir: string, name: string, provided: unknown, stackExists: boolean): string {
    if (provided !== undefined) {
        if (typeof provided !== "string") {
            throw Object.assign(new Error("compose_env_must_be_string"), { statusCode: 400 });
        }
        return provided;
    }

    if (!stackExists) {
        return "";
    }

    try {
        return fs.readFileSync(path.join(stacksDir, name, ".env"), "utf8");
    } catch (error) {
        const code = (error as NodeJS.ErrnoException).code;
        if (code === "ENOENT") {
            return "";
        }
        throw error;
    }
}

/**
 * Substitui um arquivo privado atomicamente e força permissão 0600 onde o SO
 * oferece semântica POSIX. Evita truncamento parcial de .env/metadata quando o
 * processo ou o host falha durante a gravação.
 */
export function writePrivateFileAtomic(file: string, content: string): void {
    fs.mkdirSync(path.dirname(file), { recursive: true });
    const temp = `${file}.${process.pid}.${crypto.randomUUID()}.tmp`;

    try {
        fs.writeFileSync(temp, content, { mode: 0o600 });
        try { fs.chmodSync(temp, 0o600); } catch { /* best effort on non-POSIX */ }
        fs.renameSync(temp, file);
        try { fs.chmodSync(file, 0o600); } catch { /* best effort on non-POSIX */ }
    } catch (error) {
        try { fs.rmSync(temp, { force: true }); } catch { /* best effort cleanup */ }
        throw error;
    }
}
