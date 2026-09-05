# Roadmap de implementação

## Fase 0 — Arquitetura

- [x] separar Core, Manager, Deploy e Native Agents;
- [x] Dockge como fonte de verdade operacional;
- [x] SSH para host lifecycle e Automation API para operação normal;
- [x] encerrar Generic Infrastructure Agent 0.2.0;
- [x] documentação arquitetural aprovada e incorporada.

## Fase 1 — Proteção do Dockge Core

- [x] retirar `infrastructure-agent/` por paths explícitos;
- [x] retirar workflows/artefatos do Agent 0.2.0;
- [x] remover binários da release `v1.6.0` preservando source archives do GitHub;
- [x] gate que exige os Native Agents existentes;
- [x] impedir acoplamento Core → antigo external Agent;
- [x] manter Manager/Deploy fora da imagem do Core;
- [x] restringir release SemVer do Core ao próprio runtime.

## Fase 2 — Dockge Deploy 1.0

- [x] Go module e CLI independentes;
- [x] SSH host-key verification;
- [x] chave/ssh-agent/senha por canais seguros;
- [x] inventory Linux/Docker/Compose;
- [x] bootstrap Docker/Compose em famílias Linux principais;
- [x] `dockge detect`;
- [x] `dockge install` plan-first;
- [x] `dockge upgrade` com snapshot/rollback;
- [x] migration plan read-only;
- [x] `dockge migrate` com snapshot, verificação e rollback;
- [x] `dockge rollback`;
- [x] geração de credencial do Manager;
- [x] transporte Dockge Automation API;
- [x] administração de stacks pela API;
- [x] idempotência de mutações;
- [x] testes Go/vet;
- [x] build Linux/Windows/macOS amd64/arm64;
- [x] pipeline de release independente com checksums.

## Fase 3 — Dockge Manager 1.0

- [x] FastAPI;
- [x] PostgreSQL + Alembic;
- [x] Vue 3 + TypeScript + PWA;
- [x] autenticação JWT;
- [x] `DockgeTarget` e múltiplos Targets;
- [x] secrets criptografados;
- [x] health/info/stacks;
- [x] start/stop/restart/pull/up/down;
- [x] logs;
- [x] audit events e operations;
- [x] Docker/Compose próprio sem Docker socket;
- [x] pipeline GHCR multiarch independente;
- [x] release independente do Manager.

## Fase 4 — Deployment Engine 1.0

- [x] Application/Deployment/Revision;
- [x] validação e idempotência;
- [x] estado desejado vs revisão ativa;
- [x] snapshot do runtime real;
- [x] health verification por containers;
- [x] rollback automático após falha pós-mutação;
- [x] rollback manual entre revisões ativas;
- [x] histórico de snapshots;
- [x] auditoria e histórico de operações;
- [x] PWA para criação/acompanhamento/rollback.

## Fase 5 — Multi-Dockge e observabilidade

- [x] environments;
- [x] múltiplos Dockge Targets;
- [x] dashboard de resumo;
- [x] health snapshots;
- [x] histórico operacional/auditoria;
- [x] ações essenciais responsivas em mobile;
- [ ] tags avançadas de inventário;
- [ ] notificações/alertas externos;
- [ ] filtros analíticos avançados.

Os itens abertos da Fase 5 são evoluções pós-1.0 e não impedem o uso operacional do Manager.

## Fase 6 — Evoluções futuras

- [ ] bastion/jump-host no Dockge Deploy;
- [ ] adapters adicionais de distribuições Linux quando necessários;
- [ ] SSO/OIDC no Manager;
- [ ] RBAC granular multiusuário;
- [ ] filas/workers para deploys massivos;
- [ ] políticas de aprovação/janelas de manutenção;
- [ ] alertas Web Push/e-mail/webhooks;
- [ ] separação física em repositórios próprios caso a manutenção justifique.

## Definition of Done 1.0

A 1.0 só é liberada quando:

- Core/Native Agents permanecem intactos;
- Manager não usa Docker socket;
- SSH é limitado ao host lifecycle;
- operação normal usa Automation API;
- migração/upgrade do Dockge possuem snapshot e rollback;
- deployment do Manager consulta estado real e possui recuperação após falha pós-mutação;
- CI de Core, Manager e Deploy está verde;
- Manager image e Deploy binaries usam releases/canais independentes do Dockge Core.
