# Roadmap de implementação

## Fase 0 — Congelar arquitetura

- [x] separar Core, Manager, Deploy e Native Agents;
- [x] Dockge como fonte de verdade operacional;
- [x] SSH para host e Automation API para operação normal;
- [x] encerrar conceitualmente Generic Infrastructure Agent 0.2.0;
- [ ] aprovar PR documental.

## Fase 1 — Proteção do Dockge Core

- [ ] inventariar referências ao `infrastructure-agent/`;
- [ ] criar teste/gate que garanta que Native Agents não sejam removidos junto do legado;
- [ ] identificar lacunas reais da Automation API para Manager/Deploy;
- [ ] não adicionar dependência de Manager/Deploy ao runtime Dockge.

## Fase 2 — Dockge Deploy MVP

- [ ] criar repositório dedicado;
- [ ] Go module e CLI;
- [ ] SSH host-key verification;
- [ ] inventory Linux/Docker/Compose;
- [ ] `dockge detect`;
- [ ] `dockge install`;
- [ ] migration plan;
- [ ] snapshot + rollback;
- [ ] verificação via Automation API;
- [ ] testes Linux.

## Fase 3 — Dockge Manager Foundation

- [ ] criar repositório dedicado;
- [ ] FastAPI;
- [ ] PostgreSQL + Alembic;
- [ ] Vue 3 + TypeScript + PWA;
- [ ] autenticação;
- [ ] `DockgeTarget`;
- [ ] abstração de secrets;
- [ ] health/info/stacks;
- [ ] start/stop/restart;
- [ ] audit events;
- [ ] Docker/Compose/GHCR.

## Fase 4 — Deployment Engine

- [ ] Application/Deployment/Revision;
- [ ] validação;
- [ ] idempotência;
- [ ] progresso;
- [ ] health verification;
- [ ] rollback;
- [ ] histórico.

## Fase 5 — Multi-Dockge e observabilidade

- [ ] environments/tags;
- [ ] dashboard;
- [ ] health snapshots;
- [ ] alertas/filtros;
- [ ] mobile actions.

## Fase 6 — Retirada do legado 0.2.0

- [ ] remover paths explícitos;
- [ ] remover workflows exclusivos;
- [ ] corrigir docs;
- [ ] executar CI completa;
- [ ] garantir que `backend/agent-*`, handlers nativos e `common/agent-socket.ts` não mudaram.

## Definition of Done

Nenhuma fase é concluída se quebrar stack existente, exigir Docker socket remoto, contornar a Automation API sem necessidade de host lifecycle, acoplar produto de negócio, reutilizar o contrato do Native Agent ou executar migração/upgrade sem rollback.
