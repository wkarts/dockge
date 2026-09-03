# Descrição

Explique o problema, a solução e o impacto esperado.

Fixes #(issue)

## Destino

- [ ] Esta PR tem `develop` como base para integração/homologação.
- [ ] Se esta PR promove `develop -> main`, ela representa uma promoção de release estável.
- [ ] Não tem `master` como destino; `master` é legado congelado.

## Tipo de mudança

- [ ] `feat` — nova capacidade
- [ ] `fix` — correção
- [ ] `refactor` — refatoração sem mudança funcional
- [ ] `docs` — documentação
- [ ] `ci` — CI/CD
- [ ] `chore` — manutenção
- [ ] `test` — testes
- [ ] `perf` — desempenho
- [ ] breaking change

## Checklist técnico

- [ ] Li `CONTRIBUTING.md`.
- [ ] Executei/validei TypeScript, lint e build aplicáveis.
- [ ] Adicionei testes quando a alteração possui comportamento testável.
- [ ] Atualizei documentação/contratos públicos afetados.
- [ ] Não introduzi referências operacionais ao antigo repositório, Docker Hub ou homepage do projeto de origem.
- [ ] Não armazenei segredos no repositório, imagem ou logs.
- [ ] Não introduzi shell remoto arbitrário na API/Agent.
- [ ] Dados persistentes permanecem fora da imagem Docker.

## Automação / Control Plane

Quando aplicável:

- [ ] A ação é tipada e auditável.
- [ ] O token usa o menor conjunto de scopes possível.
- [ ] O deployment/stack respeita namespace/prefixo autorizado.
- [ ] Regras comerciais permanecem no Control Plane e não no Dockge/Agent.

## Migração / rollback

Descreva como atualizar e como retornar à versão anterior quando a mudança tocar banco, persistência, Compose, API ou deployment.

## Evidências

Inclua logs, screenshots ou resultados de teste relevantes. Não publique segredos ou dados sensíveis.
