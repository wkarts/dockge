<div align="center">
  <img src="./frontend/public/icon.svg" width="128" alt="Dockge" />

# Dockge

**Docker Compose Management & Infrastructure Automation Platform**

[![CI](https://github.com/wkarts/dockge/actions/workflows/00-ci.yml/badge.svg?branch=develop)](https://github.com/wkarts/dockge/actions/workflows/00-ci.yml)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fwkarts%2Fdockge-blue)](https://github.com/wkarts/dockge/pkgs/container/dockge)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

## Projeto independente

`wkarts/dockge` é uma continuação **independente** do código-base Dockge, evoluída para operação API-first, automação de infraestrutura e integração segura com Control Planes.

O projeto não utiliza mais o repositório, Docker Hub, homepage, workflows de release ou governança do projeto de origem como canais operacionais. A atribuição exigida pela licença MIT original permanece preservada em `LICENSE`.

## Capacidades

- gerenciamento visual de stacks `compose.yaml`;
- criação, edição, start, stop, restart, pull e remoção de stacks;
- terminal web e acompanhamento de operações em tempo real;
- REST API em `/api/v1/automation` para automação máquina-a-máquina;
- Bearer tokens com armazenamento por SHA-256, scopes e expiração;
- isolamento por prefixo/namespace de deployment;
- adoção explícita de stacks legadas — nenhuma stack externa é assumida automaticamente;
- auditoria das operações da API;
- Generic Infrastructure Agent multiplataforma em `infrastructure-agent/`;
- integração outbound-only entre Agent e Control Planes;
- imagens multi-arch publicadas no GHCR;
- dados persistentes separados da imagem da aplicação.

## Git Flow canônico

```text
feature/* / fix/* / ci/*
          │
          ▼
       develop  ──> GHCR :develop + :develop-<sha>
          │
          │ promoção por PR
          ▼
         main   ──> SemVer + GHCR :X.Y.Z + :latest + GitHub Release
```

- `main`: produção e releases estáveis.
- `develop`: integração e homologação.
- `master`: branch legada congelada durante a migração administrativa; será removida após `main` tornar-se a default branch.

Consulte [`docs/BRANCHING-AND-GHCR.md`](docs/BRANCHING-AND-GHCR.md).

## Instalação com Docker Compose

Requisitos: Docker Engine com Docker Compose v2.

```bash
sudo mkdir -p /opt/dockge/data /opt/stacks
cd /opt/dockge
curl -fsSL https://raw.githubusercontent.com/wkarts/dockge/main/compose.yaml -o compose.yaml
docker compose pull
docker compose up -d
```

Por padrão o Compose canônico usa:

```text
ghcr.io/wkarts/dockge:latest
```

Para homologação:

```text
ghcr.io/wkarts/dockge:develop
```

### Persistência

```text
/opt/dockge/data  -> banco, configuração, tokens e auditoria
/opt/stacks       -> stacks e arquivos compose
```

A atualização da imagem não deve apagar esses dados.

## API-first

A API de automação fica em:

```text
/api/v1/automation
```

Exemplos de capacidades:

```text
GET    /health
GET    /info
GET    /stacks
GET    /stacks/:name
PUT    /stacks/:name
DELETE /stacks/:name
POST   /stacks/:name/pull
POST   /stacks/:name/up
POST   /stacks/:name/down
POST   /stacks/:name/start
POST   /stacks/:name/stop
POST   /stacks/:name/restart
GET    /stacks/:name/ps
GET    /stacks/:name/logs
```

A API não oferece shell remoto arbitrário. Automação deve usar ações tipadas e escopadas.

Documentação: [`docs/contracts/dockge-api-v1.md`](docs/contracts/dockge-api-v1.md).

## Generic Infrastructure Agent

O diretório [`infrastructure-agent/`](infrastructure-agent/) contém um módulo Go independente, com versão e `go.mod` próprios. Ele pode ser separado futuramente para `wkarts/infrastructure-agent` sem acoplamento ao código Node/Vue do Dockge.

O desenho é:

```text
Control Plane
     │ HTTPS/REST outbound
     ▼
Generic Infrastructure Agent
     │ REST local e autenticado
     ▼
Dockge API
     │
     ▼
Docker / Compose
```

Políticas comerciais — inadimplência, direito a upgrade, autorização do cliente, janela de manutenção e exigência de backup — pertencem ao **Control Plane consumidor**, não ao Dockge nem ao Agent.

## Atualização

```bash
cd /opt/dockge
docker compose pull
docker compose up -d
```

Em instalações de clientes, a política recomendada é atualização manual/autorizada pelo respectivo Control Plane. O executor não decide regras comerciais.

## Desenvolvimento

```bash
npm ci
npm run check-ts
npm run lint
npm run dev
```

Para o Agent:

```bash
cd infrastructure-agent
go test ./...
go build ./cmd/infra-agent
```

## Contribuição e segurança

- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`SECURITY.md`](SECURITY.md)
- Pull Requests: https://github.com/wkarts/dockge/pulls

## Licença e origem

MIT. Este repositório deriva de trabalho originalmente distribuído sob licença MIT. Os avisos de copyright existentes são mantidos conforme exigido pela licença.
