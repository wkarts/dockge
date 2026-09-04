# Retirada do Generic Infrastructure Agent 0.2.0

**Status:** retirado do Dockge Core.

## Motivo

A implementação histórica `infrastructure-agent/` foi criada como daemon ligado a Control Planes, com enrollment, heartbeat e desired-state. Esse modelo não representa o papel aprovado para implantação e gerenciamento do ecossistema.

## Removido do Core

- `infrastructure-agent/`;
- workflows exclusivos de build/release do Agent 0.2.0;
- contrato `control-plane-agent-api.md`;
- documentação de release centrada nessa implementação.

## Preservado

Do Dockge Core permanecem:

- Automation REST API;
- API tokens, scopes e namespaces;
- adoção explícita;
- `Idempotency-Key` persistente;
- audit log;
- gravação segura de stacks;
- Native Agents e `AgentManager` originais do Dockge.

## Release v1.6.0

Os binários/pacotes do Generic Infrastructure Agent 0.2.0 anexados à release v1.6.0 são retirados. Os arquivos de código-fonte gerados automaticamente pelo GitHub permanecem disponíveis.

Isso não altera o runtime Docker do Dockge 1.6.0 e não remove os Native Agents internos.

## Substituição arquitetural

O papel foi dividido em dois produtos independentes:

- **Dockge Deploy** — host lifecycle, SSH, instalação, migração, upgrade, recuperação e rollback;
- **Dockge Manager** — management plane PWA sobre a Automation API.

Ver `DOCKGE-DEPLOY.md`, `DOCKGE-MANAGER.md` e `BOUNDARIES-AND-CONTRACTS.md`.
