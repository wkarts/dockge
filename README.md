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

> A implementação independente está sendo promovida de `develop` para `main` pela PR #2. Este README administrativo já representa a governança canônica do novo projeto.

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

## Instalação com Docker Compose

Após a promoção da PR #2, o Compose canônico será obtido diretamente de `main`:

```bash
sudo mkdir -p /opt/dockge/data /opt/stacks
cd /opt/dockge
curl -fsSL https://raw.githubusercontent.com/wkarts/dockge/main/compose.yaml -o compose.yaml
docker compose pull
docker compose up -d
```

Registry oficial:

```text
ghcr.io/wkarts/dockge
```

Homologação:

```text
ghcr.io/wkarts/dockge:develop
```

Produção:

```text
ghcr.io/wkarts/dockge:<semver>
ghcr.io/wkarts/dockge:latest
```

## Responsabilidades

Regras comerciais — inadimplência, direito a upgrade, autorização do cliente, janela de manutenção e backup obrigatório — pertencem ao **Control Plane consumidor**, nunca ao Dockge nem ao Generic Infrastructure Agent.

## Contribuição e segurança

- Pull Requests: https://github.com/wkarts/dockge/pulls
- Security Advisories: https://github.com/wkarts/dockge/security/advisories/new

## Licença e origem

MIT. O histórico e os avisos de copyright do código-base original permanecem preservados conforme a licença. A preservação histórica/legal não cria vínculo operacional com o projeto de origem.
