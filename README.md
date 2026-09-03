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
- **API Access** no painel para criar, rotacionar e revogar credenciais;
- Bearer tokens armazenados somente por SHA-256, com scopes, expiração e namespaces;
- segredo de API exibido apenas uma vez na criação/rotação;
- 2FA/TOTP funcional para acesso humano;
- sessões web com validade limitada e revisão de segurança que invalida tokens antigos após senha/2FA;
- isolamento por prefixo/namespace de deployment;
- adoção explícita de stacks legadas — nenhuma stack externa é assumida automaticamente;
- auditoria das operações da API e do ciclo de credenciais;
- Generic Infrastructure Agent multiplataforma em `infrastructure-agent/`;
- integração outbound-only entre Agent e múltiplos Control Planes;
- journal persistente de `action_id` para impedir execução duplicada após falhas de rede;
- pipelines multi-arch para Dockge e Agent;
- dados persistentes separados das imagens da aplicação.

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

- `main`: produção e releases estáveis.
- `develop`: integração e homologação.
- `master`: branch legada congelada durante a migração administrativa; será removida após `main` tornar-se a default branch.

Os workflows de publicação estão implementados. **A existência de uma tag/README não substitui a confirmação de uma execução bem-sucedida do GitHub Actions/GHCR.** Antes da primeira implantação independente, confirme a imagem/tag publicada no registry.

Consulte [`docs/BRANCHING-AND-GHCR.md`](docs/BRANCHING-AND-GHCR.md).

## Instalação com Docker Compose

Requisitos: Docker Engine com Docker Compose v2.

Depois que uma imagem independente tiver sido publicada no GHCR:

```bash
sudo mkdir -p /opt/dockge/data /opt/stacks
cd /opt/dockge
curl -fsSL https://raw.githubusercontent.com/wkarts/dockge/main/compose.yaml -o compose.yaml
# Defina DOCKGE_IMAGE_TAG para uma versão publicada quando quiser pin de produção.
docker compose pull
docker compose up -d
```

Canais previstos:

```text
ghcr.io/wkarts/dockge:develop        homologação
ghcr.io/wkarts/dockge:<X.Y.Z>        produção pinada
ghcr.io/wkarts/dockge:latest         release estável mais recente
```

### Persistência

```text
/opt/dockge/data  -> banco, configuração, tokens e auditoria
/opt/stacks       -> stacks e arquivos compose
```

A atualização da imagem não deve apagar esses dados.

## Segurança humana

A tela **Settings → Security** oferece 2FA/TOTP. Mudanças de senha ou política 2FA incrementam a revisão de autenticação da conta, invalidam sessões anteriores e exigem novo login.

A desativação de autenticação web não é oferecida como operação normal de administração. O backend bloqueia `disableAuth=true` por padrão. Somente uma implantação explicitamente isolada pode liberar esse comportamento com:

```text
DOCKGE_ALLOW_DISABLE_AUTH=true
```

Para acesso remoto, mantenha Dockge em loopback/rede privada e publique a interface atrás de reverse proxy TLS. Veja [`docs/REVERSE-PROXY.md`](docs/REVERSE-PROXY.md).

## API-first

A API de automação fica em:

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

A API não oferece shell remoto arbitrário. Automação usa ações tipadas, scopes e namespaces.

As credenciais são administradas em **Settings → API Access**. O servidor retorna o segredo completo apenas no momento da criação/rotação e persiste somente seu SHA-256.

Documentação: [`docs/contracts/dockge-api-v1.md`](docs/contracts/dockge-api-v1.md).

## Generic Infrastructure Agent

O diretório [`infrastructure-agent/`](infrastructure-agent/) contém um módulo Go independente, com versão e `go.mod` próprios. Ele pode ser separado futuramente para `wkarts/infrastructure-agent` sem acoplamento ao código Node/Vue do Dockge.

```text
Control Plane A ─┐
Control Plane B ─┼─ HTTPS/REST outbound ─> Generic Infrastructure Agent
Control Plane C ─┘                              │
                                                │ REST local autenticado
                                                ▼
                                           Dockge API
                                                │
                                                ▼
                                         Docker / Compose
```

Cada Control Plane possui identidade, credencial e namespaces próprios. Um vínculo indisponível não deve bloquear os demais.

### Idempotência de ações

O Agent mantém `action-journal.json` em seu `data_dir`.

```text
Control Plane envia action_id=act-123
        ↓
Agent executa uma vez
        ↓
grava resultado no journal
        ↓
reporta resultado
```

Se a rede cair após a execução e o Control Plane reenviar `act-123`, o Agent **não executa novamente**. Ele reenvia o resultado persistido. Para uma tentativa real nova, o Control Plane precisa emitir outro `action_id`.

Contrato: [`docs/contracts/control-plane-agent-api.md`](docs/contracts/control-plane-agent-api.md).

## Instaladores do Agent

A distribuição está preparada para produzir artefatos nativos em runners do próprio sistema operacional:

- Linux amd64/arm64: binário CLI, TUI de instalação, `.deb`, `.rpm`, `systemd`;
- Windows amd64/arm64: `infra-agent.exe`, Windows Service nativo, PowerShell CLI/TUI, assistente visual e Setup `.exe`;
- macOS Intel/Apple Silicon: binário, assistente shell, LaunchDaemon e `.pkg`.

Todos os formatos mantêm modo automatizável e modo interativo apropriado ao ambiente.

## Responsabilidade do Control Plane

Políticas comerciais — inadimplência, entitlement de atualização, versão autorizada, aprovação do cliente, janela de manutenção e exigência de backup — pertencem ao **Control Plane consumidor**, não ao Dockge nem ao Agent.

Em uma instalação de cliente, a regra recomendada permanece: atualização manual/autorizada; o Control Plane valida política/backup e somente então emite as ações técnicas para o Agent.

## Desenvolvimento

Dockge:

```bash
npm ci
npm run check-ts
npm run lint
npm run test:security
npm run build:frontend
```

Agent:

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

MIT. Este repositório deriva de trabalho originalmente distribuído sob licença MIT. Os avisos de copyright existentes são mantidos conforme exigido pela licença. A preservação histórica/legal não cria dependência operacional com o projeto de origem.
