# Branching, persistência e GHCR

## Branches canônicas

- `main`: produção estável e origem de releases oficiais.
- `develop`: integração e homologação.
- `feat/*`, `feature/*`, `fix/*`, `hotfix/*`, `chore/*`, `refactor/*`, `docs/*`, `ci/*`, `test/*`, `perf/*`: branches de trabalho com PR para `develop`.
- promoção de release: PR `develop -> main`.

## Estado do antigo `master`

`master` é uma branch legada de migração. Ela não recebe desenvolvimento nem releases.

Enquanto o GitHub ainda a mantiver como default branch, seu README informa explicitamente que a linha ativa é `develop/main` e workflows herdados de release permanecem desativados. Depois que `main` for definida administrativamente como default branch, `master` deve ser excluída.

A história de commits anterior permanece no grafo Git por razões de rastreabilidade e licença; isso não significa vínculo operacional com outro repositório.

## Dockge no GHCR

PR para `develop` constrói e valida, mas não publica.

Push/merge em `develop` publica:

- `ghcr.io/wkarts/dockge:develop`
- `ghcr.io/wkarts/dockge:develop-<sha12>`

Promoção em `main` publica SemVer estável:

- `ghcr.io/wkarts/dockge:X.Y.Z`
- `ghcr.io/wkarts/dockge:latest`

O projeto não usa Docker Hub ou namespace de terceiros como registry oficial.

## Generic Infrastructure Agent

O Agent possui SemVer próprio em `infrastructure-agent/VERSION`.

Builds de PR/develop geram artefatos de validação para Linux, Windows e macOS. Releases estáveis do Agent são publicados em GitHub Releases com tag:

```text
infrastructure-agent-vX.Y.Z
```

Artefatos previstos:

- Linux amd64/arm64: binário, `.deb` e `.rpm`;
- Windows amd64/arm64: `.exe` CLI/service e Setup `.exe`;
- macOS Intel/Apple Silicon: binário e `.pkg`;
- checksums SHA-256.

## Persistência

A imagem do Dockge é descartável. Nunca deve conter o estado administrativo como única cópia.

- `/app/data`: banco SQLite, configuração, tokens API e auditoria;
- `/opt/stacks` (ou `DOCKGE_STACKS_DIR`): Compose/.env das stacks;
- volumes das aplicações: pertencem às próprias stacks.

Atualizar a imagem do Dockge não remove esses dados.
