# Branching, persistência e GHCR

## Branches

- `main`: produção estável. Substitui o antigo `master` como branch padrão.
- `develop`: integração/homologação.
- `feat/*`, `fix/*`, `chore/*`: branches de trabalho -> PR para `develop`.
- promoção de release: `develop` -> `main`.

## Imagens

PR para `develop` constrói e valida, mas não publica.

Push/merge em `develop` publica:

- `ghcr.io/wkarts/dockge:develop`
- `ghcr.io/wkarts/dockge:develop-<sha12>`

Promoção em `main` publica SemVer estável:

- `ghcr.io/wkarts/dockge:X.Y.Z`
- `ghcr.io/wkarts/dockge:latest`

## Persistência

A imagem é descartável. Nunca contém o estado administrativo.

- `/app/data`: banco SQLite, configuração, tokens API e auditoria.
- `/opt/stacks` (ou `DOCKGE_STACKS_PATH`): Compose/.env de stacks.
- volumes das aplicações: pertencem às próprias stacks.

Atualizar a imagem do Dockge não remove nenhum desses dados.
