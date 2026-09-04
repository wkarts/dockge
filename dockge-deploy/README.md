# Dockge Deploy

Ferramenta CLI multiplataforma para preparar, instalar, diagnosticar e atualizar o host Linux onde o Dockge Core roda. Não é daemon, não é Control Plane e não substitui os Native Agents internos do Dockge.

## Princípios

- SSH somente para host lifecycle, bootstrap, instalação, upgrade e recuperação;
- host key verification obrigatória;
- host desconhecido exibe o fingerprint SHA-256 e só pode ser aceito explicitamente com `--accept-new-host-key`;
- a host key aceita é persistida em `known_hosts` e qualquer mudança posterior é bloqueada;
- chave privada, `ssh-agent` via `SSH_AUTH_SOCK` e senha são suportados;
- passphrase de chave criptografada só é lida de `DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE` e não é persistida;
- comandos destrutivos não são executados por padrão;
- `install`, `upgrade` e `rollback` apenas exibem plano até receberem `--apply`;
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

### Primeiro vínculo SSH

A primeira tentativa, **sem** `--accept-new-host-key`, falha de propósito e mostra o fingerprint SHA-256 recebido:

```bash
dockge-deploy dockge detect --host 10.0.0.15
```

Confira esse fingerprint por um canal independente no servidor. Somente depois da validação, persista a chave:

```bash
dockge-deploy dockge detect \
  --host 10.0.0.15 \
  --accept-new-host-key
```

Conexões seguintes exigem exatamente a chave persistida. Uma alteração inesperada de host key é recusada.

### Autenticação SSH

O Deploy tenta a chave indicada por `--key` e usa automaticamente um agente OpenSSH compatível quando `SSH_AUTH_SOCK` estiver disponível. Como alternativas, segredos podem ser fornecidos somente por ambiente:

```bash
export DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE='passphrase-da-chave-criptografada'
export DOCKGE_DEPLOY_SSH_PASSWORD='senha-ssh-quando-necessaria'
```

Esses segredos não são aceitos como argumentos de linha de comando e não são gravados pelo Dockge Deploy.

## Instalação segura em duas etapas

```bash
# só plano
dockge-deploy dockge install --host 10.0.0.15 --version 1.6.1

# execução explícita
dockge-deploy dockge install --host 10.0.0.15 --version 1.6.1 --apply
```

## Upgrade seguro

```bash
# só plano
dockge-deploy dockge upgrade --host 10.0.0.15 --version 1.6.1

# snapshot + imagem de rollback + pull + recriação apenas do Dockge + verificação
dockge-deploy dockge upgrade --host 10.0.0.15 --version 1.6.1 --apply
```

## Rollback manual

```bash
dockge-deploy dockge rollback \
  --host 10.0.0.15 \
  --backup /opt/dockge/backups/upgrade-AAAAMMDDTHHMMSSZ \
  --apply
```

Se o usuário SSH não for root, acrescente `--sudo`; o comando usa `sudo -n` e falha se o sudo solicitar prompt interativo.

## Migração

`dockge plan-migration` é deliberadamente read-only na linha 0.1.0. Ele inventaria origem, destino, dados e stacks para que uma migração seja revisada antes de qualquer escrita. Instalações encontradas nunca são substituídas automaticamente.

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
