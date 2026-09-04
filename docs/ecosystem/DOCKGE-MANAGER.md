# Dockge Manager

## Objetivo

Aplicação PWA para administrar infraestrutura Dockge já preparada, de qualquer lugar, sem transformar o Manager em executor Docker.

## Stack proposta

```text
Backend
  Python
  FastAPI
  SQLAlchemy 2
  Alembic
  PostgreSQL
  Redis opcional

Frontend
  Vue 3
  TypeScript
  PWA

Distribuição
  Docker
  Docker Compose
  GHCR
```

## Estrutura alvo

```text
dockge-manager/
├── backend/
│   └── app/
│       ├── api/
│       ├── auth/
│       ├── targets/
│       ├── dockge/
│       ├── deployments/
│       ├── monitoring/
│       ├── audit/
│       ├── secrets/
│       └── workers/
├── frontend/
│   └── src/
│       ├── pages/
│       ├── components/
│       ├── stores/
│       └── services/
├── migrations/
├── docker/
├── compose.yaml
└── docs/
```

## Entidades principais

`Workspace`, `Environment`, `DockgeTarget`, `CredentialRef`, `Application`, `Deployment`, `DeploymentRevision`, `Operation`, `AuditEvent` e `HealthSnapshot`.

## Fonte de verdade

O Manager guarda metadata, intenção, políticas, aprovações, histórico e auditoria. O estado real vem do Dockge:

```text
running
stopped
restarting
containers
compose atual
logs
```

Uma escrita no PostgreSQL do Manager nunca significa que a operação ocorreu. O resultado deve ser reconciliado com o Dockge.

## Deployment avançado

```text
DRAFT → VALIDATING → READY → DEPLOYING → VERIFYING → HEALTHY
                                        └→ FAILED → ROLLING_BACK → ROLLED_BACK
```

O Manager cria e acompanha a intenção; o Dockge executa Docker/Compose.

## Multi-Dockge

Uma instalação do Manager deve suportar de um a muitos `DockgeTarget`, cada um com credenciais, capabilities e estado de conexão isolados.

## Mobile

Prioridades mobile: status, alertas, logs resumidos, start/stop/restart, aprovação de deployment, acompanhamento e rollback. Edição extensa de Compose permanece prioridade desktop.
