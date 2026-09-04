# Dockge Ecosystem — Implementation Status

## Estado desta entrega

A arquitetura Core / Manager / Deploy / Native Agents deixou de ser apenas blueprint e ganhou implementações independentes dentro do mesmo repositório, com toolchains, versionamento e pipelines separados. A separação física em repositórios próprios continua possível sem alterar os contratos.

### Dockge Core

- permanece em Node/Vue e continua sendo a fonte de verdade Docker/Compose;
- Automation API `/api/v1/automation` preservada;
- tokens/scopes/namespaces, auditoria e `Idempotency-Key` preservados;
- Native Agents preservados;
- release do Core não inclui binários do antigo Generic Infrastructure Agent.

### Dockge Manager `0.1.0`

Implementado em `dockge-manager/`:

- FastAPI;
- SQLAlchemy 2 + PostgreSQL + Alembic;
- Vue 3 + TypeScript + PWA;
- autenticação JWT local;
- `Workspace`, `Environment`, `CredentialRef`, `DockgeTarget`, `Application`, `Deployment`, `DeploymentRevision`, `Operation`, `AuditEvent` e `HealthSnapshot`;
- secrets criptografados com Fernet;
- múltiplos Dockge Targets;
- health polling;
- stacks, logs, start/stop/restart/pull/up/down;
- apply de Compose via API com idempotência;
- deployment por revisão e rollback;
- imagem multiarch independente `ghcr.io/wkarts/dockge-manager`.

O Manager não monta `docker.sock` e não acessa o banco interno do Dockge.

### Dockge Deploy `0.1.0`

Implementado em `dockge-deploy/`:

- CLI Go multiplataforma;
- SSH com verificação de host key e TOFU explícito apenas para host desconhecido;
- inventário/doctor;
- detecção de instalação;
- instalação em modo PLAN/APPLY;
- upgrade com snapshot, preservação da imagem atual e rollback automático;
- rollback manual de backup;
- plano de migração read-only;
- builds Linux, Windows e macOS em amd64/arm64;
- release independente `dockge-deploy-vX.Y.Z`.

## Limite deliberado

Migração destrutiva/automática entre layouts de instalação ainda não é executada diretamente. O `plan-migration` é read-only por desenho; uma futura execução de migração deve consumir um `MigrationPlan` revisado e manter snapshot/rollback verificável.
