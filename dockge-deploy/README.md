# Dockge Deploy

Ferramenta CLI multiplataforma para preparar, instalar, diagnosticar e atualizar o host Linux onde o Dockge Core roda. Não é daemon, não é Control Plane e não substitui os Native Agents internos do Dockge.

## Princípios

- SSH somente para host lifecycle, bootstrap, instalação, upgrade e recuperação;
- host key verification obrigatória;
- host desconhecido só pode ser aceito explicitamente com `--accept-new-host-key` e é gravado em `known_hosts`;
- mudança posterior da host key é bloqueada;
- comandos destrutivos não são executados por padrão;
- `install` e `upgrade` apenas exibem plano até receberem `--apply`;
- upgrade preserva stacks e dados, recria somente o container Dockge e possui rollback local automático;
- nunca executa `docker compose down -v`, `docker system prune -a --volumes` ou remoção de `/opt/stacks`.

## Comandos

```text
dockge-deploy host inspect
dockge-deploy doctor
dockge-deploy dockge detect
dockge-deploy dockge install
dockge-deploy dockge upgrade
dockge-deploy dockge rollback
dockge-deploy dockge plan-migration
dockge-deploy version
```

Exemplo de inventário:

```bash
dockge-deploy host inspect \
  --host 10.0.0.15 \
  --user root \
  --key ~/.ssh/id_ed25519
```

Primeiro vínculo, depois de conferir o fingerprint exibido pelo ambiente:

```bash
dockge-deploy dockge detect \
  --host 10.0.0.15 \
  --accept-new-host-key
```

Instalação segura em duas etapas:

```bash
# só plano
dockge-deploy dockge install --host 10.0.0.15 --version 1.6.1

# execução explícita
dockge-deploy dockge install --host 10.0.0.15 --version 1.6.1 --apply
```

Upgrade seguro:

```bash
# só plano
dockge-deploy dockge upgrade --host 10.0.0.15 --version 1.6.1

# snapshot + imagem de rollback + pull + recriação apenas do Dockge + verificação
dockge-deploy dockge upgrade --host 10.0.0.15 --version 1.6.1 --apply
```

Rollback manual de um backup criado pelo upgrade:

```bash
dockge-deploy dockge rollback --host 10.0.0.15 --backup /opt/dockge/backups/upgrade-AAAAMMDDTHHMMSSZ --apply
```

Se o usuário SSH não for root, acrescente `--sudo`; o comando usa `sudo -n` e falha se ele solicitar prompt. Autenticação por senha só é lida de `DOCKGE_DEPLOY_SSH_PASSWORD`; o valor não é aceito como argumento de linha de comando.

## Build

```bash
cd dockge-deploy
go test ./...
go build ./cmd/dockge-deploy
```

O workflow do projeto gera binários para:

- Linux amd64/arm64;
- Windows amd64/arm64;
- macOS amd64/arm64.

Os artefatos do Dockge Deploy usam release própria `dockge-deploy-vX.Y.Z` e não são anexados às releases do Dockge Core.
