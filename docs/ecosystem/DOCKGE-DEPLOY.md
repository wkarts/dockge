# Dockge Deploy

## Objetivo

Ferramenta local/remota para preparar e manter o host Linux onde Dockge roda. Não é daemon obrigatório, não depende de Control Plane e não depende do Manager.

## Plataformas do operador

- Windows amd64/arm64;
- macOS Intel/Apple Silicon;
- Linux amd64/arm64.

## Transportes

### SSH

Usado para descoberta do host, instalação de Docker, instalação inicial do Dockge, localização de instalação legada, snapshot, migração, rollback e recuperação.

### Dockge Automation API

Usada quando disponível para validar compatibilidade, listar e inspecionar stacks, executar operações normais e verificar o resultado pós-instalação/migração.

## CLI de referência

```text
dockge-deploy host inspect
dockge-deploy docker inspect
dockge-deploy docker install
dockge-deploy dockge detect
dockge-deploy dockge install
dockge-deploy dockge plan-migration
dockge-deploy dockge migrate
dockge-deploy dockge upgrade
dockge-deploy dockge rollback
dockge-deploy stack verify
dockge-deploy doctor
```

## Estrutura alvo

```text
dockge-deploy/
├── cmd/dockge-deploy/
├── core/
│   ├── capability/
│   ├── inventory/
│   ├── operation/
│   ├── migration/
│   └── rollback/
├── transport/
│   ├── ssh/
│   └── dockgeapi/
├── host/
│   ├── linux/
│   ├── debian/
│   ├── rhel/
│   └── alpine/
├── providers/
│   ├── docker/
│   └── dockge/
├── contracts/
├── tests/
└── packaging/
```

## Capabilities

Exemplos: `host.ssh`, `host.file-transfer`, `host.system-info`, `docker.inspect`, `docker.install`, `docker.compose`, `dockge.detect`, `dockge.install`, `dockge.migrate`, `dockge.upgrade`, `dockge.rollback`, `stack.list`, `stack.inspect`, `stack.start`, `stack.stop`, `stack.restart`, `stack.logs`.

## Ciclo de vida

```text
DISCOVER → INVENTORY → PLAN → PREFLIGHT → SNAPSHOT → EXECUTE → VERIFY → COMMIT
```

Falha:

```text
EXECUTE/VERIFY → FAILED → ROLLBACK → VERIFY_ROLLBACK
```

## Migração

`MigrationPlan` é entidade de primeira classe, contendo origem, destino, paths detectados, stacks, riscos, snapshot, passos, verificação e rollback. Nunca substituir uma instalação encontrada sem plano explícito e confirmação.
