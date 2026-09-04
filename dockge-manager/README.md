# Dockge Manager

Management plane web/PWA para administrar uma ou várias instalações **Dockge Core** por meio da Automation REST API. O Manager não monta `docker.sock`, não acessa o banco interno do Dockge e não escreve diretamente em `/opt/stacks`.

## Componentes

- FastAPI;
- SQLAlchemy 2 + PostgreSQL + Alembic;
- Vue 3 + TypeScript + PWA;
- credenciais Dockge criptografadas em repouso com Fernet;
- autenticação local por JWT;
- cadastro e teste de múltiplos `DockgeTarget`;
- listagem de stacks, logs e ações start/stop/restart/pull/up/down;
- apply de Compose via Automation API com `Idempotency-Key`;
- catálogo `Application` e engine inicial de `Deployment`/`DeploymentRevision` com deploy, verificação e rollback de revisão;
- auditoria e registro de operações.

## Inicialização

Crie um `.env` a partir de `.env.example` e gere segredos fortes:

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

Copie os valores gerados para `.env` e execute:

```bash
docker compose pull
docker compose up -d
```

Por padrão a UI fica em `http://127.0.0.1:8010`. Para acesso remoto, publique o serviço atrás de reverse proxy HTTPS.

## Cadastrar um Dockge

No Dockge Core, crie um token em **Settings → API Access** com scopes compatíveis com as ações desejadas. Cadastre no Manager a URL HTTPS e o token. O segredo é criptografado antes de ser persistido no PostgreSQL e não volta integralmente para a UI.

Para laboratório em rede privada sem TLS, `DOCKGE_MANAGER_ALLOW_HTTP_TARGETS=true` libera URLs HTTP explicitamente. Produção deve usar HTTPS.

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

## Segurança

- nenhum Docker socket no Manager;
- token do Dockge criptografado com Fernet;
- compose env de revisões criptografado;
- JWT com expiração;
- operações mutantes usam idempotência do Dockge;
- ações e falhas são auditadas;
- `delete` de Target remove somente o cadastro do Manager, nunca stacks do Dockge.
