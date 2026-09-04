# Dockge Ecosystem Blueprint 1.0

**Data:** 2026-09-04  
**Estado:** arquitetura de referência aprovada para início de implementação.

Este documento fixa a separação entre:

1. **Dockge Core** — o orquestrador Docker/Compose existente;
2. **Dockge Manager** — PWA de gerenciamento avançado de uma ou várias instalações Dockge;
3. **Dockge Deploy** — ferramenta de implantação, bootstrap, migração, atualização, diagnóstico e recuperação por SSH;
4. **Dockge Native Agents** — tecnologia interna do Dockge existente, preservada.

## Regra-mãe

> O Dockge continua sendo o executor e a fonte de verdade operacional. Manager e Deploy consomem contratos públicos/estáveis e não reimplementam, contornam ou alteram a estrutura interna do Dockge.

## Responsabilidades

```text
Host/Linux       ← Dockge Deploy
Docker/Compose   ← Dockge Core
Stacks/runtime   ← Dockge Core
Gestão agregada  ← Dockge Manager
Agents internos  ← Dockge Core / preservados
```

## Invariantes

- não montar `docker.sock` no Manager;
- não acessar banco interno do Dockge pelo Manager/Deploy;
- não escrever em `/opt/stacks` por fora do Dockge quando a Automation API puder executar a operação;
- não alterar Native Agents para atender Manager/Deploy;
- SSH é usado para host/bootstrap/migração/recuperação;
- Automation API é preferida para administração normal do Dockge e stacks;
- migração e upgrade exigem plano, preflight, snapshot, verificação e rollback;
- produtos de negócio não entram no Core dos novos componentes;
- Dockge, Manager e Deploy têm versões e ciclos de vida independentes.

## Documentos

- `DOCKGE-MANAGER.md`
- `DOCKGE-DEPLOY.md`
- `BOUNDARIES-AND-CONTRACTS.md`
- `LEGACY-AGENT-RETIREMENT.md`
- `ROADMAP.md`
