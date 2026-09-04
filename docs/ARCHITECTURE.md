# Dockge Ecosystem Architecture

> Arquitetura de referência aprovada em 2026-09-04.

O Dockge permanece o **orquestrador Docker/Compose e a fonte de verdade operacional**. Os novos componentes de gerenciamento e implantação devem consumir contratos públicos e não alterar a estrutura interna do Dockge.

```text
                       ┌────────────────────────────┐
                       │       Dockge Manager       │
                       │ FastAPI + Vue 3 + PWA      │
                       │ Docker / GHCR              │
                       └─────────────┬──────────────┘
                                     │ HTTPS
                                     │ Automation API
              ┌──────────────────────┼──────────────────────┐
              ▼                      ▼                      ▼
         Dockge Core A          Dockge Core B          Dockge Core C
              │                      │                      │
              ▼                      ▼                      ▼
            Docker                 Docker                 Docker
              │                      │                      │
            Stacks                 Stacks                 Stacks

════════════════════════ CAMADA DE IMPLANTAÇÃO ════════════════════════

 Windows / macOS / Linux
              │
              │ Dockge Deploy
              │ SSH + Dockge API
              ▼
         Servidor Linux
              │
     Docker → Dockge Core → Stacks
```

## Quatro domínios independentes

### Dockge Core

Responsável por Docker/Compose, stacks, runtime, logs, arquivos de stack, UI, Automation API e Agents nativos.

### Dockge Manager

Management plane PWA. Mantém catálogo, ambientes, deployments, desired state, histórico, auditoria e observabilidade agregada. Não executa Docker diretamente e não monta `docker.sock`.

### Dockge Deploy

Ferramenta de host lifecycle executada a partir de Windows/macOS/Linux. Usa SSH para inventário, bootstrap, instalação, migração, upgrade e recuperação; usa Automation API quando o Dockge está operacional. Não depende de Control Plane nem de Manager.

### Dockge Native Agents

Tecnologia interna existente do próprio Dockge. Permanece preservada e não é substituída pelos novos componentes.

## Invariantes

- Manager e Deploy não acessam o banco interno do Dockge;
- Manager não executa Docker diretamente;
- SSH fica restrito a host lifecycle, bootstrap, migração e recuperação;
- administração normal de stacks usa Automation API;
- não escrever em `/opt/stacks` por fora do Dockge quando a API puder executar a operação;
- não alterar `backend/agent-*`, `backend/models/agent.ts`, handlers nativos ou `common/agent-socket.ts` para atender Manager/Deploy;
- migrações e upgrades seguem `PLAN → PREFLIGHT → SNAPSHOT → EXECUTE → VERIFY → COMMIT`, com rollback;
- nenhuma regra específica de PIGE360, Connect|API, ERP, Scheduler ou outro produto entra no Core desses componentes;
- Core, Manager e Deploy têm versionamento independente.

## Documentação detalhada

Ver `docs/ecosystem/README.md`.
