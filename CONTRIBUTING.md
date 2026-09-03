# Contribuindo com o Dockge

Este repositório é mantido como projeto independente em `wkarts/dockge`.

## Fluxo de branches

- `main`: produção e releases estáveis.
- `develop`: integração e homologação.
- `feat/*`, `feature/*`, `fix/*`, `hotfix/*`, `chore/*`, `refactor/*`, `docs/*`, `ci/*`, `test/*`, `perf/*`: branches de trabalho.
- `master`: branch legada congelada durante a migração para `main`; não é destino de desenvolvimento.

Toda mudança funcional deve entrar por Pull Request. Features e correções comuns têm `develop` como base. A promoção para produção ocorre por PR `develop -> main`.

## Regras de Pull Request

Uma PR deve:

- ter escopo claro e reversível;
- preservar dados e compatibilidade sempre que possível;
- passar CI, TypeScript, lint e build do frontend;
- atualizar documentação quando alterar contratos públicos;
- não introduzir publicação em registries ou namespaces que não pertençam a `wkarts`;
- não introduzir shell remoto arbitrário na API de automação ou no Generic Infrastructure Agent;
- manter regras comerciais fora do Dockge/Agent: licenciamento, inadimplência, direito a upgrade e autorização pertencem ao Control Plane consumidor.

Mudanças incompatíveis devem ser identificadas explicitamente como `BREAKING CHANGE` e acompanhadas de migração.

## Convenções de commit

Usamos Conventional Commits:

```text
feat: nova capacidade
fix: correção
chore: manutenção
refactor: refatoração sem mudança funcional
docs: documentação
ci: integração/entrega contínua
test: testes
perf: desempenho
```

`feat` promove incremento minor no próximo release estável; correções/manutenção promovem patch; breaking changes promovem major.

## Ambiente Dockge

Requisitos:

- Node.js >= 22.14.0;
- npm;
- Docker Engine + Docker Compose v2;
- Git.

Instalação:

```bash
npm ci
```

Validação:

```bash
npm run check-ts
npm run lint
npm run build:frontend
```

Desenvolvimento:

```bash
npm run dev
```

## Generic Infrastructure Agent

O Agent possui módulo Go próprio em `infrastructure-agent/`.

```bash
cd infrastructure-agent
go test ./...
go build ./cmd/infra-agent
```

O Agent deve permanecer genérico e desacoplado de produtos consumidores. Não adicione condicionais específicas de PIGE360, Connect API, ERP ou qualquer outra plataforma no núcleo do Agent.

## Estilo

- 4 espaços em TypeScript/Vue conforme `.editorconfig`;
- ESLint obrigatório;
- nomes TypeScript/JavaScript em camelCase;
- SQLite em snake_case;
- CSS/SCSS em kebab-case;
- APIs e componentes públicos devem ter documentação suficiente para manutenção.

## Persistência

Nunca armazene dados operacionais necessários somente dentro da imagem Docker. Configuração, banco, tokens, auditoria e stacks devem permanecer em volumes/pastas persistentes.

## Segurança

Vulnerabilidades não devem ser publicadas em issues abertas. Siga `SECURITY.md`.

## Licença e atribuição

O projeto continua sob MIT. Arquivos derivados do código-base original devem preservar avisos de copyright/licença aplicáveis. Independência operacional não remove as obrigações de atribuição da licença.
