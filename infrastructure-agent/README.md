# Generic Infrastructure Agent

Agente de infraestrutura multiplataforma, independente de produto, para vincular VPS/servidores a um ou mais Control Planes por REST/HTTPS outbound-only e executar ações tipadas através de uma API local do Dockge.

Ele não pertence ao PIGE360, Connect|API, ERP, Scheduler ou Mailcow. Cada plataforma é apenas um consumidor autorizado do mesmo agente.

## Arquitetura

```text
Control Plane A ─┐
Control Plane B ─┼── REST/HTTPS ──> Generic Infrastructure Agent ──> Dockge API (loopback) ──> Docker/Compose
Control Plane C ─┘
```

Princípios:

- comunicação permanente de saída; nenhuma porta administrativa do Agent precisa ficar pública;
- um host pode ser inscrito em múltiplos Control Planes;
- cada Control Plane recebe escopo independente por prefixos de deployment;
- políticas comerciais, inadimplência, autorização de upgrade e janelas de manutenção pertencem ao Control Plane, não ao Agent;
- o Agent não aceita shell arbitrário vindo da rede;
- Dockge é o executor Docker/Compose local e deve ficar em loopback/rede privada por padrão;
- tokens ficam em arquivos de secrets separados do `agent.json`;
- instalações existentes são descobertas e preservadas; adoção é explícita.

## CLI

```text
infra-agent version
infra-agent --config <arquivo> configure
infra-agent --config <arquivo> enroll
infra-agent --config <arquivo> inventory
infra-agent --config <arquivo> doctor
infra-agent --config <arquivo> once
infra-agent --config <arquivo> run
```

`configure` é destinado aos instaladores. Ele recebe os dados por variáveis `INFRA_AGENT_*`, grava os secrets separadamente e materializa o JSON sem credenciais sensíveis embutidas.

## Experiência de instalação

### Linux

O pacote inclui `install.sh`/`infra-agent-installer` com menu interativo, diagnóstico do host, configuração segura e enrollment.

```bash
sudo ./install.sh
```

Pacotes `.deb` e `.rpm` são deliberadamente seguros para automação: a instalação do pacote não bloqueia `apt`, `dnf` ou pipelines esperando input. Ao terminar, exibem:

```bash
sudo infra-agent-installer
```

Esse comando abre o mesmo assistente visual de terminal. Essa separação preserva uma boa UX sem tornar upgrades de pacote frágeis.

### Windows

A distribuição contém:

- `infra-agent.exe` — binário/CLI e serviço;
- `infrastructure-agent-setup-<versão>-windows-amd64.exe` — instalador gráfico;
- `configure-ui.ps1` — configuração gráfica do vínculo;
- `install-service.ps1` — administrador CLI/TUI;
- `install.cmd` — launcher com elevação UAC.

O instalador registra `InfrastructureAgent` como Windows Service e oferece abrir o assistente visual de configuração ao final.

### macOS

A distribuição contém binários Intel/Apple Silicon, `.pkg`, `LaunchDaemon` e um assistente `install-macos.sh` com menu.

## Contrato REST do Control Plane

```text
POST /api/v1/infrastructure/agents/enroll
POST /api/v1/infrastructure/agents/{id}/heartbeat
GET  /api/v1/infrastructure/agents/{id}/desired-state
POST /api/v1/infrastructure/agents/{id}/actions/{action_id}/result
```

O Agent apenas reconcilia ações tipadas e autorizadas.

## Contrato REST esperado do Dockge API-first

```text
GET    /api/v1/automation/health
GET    /api/v1/automation/stacks
GET    /api/v1/automation/stacks/{deployment}
PUT    /api/v1/automation/stacks/{deployment}
DELETE /api/v1/automation/stacks/{deployment}
POST   /api/v1/automation/stacks/{deployment}/actions/{pull|up|down|restart|start|stop}
GET    /api/v1/automation/stacks/{deployment}/ps
```

Tokens do Dockge usam escopos e namespaces. Uma credencial de PIGE360, por exemplo, pode operar somente `pige360-*` e não enxergar/modificar stacks de outro fornecedor.

## Versionamento e distribuição

- `develop`: integração/homologação; builds são artifacts de CI.
- `main`: linha estável.
- tags `infrastructure-agent-vX.Y.Z`: GitHub Release estável do Agent.
- binários: Linux amd64/arm64, Windows amd64/arm64, macOS amd64/arm64;
- Linux: `.deb` e `.rpm`;
- Windows: Setup `.exe` + PowerShell/BAT;
- macOS: `.pkg` + shell;
- todos os pacotes recebem SHA-256.

O workflow `30 · Agent Build` usa runners nativos Linux, Windows e macOS. O workflow `60 · Agent Release` publica os artefatos estáveis somente a partir de `main`, mantendo a versão do Agent independente da versão do Dockge.

## Segurança

2FA é uma característica de sessões humanas no Control Plane/Dockge. Comunicação máquina-a-máquina usa credenciais próprias, revogáveis e com escopo; futuramente pode usar mTLS por instalação. Não é correto exigir TOTP humano em cada heartbeat ou reconciliação automática.
