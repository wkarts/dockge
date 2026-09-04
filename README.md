<div align="center">
  <img src="./frontend/public/icon.svg" width="128" alt="Dockge" />

# Dockge

**Docker Compose Management & Automation API**

[![CI](https://github.com/wkarts/dockge/actions/workflows/00-ci.yml/badge.svg?branch=main)](https://github.com/wkarts/dockge/actions/workflows/00-ci.yml)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fwkarts%2Fdockge-blue)](https://github.com/wkarts/dockge/pkgs/container/dockge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

## Dockge Core

`wkarts/dockge` é o orquestrador Docker/Compose do ecossistema Dockge e permanece a **fonte de verdade operacional** para stacks, containers, logs e operações de Compose.

O Core não depende de Dockge Manager ou Dockge Deploy. A tecnologia de **Native Agents** existente no Dockge permanece preservada e independente.

## Capacidades

- gerenciamento visual de stacks `compose.yaml`;
- criação, edição, start, stop, restart, pull e remoção de stacks;
- terminal web e acompanhamento de operações em tempo real;
- REST API em `/api/v1/automation` para automação máquina-a-máquina;
- **API Access** no painel para criar, rotacionar e revogar credenciais;
- Bearer tokens armazenados somente por SHA-256, com scopes, expiração e namespaces;
- segredo de API exibido apenas uma vez na criação/rotação;
- 2FA/TOTP para acesso humano;
- isolamento por prefixo/namespace de deployment;
- adoção explícita de stacks legadas;
- auditoria das operações da API e do ciclo de credenciais;
- idempotência persistente em mutações via `Idempotency-Key`;
- imagem multi-arch no GHCR;
- dados persistentes separados da imagem da aplicação.

## Ecossistema

A arquitetura oficial separa quatro domínios:

```text
Dockge Core        orquestrador Docker/Compose e fonte de verdade
Dockge Manager     management plane PWA (FastAPI + Vue)
Dockge Deploy      host lifecycle via SSH + Dockge Automation API
Native Agents      tecnologia interna existente do Dockge
```

Manager e Deploy consomem contratos públicos; eles não acessam o banco interno do Dockge, não alteram os Native Agents e não substituem o motor Docker/Compose do Core.

Documentação: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) e [`docs/ecosystem/`](docs/ecosystem/README.md).

## Git Flow canônico

```text
feature/* / fix/* / ci/*
          │
          ▼
       develop  ──> build/test ──> GHCR :develop + :develop-<sha>
          │
          │ promoção por PR
          ▼
         main   ──> SemVer + GHCR :X.Y.Z + :latest + GitHub Release
```

O workflow de release do **Dockge Core** é restrito aos caminhos do próprio runtime. Alterações exclusivas de Manager/Deploy/documentação não devem incrementar a versão do Core.

## Instalação com Docker Compose

Requisitos: Docker Engine com Docker Compose v2.

```bash
sudo mkdir -p /opt/dockge/data /opt/stacks
cd /opt/dockge
curl -fsSL https://raw.githubusercontent.com/wkarts/dockge/main/compose.yaml -o compose.yaml
# Em produção, prefira fixar DOCKGE_IMAGE_TAG em uma versão publicada.
docker compose pull
docker compose up -d
```

Canais:

```text
ghcr.io/wkarts/dockge:develop
ghcr.io/wkarts/dockge:<X.Y.Z>
ghcr.io/wkarts/dockge:latest
```

### Persistência

```text
/opt/dockge/data  -> banco, configuração, tokens e auditoria
/opt/stacks       -> stacks e arquivos compose
```

Atualizar/recriar somente o container Dockge não deve apagar os stacks ou os dados persistentes.

## Segurança humana

A tela **Settings → Security** oferece 2FA/TOTP. A desativação de autenticação web não é operação normal; `disableAuth=true` é bloqueado por padrão e somente pode ser liberado em implantação isolada com `DOCKGE_ALLOW_DISABLE_AUTH=true`.

Para acesso remoto, mantenha Dockge em loopback/rede privada e publique a interface atrás de reverse proxy TLS. Veja [`docs/REVERSE-PROXY.md`](docs/REVERSE-PROXY.md).

## Automation REST API

Base:

```text
/api/v1/automation
```

Principais endpoints:

```text
GET    /health
GET    /info
GET    /stacks
GET    /stacks/:name
PUT    /stacks/:name
DELETE /stacks/:name
POST   /stacks/:name/actions/pull
POST   /stacks/:name/actions/up
POST   /stacks/:name/actions/down
POST   /stacks/:name/actions/start
POST   /stacks/:name/actions/stop
POST   /stacks/:name/actions/restart
GET    /stacks/:name/ps
GET    /stacks/:name/logs
```

A API não oferece shell remoto arbitrário. Usa ações tipadas, scopes, namespaces, adoção explícita e idempotência persistente.

Documentação: [`docs/contracts/dockge-api-v1.md`](docs/contracts/dockge-api-v1.md).

## Generic Infrastructure Agent 0.2.0

A implementação externa `infrastructure-agent/` foi **retirada**. Ela tinha sido criada como daemon de Control Plane e não corresponde ao desenho aprovado do ecossistema.

Isso **não remove nem altera os Native Agents do Dockge**.

O substituto funcional é o **Dockge Deploy**, definido em [`docs/ecosystem/DOCKGE-DEPLOY.md`](docs/ecosystem/DOCKGE-DEPLOY.md): ferramenta de host lifecycle executada a partir de Windows/macOS/Linux, usando SSH para bootstrap/migração/recuperação e Automation API para operações suportadas.

## Desenvolvimento do Core

```bash
npm ci
npm run check-ts
npm run lint
npm run test:security
npm run build:frontend
```

## Contribuição e segurança

- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`SECURITY.md`](SECURITY.md)
- Pull Requests: https://github.com/wkarts/dockge/pulls

## Licença e origem

MIT. Este repositório deriva de trabalho originalmente distribuído sob licença MIT. Os avisos de copyright existentes são mantidos conforme exigido pela licença. A preservação histórica/legal não cria dependência operacional com o projeto de origem.
