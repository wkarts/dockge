# Decisões arquiteturais

## ADR-0001 — Separar Core, Manager e Deploy

**Status:** Aceito — 2026-09-04

Dockge Core, Dockge Manager e Dockge Deploy têm responsabilidades e ciclos de vida independentes. A integração ocorre por contratos públicos, principalmente Automation REST API.

## ADR-0002 — Dockge é a fonte de verdade operacional

**Status:** Aceito — 2026-09-04

O Manager guarda intenção, metadata e histórico, mas o estado real de stack/container é lido do Dockge. Uma atualização no banco do Manager nunca é prova de sucesso operacional.

## ADR-0003 — Manager como FastAPI + Vue PWA

**Status:** Aceito — 2026-09-04

O Dockge Manager será aplicação web/PWA responsiva, distribuível em Docker/GHCR, permitindo desktop e mobile sem replicar clientes nativos.

## ADR-0004 — Deploy usa SSH + Dockge API

**Status:** Aceito — 2026-09-04

SSH atende host lifecycle: bootstrap, instalação, migração e recuperação. A Automation API é preferencial após o Dockge estar operacional. Dockge Deploy não é daemon de Control Plane obrigatório.
