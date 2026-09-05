# Dockge Deploy

Ferramenta CLI multiplataforma para preparar, instalar, migrar, diagnosticar e administrar o host Linux onde o Dockge Core roda.

**Não é daemon, não é Control Plane e não substitui os Native Agents internos do Dockge.**

## Responsabilidade

```text
Windows / macOS / Linux
        │
        │ dockge-deploy
        │
        ├── SSH ───────────────► Linux / Docker / lifecycle do próprio Dockge
        │
        └── Automation API ────► stacks / logs / operações normais
```

## Princípios

- SSH somente para host lifecycle, bootstrap, instalação, migração, upgrade e recuperação;
- administração normal de stacks usa `/api/v1/automation`;
- host-key verification obrigatória;
- host desconhecido exibe fingerprint SHA-256 e só pode ser aceito explicitamente com `--accept-new-host-key`;
- host key aceita é persistida em `known_hosts`; mudança posterior é bloqueada;
- chave privada, `ssh-agent` via `SSH_AUTH_SOCK` e senha são suportados;
- passphrase só é lida de `DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE`;
- senha SSH só é lida de `DOCKGE_DEPLOY_SSH_PASSWORD`;
- token Dockge só é lido de `DOCKGE_DEPLOY_DOCKGE_TOKEN`;
- segredos não são aceitos como argumentos de CLI nem gravados pelo Deploy;
- mutações são plan-first e exigem `--apply`;
- operações API usam `Idempotency-Key`;
- redirects HTTP são bloqueados;
- HTTP remoto é bloqueado por padrão;
- migração/upgrade preservam stacks e volumes de aplicações e recriam somente Dockge;
- nunca executa `docker compose down -v`, `docker system prune -a --volumes` ou remoção de `/opt/stacks`.

## Comandos

```text
dockge-deploy host inspect
dockge-deploy doctor

dockge-deploy docker install

dockge-deploy dockge detect
dockge-deploy dockge install
dockge-deploy dockge upgrade
dockge-deploy dockge plan-migration
dockge-deploy dockge migrate
dockge-deploy dockge rollback
dockge-deploy dockge manager-token

dockge-deploy stack list
dockge-deploy stack inspect
dockge-deploy stack logs
dockge-deploy stack apply
dockge-deploy stack start
dockge-deploy stack stop
dockge-deploy stack restart
dockge-deploy stack pull
dockge-deploy stack up
dockge-deploy stack down

dockge-deploy version
```

## Primeiro vínculo SSH

A primeira conexão a um host desconhecido falha de propósito e mostra o fingerprint recebido:

```bash
dockge-deploy host inspect \
  --host 10.0.0.15 \
  --user root
```

Valide o fingerprint por um canal independente e então aceite-o uma única vez:

```bash
dockge-deploy host inspect \
  --host 10.0.0.15 \
  --user root \
  --accept-new-host-key
```

Conexões seguintes exigem exatamente a chave persistida. Host key alterada é recusada mesmo quando `--accept-new-host-key` é informado.

## Autenticação SSH

```bash
export DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE='passphrase-da-chave-criptografada'
export DOCKGE_DEPLOY_SSH_PASSWORD='senha-ssh-quando-necessaria'
```

O Deploy também usa automaticamente um agente OpenSSH compatível quando `SSH_AUTH_SOCK` estiver disponível.

## Bootstrap do Docker

Primeiro visualize o plano:

```bash
dockge-deploy docker install --host 10.0.0.15 --sudo
```

Depois execute explicitamente:

```bash
dockge-deploy docker install --host 10.0.0.15 --sudo --apply
```

O bootstrap possui adapters para famílias Debian/Ubuntu, RPM (Fedora/RHEL/Rocky/Alma e similares) e Alpine. Se Docker Engine + Compose v2 já estiverem disponíveis, a operação é idempotente e não reinstala o runtime.

## Instalação nova do Dockge

```bash
# plano
dockge-deploy dockge install \
  --host 10.0.0.15 \
  --version 1.6.1

# execução
dockge-deploy dockge install \
  --host 10.0.0.15 \
  --version 1.6.1 \
  --apply
```

Uma instalação já existente é recusada pelo fluxo de `install`. Use `upgrade` ou `migrate` conforme o caso.

## Upgrade de wkarts/dockge

```bash
# plano
dockge-deploy dockge upgrade --host 10.0.0.15 --version 1.6.1

# snapshot + imagem local de rollback + pull + recriação somente do Dockge + verificação
dockge-deploy dockge upgrade --host 10.0.0.15 --version 1.6.1 --apply
```

O backup fica em:

```text
/opt/dockge/backups/upgrade-AAAAMMDDTHHMMSSZ
```

## Migração de Dockge legado

A análise é read-only:

```bash
dockge-deploy dockge plan-migration \
  --host 10.0.0.15 \
  --path /opt/dockge \
  --stacks-path /opt/stacks \
  --version 1.6.1
```

Após revisar o inventário:

```bash
dockge-deploy dockge migrate \
  --host 10.0.0.15 \
  --path /opt/dockge \
  --stacks-path /opt/stacks \
  --version 1.6.1 \
  --apply
```

O fluxo:

```text
DISCOVER
  ↓
PLAN / INVENTORY
  ↓
SNAPSHOT compose + .env + data + imagem atual + lista de stacks
  ↓
STOP somente Dockge
  ↓
substituir configuração do orquestrador
  ↓
PULL ghcr.io/wkarts/dockge
  ↓
UP somente Dockge
  ↓
VERIFY container + lista de stacks
  ↓
COMMIT
```

Se qualquer etapa falhar depois da mutação, o Compose, `.env`, dados e instalação anterior do Dockge são restaurados. Os containers das aplicações gerenciadas continuam pertencendo ao Docker Engine e não são derrubados pela migração.

## Rollback manual

```bash
dockge-deploy dockge rollback \
  --host 10.0.0.15 \
  --backup /opt/dockge/backups/migration-AAAAMMDDTHHMMSSZ \
  --apply
```

## Criar credencial para Dockge Manager

Plano:

```bash
dockge-deploy dockge manager-token --host 10.0.0.15
```

Criação explícita:

```bash
dockge-deploy dockge manager-token \
  --host 10.0.0.15 \
  --apply
```

O segredo completo é exibido uma única vez pelo Dockge. Para rotacionar um token existente com o mesmo nome, use `--replace --apply`.

## Administração via Automation API

Defina o token somente no ambiente:

```bash
export DOCKGE_DEPLOY_DOCKGE_TOKEN='dkg_...'
```

Consultas:

```bash
dockge-deploy stack list --url https://dockge.exemplo.com

dockge-deploy stack inspect \
  --url https://dockge.exemplo.com \
  --name connect-api

dockge-deploy stack logs \
  --url https://dockge.exemplo.com \
  --name connect-api \
  --tail 500
```

Deployment de uma aplicação local:

```bash
# plano
dockge-deploy stack apply \
  --url https://dockge.exemplo.com \
  --name connect-api \
  --compose ./compose.yaml \
  --env ./.env

# execução explícita
dockge-deploy stack apply \
  --url https://dockge.exemplo.com \
  --name connect-api \
  --compose ./compose.yaml \
  --env ./.env \
  --apply
```

Operações:

```bash
dockge-deploy stack restart --url https://dockge.exemplo.com --name connect-api
# exibe plano

dockge-deploy stack restart --url https://dockge.exemplo.com --name connect-api --apply
```

Para repetir exatamente uma tentativa após perda de rede, reutilize explicitamente a mesma chave:

```bash
--idempotency-key deploy-change-20260905-001
```

Sem essa opção, uma nova chave criptograficamente aleatória é gerada para cada mutação lógica.

## Build e distribuição

```bash
cd dockge-deploy
go test ./...
go vet ./...
go build ./cmd/dockge-deploy
```

O workflow produz binários e checksums para:

- Linux amd64/arm64;
- Windows amd64/arm64;
- macOS Intel/Apple Silicon.

Os binários usam uma release independente:

```text
dockge-deploy-vX.Y.Z
```

Eles **não são anexados às releases do Dockge Core**.
