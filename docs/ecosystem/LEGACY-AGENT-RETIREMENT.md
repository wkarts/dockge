# Retirada do Generic Infrastructure Agent 0.2.0

## Situação

A implementação `infrastructure-agent/` 0.2.0 foi construída como daemon ligado a Control Planes, com enrollment, heartbeat, desired-state e execução através de Dockge API local. Esse modelo não representa mais o papel aprovado.

## Decisão

A linha 0.2.0 é **legada/encerrada**. Não será base do Dockge Deploy nem do Dockge Manager.

## Preservar no Dockge Core

- Automation REST API;
- tokens/scopes/namespaces;
- `Idempotency-Key`;
- audit log;
- gravação segura de stacks;
- Native Agents e AgentManager.

## Retirar em mudança separada e auditável

Candidatos explícitos:

```text
infrastructure-agent/
.github/workflows/30-agent-build.yml
.github/workflows/55-agent-assets.yml
.github/workflows/60-agent-release.yml
docs/contracts/control-plane-agent-api.md
```

Também devem ser revisadas referências no README, release docs e demais documentos.

## Proteção obrigatória

A retirada nunca usará padrões amplos como `*agent*`, pois isso poderia atingir a tecnologia de Native Agents. A lista de paths removidos deve ser explícita e a CI deve comprovar que os arquivos nativos não foram alterados.

## Release 1.6.0

Os artefatos 0.2.0 já publicados são históricos. Remover assets de uma release existente é decisão operacional separada; retirar o legado do código futuro não exige reescrever a release 1.6.0.

## Sequência

1. aprovar a arquitetura nova;
2. preparar os projetos Manager e Deploy;
3. criar scaffold mínimo dos novos produtos;
4. retirar o legado 0.2.0 em PR isolado;
5. executar CI completa do Dockge;
6. verificar explicitamente que Automation API e Native Agents permanecem intactos.
