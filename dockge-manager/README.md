# Dockge Manager

Management plane web/PWA para administrar uma ou várias instalações **Dockge Core** pela Automation REST API.

O Manager **não é um segundo orquestrador**: não monta `docker.sock`, não acessa o banco interno do Dockge e não escreve diretamente em `/opt/stacks`. O Dockge continua sendo a fonte de verdade operacional.

## Stack

- Python 3.13 + FastAPI;
- SQLAlchemy 2 + PostgreSQL + Alembic;
- Vue 3 + TypeScript + PWA responsiva;
- Docker/Compose;
- imagem multiarch no GHCR;
- autenticação local JWT;
- Fernet para segredos em repouso.

## Capacidades

### Infraestrutura

- múltiplos `DockgeTarget`;
- teste de conectividade e versão;
- listagem de stacks;
- logs;
- start/stop/restart/pull/up/down;
- operações pela `/api/v1/automation` com tokens máquina-a-máquina.

### Deployment avançado

- catálogo de `Application`;
- `Deployment` por Target/stack;
- revisões de Compose;
- `.env` de revisão criptografado;
- adoção explícita de stack externa;
- `Idempotency-Key` por mutação;
- snapshot do runtime real antes da execução;
- distinção entre revisão desejada (`current_revision`) e realmente ativa (`active_revision`);
- verificação de containers após `up`;
- rollback automático quando a mutação já ocorreu e a verificação/etapa seguinte falha;
- restauração do Compose + `.env` anterior quando a stack já existia;
- remoção apenas da stack API-managed recém-criada quando ela não existia antes do deployment;
- nenhuma remoção automática de volumes;
- rollback manual entre revisões já implantadas;
- histórico de snapshots, operações e auditoria.

Fluxo:

```text
DRAFT
  ↓
SNAPSHOT RUNTIME
  ↓
APPLY
  ↓
UP
  ↓
VERIFY
  ├── saudável ──► HEALTHY
  └── falha ─────► RESTORE SNAPSHOT ──► ROLLED_BACK
                                      └► ROLLBACK_FAILED (se a recuperação falhar)
```

Uma falha de precondição **antes da primeira mutação bem-sucedida** não dispara adoção, remoção ou rollback remoto.

## PWA

A interface inclui:

- resumo de Dockges, stacks e deployments;
- gerenciamento de Targets;
- operações de stack e logs;
- cadastro de aplicações;
- criação e execução de deployments;
- rollback;
- acompanhamento de operações e auditoria.

O modo claro é o padrão da interface.

## Inicialização

Crie `.env` a partir de `.env.example` e gere segredos fortes:

```bash
cd dockge-manager
cp .env.example .env
python - <<'PY'
import base64, os, secrets
print('DOCKGE_MANAGER_DB_PASSWORD=' + secrets.token_urlsafe(32))
print('DOCKGE_MANAGER_JWT_SECRET=' + secrets.token_urlsafe(64))
print('DOCKGE_MANAGER_FERNET_KEY=' + base64.urlsafe_b64encode(os.urandom(32)).decode())
print('DOCKGE_MANAGER_ADMIN_PASSWORD=' + secrets.token_urlsafe(32))
PY
```

Copie os valores para `.env` e execute:

```bash
docker compose pull
docker compose up -d
```

Por padrão a UI fica em:

```text
http://127.0.0.1:8010
```

Para acesso remoto, publique atrás de reverse proxy HTTPS.

## Criar a credencial Dockge

Você pode criar o token manualmente em **Settings → API Access** ou usar o Dockge Deploy:

```bash
# plano
dockge-deploy dockge manager-token --host 10.0.0.15

# cria a credencial e mostra o segredo uma única vez
dockge-deploy dockge manager-token --host 10.0.0.15 --apply
```

O perfil gerado contempla as operações técnicas usadas pelo Manager, incluindo `server:read`, leitura/escrita/operação de stacks, adoção explícita e delete necessário apenas para recuperar um deployment que criou uma stack inexistente e falhou posteriormente.

Cadastre na PWA:

```text
Nome      produção-01
URL       https://dockge.exemplo.com
Token     dkg_...
TLS       validar
```

O token é criptografado antes da persistência e não volta integralmente à UI.

## HTTP/TLS

Produção deve usar HTTPS para os Targets.

Somente laboratório/rede privada pode habilitar:

```text
DOCKGE_MANAGER_ALLOW_HTTP_TARGETS=true
```

Redirects HTTP vindos do endpoint Dockge não são seguidos automaticamente.

## Política de verificação

Defaults:

```text
DOCKGE_MANAGER_DEPLOYMENT_VERIFY_ATTEMPTS=8
DOCKGE_MANAGER_DEPLOYMENT_VERIFY_INTERVAL_SECONDS=2
```

Uma stack é considerada saudável quando existe pelo menos um container e todos os containers retornados pelo `docker compose ps` estão `running`; quando healthcheck está presente, ele deve estar `healthy`.

## Persistência

O PostgreSQL guarda metadata, desired state, revisões, operações, auditoria e snapshots.

O estado real de containers/stacks **não é autoritativo no PostgreSQL**. Ele é reconciliado com o Dockge.

Segredos persistidos:

- token de cada Dockge Target: Fernet;
- `.env` das revisões: Fernet;
- `.env` capturado em snapshot: Fernet.

A API de snapshots não devolve o Compose ou `.env` capturado; expõe apenas metadata do snapshot.

## Desenvolvimento

Backend:

```bash
cd dockge-manager
python -m venv .venv
. .venv/bin/activate
pip install -r requirements-dev.txt
PYTHONPATH=backend pytest -q backend/tests
```

Frontend:

```bash
cd dockge-manager/frontend
npm install
npm run build
```

Imagem:

```bash
docker build -f Dockerfile . -t dockge-manager:local
```

## Distribuição

O workflow publica:

```text
ghcr.io/wkarts/dockge-manager:<VERSION>
ghcr.io/wkarts/dockge-manager:latest
ghcr.io/wkarts/dockge-manager:sha-<commit>
```

A imagem é independente de `ghcr.io/wkarts/dockge` e não altera o SemVer do Dockge Core.

## Segurança e limites

- nenhum Docker socket;
- nenhum acesso direto ao banco interno do Dockge;
- nenhum acesso direto a `/opt/stacks`;
- token Dockge criptografado;
- Compose env criptografado;
- JWT expirável;
- operações mutantes idempotentes;
- ações e falhas auditadas;
- `delete` de Target remove somente cadastro/credencial do Manager;
- rollback de deployment nunca solicita remoção de volumes;
- Native Agents do Dockge não são usados nem alterados pelo Manager.
