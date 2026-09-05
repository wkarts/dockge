# Dockge Ecosystem — Quick Start operacional

Este guia cobre os dois cenários suportados pelo ecossistema:

1. preparar um servidor Linux novo e instalar o Dockge;
2. migrar uma instalação Dockge existente para `wkarts/dockge`;
3. conectar a instalação pronta ao Dockge Manager.

O Dockge Core atual permanece o orquestrador. O Manager não acessa Docker diretamente e o Dockge Deploy usa SSH somente no lifecycle do host/orquestrador.

---

## 1. Obter Dockge Deploy

Release estável:

```text
https://github.com/wkarts/dockge/releases/tag/dockge-deploy-v1.0.2
```

Escolha o pacote correspondente:

```text
Linux x64      dockge-deploy-1.0.2-linux-amd64.tar.gz
Linux ARM64    dockge-deploy-1.0.2-linux-arm64.tar.gz
Windows x64    dockge-deploy-1.0.2-windows-amd64.zip
Windows ARM64  dockge-deploy-1.0.2-windows-arm64.zip
macOS Intel    dockge-deploy-1.0.2-darwin-amd64.tar.gz
macOS ARM64    dockge-deploy-1.0.2-darwin-arm64.tar.gz
```

Valide o arquivo com `SHA256SUMS.txt` antes do uso.

No Windows, os exemplos abaixo usam `dockge-deploy.exe`; em Linux/macOS, `dockge-deploy`.

---

## 2. Primeiro vínculo SSH

Consulta inicial:

```bash
dockge-deploy host inspect --host 203.0.113.10 --user root
```

Para um host ainda não conhecido, o Deploy mostra o fingerprint e recusa a conexão. Valide o fingerprint por canal independente e então aceite-o uma vez:

```bash
dockge-deploy host inspect \
  --host 203.0.113.10 \
  --user root \
  --accept-new-host-key
```

Mudanças posteriores de host key são bloqueadas.

Se usar chave criptografada:

```bash
export DOCKGE_DEPLOY_SSH_KEY_PASSPHRASE='...'
```

Se usar senha:

```bash
export DOCKGE_DEPLOY_SSH_PASSWORD='...'
```

Os segredos não são argumentos de CLI.

---

## 3A. Servidor Linux novo

### Inspecionar

```bash
dockge-deploy doctor --host 203.0.113.10 --user root
```

### Planejar bootstrap do Docker

```bash
dockge-deploy docker install --host 203.0.113.10 --user root
```

### Instalar Docker/Compose

```bash
dockge-deploy docker install \
  --host 203.0.113.10 \
  --user root \
  --apply
```

Para usuário não-root com sudo não interativo, acrescente `--sudo`.

### Planejar Dockge

```bash
dockge-deploy dockge install \
  --host 203.0.113.10 \
  --user root \
  --version 1.6.1 \
  --path /opt/dockge \
  --stacks-path /opt/stacks
```

### Instalar Dockge

```bash
dockge-deploy dockge install \
  --host 203.0.113.10 \
  --user root \
  --version 1.6.1 \
  --path /opt/dockge \
  --stacks-path /opt/stacks \
  --apply
```

O instalador recusa sobrescrever uma instalação existente em qualquer um dos nomes Compose padrão:

```text
compose.yaml
compose.yml
docker-compose.yaml
docker-compose.yml
```

---

## 3B. Migrar Dockge existente

Primeiro execute somente o inventário:

```bash
dockge-deploy dockge plan-migration \
  --host 203.0.113.10 \
  --user root \
  --path /opt/dockge \
  --stacks-path /opt/stacks \
  --version 1.6.1
```

O `plan-migration` é read-only. Na versão 1.0.2 ele também mostra o **bind mount real de `/app/data`**, o `DOCKGE_STACKS_DIR` da instalação em execução, o source do mount de stacks e se o caminho informado em `--stacks-path` coincide com o runtime atual.

Antes de aplicar, confirme que o inventário informa:

```text
source_data_mount_type=bind
source_data_mount_source=<caminho real dos dados Dockge>
source_stacks_mount_type=bind
stacks_path_match=true
```

O fluxo automático falha de forma segura quando `/app/data` usa named volume, quando o caminho real das stacks não coincide com o informado, ou quando o mount de stacks não segue o layout bind suportado. Nesses casos, faça uma migração revisada/manual em vez de forçar a automação.

Depois da revisão:

```bash
dockge-deploy dockge migrate \
  --host 203.0.113.10 \
  --user root \
  --path /opt/dockge \
  --stacks-path /opt/stacks \
  --version 1.6.1 \
  --apply
```

A migração:

```text
inventaria origem + mounts reais
→ snapshot Compose/.env/data real/imagem/lista de stacks
→ para somente o container Dockge
→ preserva o bind real de /app/data e parâmetros operacionais compatíveis
→ instala a configuração wkarts/dockge
→ pull
→ sobe somente Dockge
→ verifica container + bind de dados + conjunto de stacks
→ commit
```

Em falha, restaura a instalação anterior e o conteúdo do bind de dados registrado no snapshot. Não executa `down -v`, não remove volumes das aplicações e não remove `/opt/stacks`.

Os quatro nomes Compose padrão `.yaml`/`.yml` são reconhecidos.

### Upgrade de uma instalação já canônica

`dockge upgrade` também detecta o mount real de `/app/data`, grava esse caminho no backup e restaura o conteúdo desse mesmo bind em caso de rollback. O upgrade automático aceita o layout canônico com `/app/data` em bind mount e recusa layouts de named volume sem plano revisado.

---

## 4. Criar token dedicado ao Manager

Planejar:

```bash
dockge-deploy dockge manager-token \
  --host 203.0.113.10 \
  --user root
```

Criar:

```bash
dockge-deploy dockge manager-token \
  --host 203.0.113.10 \
  --user root \
  --apply
```

Copie o segredo `dkg_...` exibido uma única vez.

Para rotacionar o mesmo perfil:

```bash
dockge-deploy dockge manager-token \
  --host 203.0.113.10 \
  --user root \
  --replace \
  --apply
```

---

## 5. Instalar Dockge Manager

A imagem estável é:

```text
ghcr.io/wkarts/dockge-manager:1.0.1
```

Prepare o diretório:

```bash
git clone https://github.com/wkarts/dockge.git
cd dockge/dockge-manager
cp .env.example .env
```

Gere segredos fortes:

```bash
python - <<'PY'
import base64, os, secrets
print('DOCKGE_MANAGER_DB_PASSWORD=' + secrets.token_urlsafe(32))
print('DOCKGE_MANAGER_JWT_SECRET=' + secrets.token_urlsafe(64))
print('DOCKGE_MANAGER_FERNET_KEY=' + base64.urlsafe_b64encode(os.urandom(32)).decode())
print('DOCKGE_MANAGER_ADMIN_PASSWORD=' + secrets.token_urlsafe(32))
PY
```

Preencha `.env` e fixe a imagem:

```text
DOCKGE_MANAGER_TAG=1.0.1
```

Suba:

```bash
docker compose pull
docker compose up -d
```

Valide:

```bash
docker compose ps
docker compose logs --tail=200 manager
```

A porta padrão é `127.0.0.1:8010`. Para uso remoto, publique o Manager por reverse proxy HTTPS.

---

## 6. Cadastrar o Dockge no Manager

Na PWA:

```text
Infraestrutura
→ Novo Target
```

Informe:

```text
Nome      produção-01
URL       https://dockge.exemplo.com
Token     dkg_...
TLS       validar
```

Depois use **Testar conexão**.

O token fica criptografado no PostgreSQL do Manager e não é devolvido integralmente à UI.

---

## 7. Deploy de aplicação pelo Manager

Fluxo:

```text
Aplicações
→ cadastrar aplicação
→ Novo Deployment
→ escolher Dockge Target
→ informar nome da stack
→ informar compose.yaml
→ informar .env quando necessário
→ Criar DRAFT
→ revisar
→ Deploy
```

Execução interna:

```text
SNAPSHOT do runtime atual
→ APPLY pela Dockge Automation API + Idempotency-Key
→ se a resposta se perder: retry com a MESMA chave
→ UP
→ VERIFY containers/health
→ HEALTHY
```

Se uma mutação falhar depois de ter sido aplicada **ou se o resultado permanecer incerto**:

```text
ROLLING_BACK
→ reconcilia o estado real com o snapshot anterior
→ restaura somente quando necessário
→ verifica novamente
→ ROLLED_BACK
```

Quando o Core responder `idempotency_result_in_doubt`, a operação fica registrada como `IN_DOUBT`. O Manager não cria uma segunda intenção com outra chave para “tentar de novo”.

Se a stack não existia antes, a recuperação só remove uma stack que agora exista e esteja marcada como API-managed. Uma stack externa sem esse marcador nunca é apagada automaticamente, e named volumes não são removidos.

---

## 8. Uso direto da Automation API pelo Dockge Deploy

Defina o token no ambiente:

```bash
export DOCKGE_DEPLOY_DOCKGE_TOKEN='dkg_...'
```

Exemplos:

```bash
dockge-deploy stack list --url https://dockge.exemplo.com

dockge-deploy stack logs \
  --url https://dockge.exemplo.com \
  --name connect-api \
  --tail 300

dockge-deploy stack restart \
  --url https://dockge.exemplo.com \
  --name connect-api
```

O último comando exibe o plano. Para executar:

```bash
dockge-deploy stack restart \
  --url https://dockge.exemplo.com \
  --name connect-api \
  --apply
```

---

## 9. Fronteiras que não podem ser quebradas

```text
Dockge Deploy → SSH apenas para host/Dockge lifecycle
Dockge Manager → HTTPS Automation API
Dockge Core → Docker/Compose/stacks
Native Agents → tecnologia interna preservada
```

Nunca introduzir no Manager:

```text
/var/run/docker.sock
acesso direto ao banco do Dockge
escrita direta em /opt/stacks
uso do AgentManager nativo como API externa
```
