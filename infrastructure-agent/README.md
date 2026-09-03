# Generic Infrastructure Agent

Agente de infraestrutura multiplataforma, independente de produto, para vincular VPS/servidores a um ou mais Control Planes por REST/HTTPS outbound-only e executar ações tipadas através de uma API local do Dockge.

Ele não pertence ao PIGE360, Connect|API, ERP, Scheduler ou Mailcow. Cada plataforma é apenas um consumidor autorizado do mesmo agente.

## Arquitetura

```text
Control Plane A ── HTTPS ─┐
Control Plane B ── HTTPS ─┼─> Generic Infrastructure Agent
Control Plane C ── HTTPS ─┘           │
                                      │ credencial Dockge própria
                                      │ para cada vínculo
                                      ▼
                                Dockge API local
                                      │
                                      ▼
                                Docker / Compose
```

Princípios:

- comunicação permanente de saída; nenhuma porta administrativa do Agent precisa ficar pública;
- um host pode ser inscrito em múltiplos Control Planes;
- cada Control Plane recebe identidade, credencial remota, credencial Dockge local e namespaces próprios;
- uma credencial local de `pige360` não precisa ter acesso a stacks `connect-api-*` e vice-versa;
- configurações históricas com uma credencial Dockge global continuam suportadas como fallback de compatibilidade;
- políticas comerciais, inadimplência, autorização de upgrade e janelas de manutenção pertencem ao Control Plane, não ao Agent;
- o Agent não aceita shell arbitrário vindo da rede;
- Dockge é o executor Docker/Compose local e deve ficar em loopback/rede privada por padrão;
- segredos ficam em arquivos separados do `agent.json`;
- instalações existentes são descobertas e preservadas; adoção/migração é sempre explícita.

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

`configure` adiciona ou atualiza somente o vínculo cujo nome foi informado. Os demais Control Planes já configurados são preservados. A gravação do JSON e do journal usa substituição atômica específica para Unix/Windows.

## Idempotência

Cada ação recebida do Control Plane precisa de `action_id`. O Agent persiste o resultado em `action-journal.json`, com chave `controller + action_id`.

```text
action_id=act-123
     ↓
executa uma vez
     ↓
persiste resultado
     ↓
reporta

rede falhou e act-123 chegou de novo
     ↓
NÃO executa de novo
     ↓
reenvia resultado persistido
```

Para uma nova tentativa operacional o Control Plane gera outro `action_id`.

## Descoberta de runtime e coexistência

O inventário diferencia:

- Docker/Compose instalados ou ausentes;
- endpoint Dockge alcançável;
- Dockge com `/api/v1/automation` realmente compatível;
- containers Dockge existentes, inclusive parados;
- nome, imagem e estado de cada container detectado.

Uma página HTML de Dockge antigo respondendo HTTP 200 **não** é confundida com a Automation API.

O Agent não remove, substitui, atualiza ou adota Dockge de outro provider automaticamente.

## Experiência de instalação

### Linux

O pacote inclui `infra-agent-installer`, um assistente TUI que pode trabalhar em três níveis:

```text
Agent apenas
Agent + configuração de Control Plane
Agent + bootstrap completo do host
```

No bootstrap completo ele:

1. diagnostica Docker/Compose;
2. oferece instalar Docker pelo repositório oficial em distribuições suportadas;
3. inventaria Dockge(s) existentes, inclusive parados;
4. preserva todos por padrão;
5. pode reutilizar uma instância `ghcr.io/wkarts/dockge` já existente;
6. ou instalar uma nova instância API-first em coexistência, loopback e diretórios próprios;
7. cria/rotaciona uma credencial Dockge exclusiva para o Control Plane atual;
8. configura/enrolla o Agent.

```bash
sudo infra-agent-installer
```

Pacotes `.deb` e `.rpm` são não interativos durante `apt/dnf`; o menu é iniciado depois pelo comando acima. Ambos incluem a biblioteca de bootstrap do host.

### Windows

A distribuição contém:

- `infra-agent.exe` — CLI e Windows Service nativo;
- Setup `.exe` para amd64 e arm64;
- PowerShell CLI/TUI;
- assistente visual PowerShell;
- launcher `.cmd` com UAC;
- delayed auto-start, políticas de recuperação e ACLs restritas da configuração.

### macOS

A distribuição contém binários Intel/Apple Silicon, `.pkg`, LaunchDaemon e assistente shell. Os caminhos nativos ficam em `/Library/Application Support/InfrastructureAgent`.

## Contrato REST do Control Plane

```text
POST /api/v1/infrastructure/agents/enroll
POST /api/v1/infrastructure/agents/{id}/heartbeat
GET  /api/v1/infrastructure/agents/{id}/desired-state
POST /api/v1/infrastructure/agents/{id}/actions/{action_id}/result
```

O Agent apenas reconcilia ações tipadas e autorizadas. Um Control Plane indisponível não paralisa os demais; enrollment pendente é tentado novamente nos ciclos seguintes.

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

Tokens Dockge usam scopes e namespaces. O instalador pode gerar a credencial local via CLI interna do próprio Dockge; o segredo passa ao `configure` apenas para ser materializado no arquivo protegido daquele vínculo.

## Versionamento e distribuição

- `develop`: integração/homologação; builds são artifacts de CI;
- `main`: linha estável;
- tags `infrastructure-agent-vX.Y.Z`: GitHub Release estável do Agent;
- Linux amd64/arm64: binário, `.deb`, `.rpm`, systemd;
- Windows amd64/arm64: binário/serviço + Setup `.exe`;
- macOS amd64/arm64: binário + `.pkg` + LaunchDaemon;
- todos os pacotes recebem SHA-256.

O workflow `30 · Agent Build` usa runners nativos Linux, Windows e macOS. O workflow `60 · Agent Release` publica artefatos estáveis somente a partir de `main`, mantendo a versão do Agent independente da versão do Dockge.

## Segurança

2FA pertence às sessões humanas no Control Plane/Dockge. Comunicação máquina-a-máquina usa credenciais próprias, revogáveis e escopadas; futuramente pode adicionar mTLS por instalação. Não se exige TOTP humano em heartbeat ou reconciliação automática.
