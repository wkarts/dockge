#!/usr/bin/env bash
# Library sourced by install.sh. It intentionally does not execute on import.

BOOTSTRAP_DOCKGE_URL="${BOOTSTRAP_DOCKGE_URL:-}"
BOOTSTRAP_DOCKGE_TOKEN="${BOOTSTRAP_DOCKGE_TOKEN:-}"
BOOTSTRAP_DOCKGE_CONTAINER="${BOOTSTRAP_DOCKGE_CONTAINER:-}"

bootstrap_have_docker() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

bootstrap_have_compose() {
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1
}

bootstrap_install_docker() {
  if bootstrap_have_docker && bootstrap_have_compose; then
    ok "Docker Engine e Compose v2 já estão disponíveis."
    return 0
  fi

  need_root
  [[ -r /etc/os-release ]] || { fail "Não foi possível identificar a distribuição Linux."; return 1; }
  # shellcheck disable=SC1091
  . /etc/os-release
  local distro="${ID:-}"
  local codename="${VERSION_CODENAME:-}"

  info "Instalando Docker Engine pelo repositório oficial para ${distro:-Linux}..."
  case "$distro" in
    debian|ubuntu)
      command -v apt-get >/dev/null 2>&1 || { fail "apt-get não encontrado."; return 1; }
      apt-get update
      DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl gnupg
      install -m 0755 -d /etc/apt/keyrings
      curl -fsSL "https://download.docker.com/linux/$distro/gpg" -o /etc/apt/keyrings/docker.asc
      chmod a+r /etc/apt/keyrings/docker.asc
      [[ -n "$codename" ]] || { fail "VERSION_CODENAME não encontrado em /etc/os-release."; return 1; }
      printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/%s %s stable\n' \
        "$(dpkg --print-architecture)" "$distro" "$codename" > /etc/apt/sources.list.d/docker.list
      apt-get update
      DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
      ;;
    fedora|centos|rhel|rocky|almalinux)
      command -v dnf >/dev/null 2>&1 || { fail "dnf não encontrado."; return 1; }
      dnf -y install dnf-plugins-core ca-certificates curl
      local repo_family="$distro"
      case "$distro" in
        rhel|rocky|almalinux) repo_family="centos" ;;
      esac
      if dnf config-manager --help 2>&1 | grep -q -- '--add-repo'; then
        dnf config-manager --add-repo "https://download.docker.com/linux/$repo_family/docker-ce.repo"
      else
        dnf config-manager addrepo --from-repofile="https://download.docker.com/linux/$repo_family/docker-ce.repo"
      fi
      dnf -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
      ;;
    *)
      fail "Distribuição '$distro' ainda não possui bootstrap automático. Instale Docker Engine + Compose v2 e execute o instalador novamente."
      return 2
      ;;
  esac

  if command -v systemctl >/dev/null 2>&1; then
    systemctl enable --now docker
  fi
  bootstrap_have_docker || { fail "Docker foi instalado, mas o daemon não está disponível."; return 3; }
  bootstrap_have_compose || { fail "Docker Compose v2 não ficou disponível."; return 4; }
  ok "Docker Engine e Compose v2 prontos."
}

bootstrap_list_dockge() {
  bootstrap_have_docker || return 0
  docker ps -a --format '{{.Names}}\t{{.Image}}\t{{.State}}\t{{.Ports}}' 2>/dev/null | awk 'BEGIN{IGNORECASE=1} /dockge/' || true
}

bootstrap_wkarts_dockge_container() {
  bootstrap_have_docker || return 1
  docker ps -a --format '{{.Names}}\t{{.Image}}' 2>/dev/null | awk 'BEGIN{IGNORECASE=1} $2 ~ /ghcr\.io\/wkarts\/dockge/ {print $1; exit}'
}

bootstrap_port_available() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ! ss -ltnH 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]${port}$"
    return
  fi
  ! curl -fsS --max-time 1 "http://127.0.0.1:$port/" >/dev/null 2>&1
}

bootstrap_suggest_port() {
  local port=5001
  while ! bootstrap_port_available "$port"; do
    port=$((port + 1))
    [[ "$port" -le 5099 ]] || { echo 0; return 1; }
  done
  echo "$port"
}

bootstrap_wait_container() {
  local container="$1"
  local attempts=60
  while (( attempts > 0 )); do
    if [[ "$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)" == "true" ]]; then
      return 0
    fi
    sleep 1
    attempts=$((attempts - 1))
  done
  return 1
}

bootstrap_create_agent_token() {
  local container="$1"
  local prefixes="$2"
  [[ -n "$prefixes" ]] || { fail "Informe pelo menos um prefixo de deployment para a credencial do Agent."; return 1; }

  local token
  token="$(docker exec "$container" npm run --silent api-token:create -- \
    --name "infrastructure-agent-$(hostname -s 2>/dev/null || hostname)" \
    --scopes 'server:read,stacks:read,stacks:write,stacks:delete,stacks:operate,stacks:adopt' \
    --prefixes "$prefixes" \
    --token-only 2>/dev/null)" || {
      fail "Não foi possível criar a credencial local do Agent dentro do Dockge."
      return 1
    }
  [[ "$token" == dkg_* ]] || { fail "Dockge retornou uma credencial em formato inesperado."; return 1; }
  BOOTSTRAP_DOCKGE_TOKEN="$token"
}

bootstrap_existing_wkarts_dockge() {
  local container
  container="$(bootstrap_wkarts_dockge_container || true)"
  [[ -n "$container" ]] || return 1

  info "Dockge API-first deste projeto detectado: $container"
  local state
  state="$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)"
  if [[ "$state" != "true" ]]; then
    read -r -p "O container está parado. Iniciar '$container'? [S/n]: " answer
    if [[ "${answer,,}" != "n" && "${answer,,}" != "nao" && "${answer,,}" != "não" ]]; then
      docker start "$container" >/dev/null
    else
      return 1
    fi
  fi
  bootstrap_wait_container "$container" || { fail "Dockge não iniciou dentro do tempo esperado."; return 1; }

  local host_port
  host_port="$(docker port "$container" 5001/tcp 2>/dev/null | head -n1 | sed -E 's/.*:([0-9]+)$/\1/' || true)"
  [[ "$host_port" =~ ^[0-9]+$ ]] || host_port=5001
  BOOTSTRAP_DOCKGE_URL="http://127.0.0.1:$host_port"
  BOOTSTRAP_DOCKGE_CONTAINER="$container"
  return 0
}

bootstrap_deploy_wkarts_dockge() {
  need_root
  bootstrap_have_docker && bootstrap_have_compose || {
    fail "Docker Engine + Compose v2 são obrigatórios antes de instalar Dockge."
    return 1
  }

  local suggested_port port tag root_dir stacks_dir container
  suggested_port="$(bootstrap_suggest_port)" || true
  [[ "$suggested_port" != "0" && -n "$suggested_port" ]] || { fail "Não encontrei porta livre entre 5001 e 5099."; return 1; }
  port="$(prompt_default 'Porta local para o Dockge API-first' "$suggested_port")"
  [[ "$port" =~ ^[0-9]+$ ]] || { fail "Porta inválida."; return 1; }
  bootstrap_port_available "$port" || { fail "A porta $port já está em uso."; return 1; }

  tag="$(prompt_default 'Canal/tag da imagem ghcr.io/wkarts/dockge' 'latest')"
  root_dir="$(prompt_default 'Diretório persistente deste Dockge' '/opt/wkarts-dockge')"
  stacks_dir="$(prompt_default 'Diretório de stacks gerenciadas por esta instância' '/opt/wkarts-stacks')"
  container="wkarts-dockge"
  if docker inspect "$container" >/dev/null 2>&1; then
    container="wkarts-dockge-$port"
  fi

  info "Preservando qualquer Dockge existente e criando instância independente em $root_dir."
  install -d -m 0750 "$root_dir" "$root_dir/data" "$stacks_dir"
  cat > "$root_dir/compose.yaml" <<COMPOSE
services:
  dockge:
    container_name: $container
    image: ghcr.io/wkarts/dockge:$tag
    restart: unless-stopped
    ports:
      - "127.0.0.1:$port:5001"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - $root_dir/data:/app/data
      - $stacks_dir:$stacks_dir
    environment:
      DOCKGE_STACKS_DIR: $stacks_dir
      DOCKGE_API_TOKENS_FILE: /app/data/api-tokens.json
      DOCKGE_API_AUDIT_FILE: /app/data/api-audit.jsonl
      DOCKGE_ENABLE_CONSOLE: "false"
      DOCKGE_ALLOW_DISABLE_AUTH: "false"
      DOCKGE_TOTP_ISSUER: Dockge
    security_opt:
      - no-new-privileges:true
COMPOSE
  chmod 0640 "$root_dir/compose.yaml"

  info "Baixando imagem $tag sem modificar containers externos..."
  if ! docker compose -f "$root_dir/compose.yaml" pull; then
    fail "A imagem não pôde ser baixada. O Dockge existente não foi alterado; o compose preparado ficou em $root_dir/compose.yaml."
    return 2
  fi
  docker compose -f "$root_dir/compose.yaml" up -d
  bootstrap_wait_container "$container" || { fail "A nova instância não iniciou dentro do tempo esperado."; return 3; }

  BOOTSTRAP_DOCKGE_URL="http://127.0.0.1:$port"
  BOOTSTRAP_DOCKGE_CONTAINER="$container"
  ok "Dockge API-first iniciado em $BOOTSTRAP_DOCKGE_URL (loopback)."
}

bootstrap_prepare_dockge_for_agent() {
  local existing
  existing="$(bootstrap_list_dockge)"
  if [[ -n "$existing" ]]; then
    echo
    say "${BOLD}Dockge(s) detectado(s) no host:${RESET}"
    printf '%s\n' "$existing"
    echo
    warn "Nenhuma instalação existente será removida, atualizada ou adotada automaticamente."
  fi

  if bootstrap_existing_wkarts_dockge; then
    read -r -p "Usar esta instância API-first para o Agent? [S/n]: " answer
    if [[ "${answer,,}" != "n" && "${answer,,}" != "nao" && "${answer,,}" != "não" ]]; then
      return 0
    fi
    BOOTSTRAP_DOCKGE_URL=""
    BOOTSTRAP_DOCKGE_CONTAINER=""
  fi

  read -r -p "Instalar uma nova instância Dockge API-first em coexistência? [S/n]: " answer
  if [[ "${answer,,}" == "n" || "${answer,,}" == "nao" || "${answer,,}" == "não" ]]; then
    warn "Dockge não será alterado. Você poderá informar manualmente a URL/credencial durante a configuração do Agent."
    return 0
  fi
  bootstrap_deploy_wkarts_dockge
}
