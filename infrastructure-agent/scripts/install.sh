#!/usr/bin/env bash
set -euo pipefail

APP_NAME="Generic Infrastructure Agent"
SERVICE_NAME="infrastructure-agent"
PREFIX="${INFRA_AGENT_PREFIX:-/usr}"
CONFIG_DIR="${INFRA_AGENT_CONFIG_DIR:-/etc/infrastructure-agent}"
DATA_DIR="${INFRA_AGENT_DATA_DIR:-/var/lib/infrastructure-agent}"
CONFIG_FILE="$CONFIG_DIR/agent.json"
SYSTEMD_UNIT="/etc/systemd/system/${SERVICE_NAME}.service"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

if [[ -t 1 ]]; then
  BOLD='\033[1m'; DIM='\033[2m'; GREEN='\033[32m'; CYAN='\033[36m'; YELLOW='\033[33m'; RED='\033[31m'; RESET='\033[0m'
else
  BOLD=''; DIM=''; GREEN=''; CYAN=''; YELLOW=''; RED=''; RESET=''
fi

say()  { printf '%b\n' "$*"; }
ok()   { say "${GREEN}✓${RESET} $*"; }
info() { say "${CYAN}●${RESET} $*"; }
warn() { say "${YELLOW}!${RESET} $*"; }
fail() { say "${RED}✕${RESET} $*" >&2; }
line() { printf '%*s\n' 72 '' | tr ' ' '─'; }

banner() {
  clear 2>/dev/null || true
  say "${BOLD}╔══════════════════════════════════════════════════════════════════════╗${RESET}"
  say "${BOLD}║                GENERIC INFRASTRUCTURE AGENT                        ║${RESET}"
  say "${BOLD}║       Bootstrap • Enrollment • Docker/Dockge • Control Plane       ║${RESET}"
  say "${BOLD}╚══════════════════════════════════════════════════════════════════════╝${RESET}"
  say "${DIM}Agente genérico para múltiplos Control Planes e plataformas.${RESET}"
  echo
}

need_root() {
  if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    fail "Esta operação requer privilégios administrativos."
    say "Execute novamente com: ${BOLD}sudo $0${RESET}"
    exit 1
  fi
}

prompt_default() {
  local prompt="$1" default="$2" value
  if [[ -n "$default" ]]; then
    read -r -p "$prompt [$default]: " value
    printf '%s' "${value:-$default}"
  else
    read -r -p "$prompt: " value
    printf '%s' "$value"
  fi
}

BOOTSTRAP_LIB=""
for candidate in \
  "$SCRIPT_DIR/bootstrap-host.sh" \
  "$ROOT_DIR/scripts/bootstrap-host.sh" \
  "/usr/lib/infrastructure-agent/bootstrap-host.sh"; do
  if [[ -f "$candidate" ]]; then
    BOOTSTRAP_LIB="$candidate"
    break
  fi
done
if [[ -n "$BOOTSTRAP_LIB" ]]; then
  # shellcheck disable=SC1090
  source "$BOOTSTRAP_LIB"
fi

arch_name() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) fail "Arquitetura não suportada: $(uname -m)"; exit 2 ;;
  esac
}

find_binary() {
  local arch candidate
  arch="$(arch_name)"
  for candidate in \
    "$ROOT_DIR/dist/bin/infra-agent-linux-$arch" \
    "$SCRIPT_DIR/infra-agent-linux-$arch" \
    "$SCRIPT_DIR/../infra-agent-linux-$arch" \
    "./infra-agent-linux-$arch"; do
    [[ -f "$candidate" ]] && { printf '%s' "$candidate"; return 0; }
  done
  command -v infra-agent 2>/dev/null || return 1
}

status_line() {
  local label="$1" value="$2"
  printf '  %-24s %s\n' "$label" "$value"
}

show_environment() {
  banner
  say "${BOLD}Diagnóstico rápido do host${RESET}"
  line
  status_line "Sistema" "$(uname -s) $(uname -r)"
  status_line "Arquitetura" "$(uname -m)"
  status_line "Docker" "$(docker --version 2>/dev/null || echo 'não detectado')"
  status_line "Compose" "$(docker compose version 2>/dev/null || echo 'não detectado')"
  if command -v infra-agent >/dev/null 2>&1; then
    status_line "Agent" "$(infra-agent version 2>/dev/null || echo instalado)"
  else
    status_line "Agent" "não instalado"
  fi
  if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files "$SERVICE_NAME.service" >/dev/null 2>&1; then
    status_line "Serviço Agent" "$(systemctl is-active "$SERVICE_NAME.service" 2>/dev/null || true)"
  else
    status_line "Serviço Agent" "não registrado"
  fi
  status_line "Configuração" "$([[ -f "$CONFIG_FILE" ]] && echo "$CONFIG_FILE" || echo 'pendente')"

  if [[ -n "$BOOTSTRAP_LIB" ]] && command -v docker >/dev/null 2>&1; then
    local dockges count
    dockges="$(bootstrap_list_dockge || true)"
    if [[ -n "$dockges" ]]; then
      count="$(printf '%s\n' "$dockges" | grep -c . || true)"
      status_line "Dockge containers" "$count detectado(s) — nenhum será alterado automaticamente"
    else
      status_line "Dockge containers" "nenhum detectado"
    fi
  else
    status_line "Dockge containers" "inventário indisponível"
  fi
  line
}

install_files() {
  need_root
  local binary
  binary="$(find_binary)" || { fail "Binário do Agent não encontrado neste pacote."; return 3; }
  info "Instalando binário e preparando dados persistentes..."
  install -d -m 0750 "$CONFIG_DIR" "$CONFIG_DIR/secrets"
  install -d -m 0700 "$DATA_DIR"
  install -m 0755 "$binary" "$PREFIX/bin/infra-agent"
  if [[ -f "$ROOT_DIR/packaging/systemd/infrastructure-agent.service" ]]; then
    install -m 0644 "$ROOT_DIR/packaging/systemd/infrastructure-agent.service" "$SYSTEMD_UNIT"
  fi
  if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME.service" >/dev/null
    ok "Serviço systemd registrado."
  else
    warn "systemd não detectado; o Agent poderá ser executado manualmente."
  fi
  ok "Agent instalado em $PREFIX/bin/infra-agent"
}

prepare_runtime() {
  need_root
  [[ -n "$BOOTSTRAP_LIB" ]] || {
    warn "Biblioteca de bootstrap não está presente neste pacote; Docker/Dockge não serão alterados."
    return 0
  }

  if ! bootstrap_have_docker || ! bootstrap_have_compose; then
    warn "Docker Engine e/ou Compose v2 não estão prontos."
    read -r -p 'Instalar/configurar Docker pelo repositório oficial agora? [S/n]: ' answer
    if [[ "${answer,,}" != "n" && "${answer,,}" != "nao" && "${answer,,}" != "não" ]]; then
      bootstrap_install_docker
    else
      warn "Docker não será instalado. O Agent pode ser configurado agora e o runtime concluído posteriormente."
      return 0
    fi
  fi

  bootstrap_prepare_dockge_for_agent
}

configure_agent() {
  need_root
  command -v infra-agent >/dev/null 2>&1 || { fail "Instale o Agent antes de configurar."; return 1; }
  banner
  say "${BOLD}Configuração e vínculo com Control Plane${RESET}"
  say "${DIM}Adicionar um novo vínculo preserva todos os Control Planes já configurados.${RESET}"
  say "${DIM}Cada vínculo recebe sua própria credencial local Dockge e seus próprios namespaces.${RESET}"
  line

  local controller url prefixes dockge_url enrollment dockge_credential environment
  controller="$(prompt_default 'Nome lógico deste Control Plane' 'control-plane')"
  url="$(prompt_default 'URL REST/HTTPS do Control Plane' 'https://api.example.com')"
  prefixes="$(prompt_default 'Prefixos de deployments permitidos (separados por vírgula)' '')"
  dockge_url="$(prompt_default 'Dockge API local' "${BOOTSTRAP_DOCKGE_URL:-http://127.0.0.1:5001}")"
  environment="$(prompt_default 'Ambiente/label do host' 'production')"

  echo
  read -r -s -p 'Credencial de enrollment do Control Plane (não será exibida): ' enrollment
  echo

  dockge_credential=""
  if [[ -n "${BOOTSTRAP_DOCKGE_CONTAINER:-}" && -n "$prefixes" && -n "$BOOTSTRAP_LIB" ]]; then
    read -r -p "Criar/rotacionar a credencial Dockge exclusiva deste Control Plane em '$BOOTSTRAP_DOCKGE_CONTAINER'? [S/n]: " answer
    if [[ "${answer,,}" != "n" && "${answer,,}" != "nao" && "${answer,,}" != "não" ]]; then
      if bootstrap_create_agent_token "$BOOTSTRAP_DOCKGE_CONTAINER" "$controller" "$prefixes"; then
        dockge_credential="$BOOTSTRAP_DOCKGE_TOKEN"
        ok "Credencial local exclusiva do vínculo '$controller' criada/rotacionada."
      fi
    fi
  fi

  if [[ -z "$dockge_credential" ]]; then
    read -r -s -p 'Credencial da Dockge API deste vínculo (Enter para preservar a já configurada): ' dockge_credential
    echo
  fi

  INFRA_AGENT_CONTROLLER_NAME="$controller" \
  INFRA_AGENT_CONTROLLER_URL="$url" \
  INFRA_AGENT_ALLOWED_PREFIXES="$prefixes" \
  INFRA_AGENT_DOCKGE_URL="$dockge_url" \
  INFRA_AGENT_ENROLLMENT_TOKEN="$enrollment" \
  INFRA_AGENT_DOCKGE_TOKEN="$dockge_credential" \
  INFRA_AGENT_ENVIRONMENT="$environment" \
  INFRA_AGENT_DATA_DIR="$DATA_DIR" \
  infra-agent --config "$CONFIG_FILE" configure

  enrollment=""
  dockge_credential=""
  BOOTSTRAP_DOCKGE_TOKEN=""

  ok "Configuração criada; credenciais ficam separadas do agent.json e isoladas por Control Plane."
  if [[ -f "$CONFIG_DIR/secrets/$controller-enrollment.credential" ]]; then
    info "Realizando enrollment no Control Plane..."
    if infra-agent --config "$CONFIG_FILE" enroll; then
      ok "Enrollment concluído; credencial bootstrap removida pelo Agent."
    else
      warn "Enrollment não concluiu. O serviço tentará novamente nos próximos ciclos."
    fi
  else
    warn "Nenhuma credencial de enrollment foi gravada; o vínculo remoto permanece pendente."
  fi
}

start_service() {
  need_root
  if command -v systemctl >/dev/null 2>&1; then
    systemctl restart "$SERVICE_NAME.service"
    sleep 1
    systemctl --no-pager --full status "$SERVICE_NAME.service" || true
  else
    warn "systemd não disponível. Use: infra-agent --config '$CONFIG_FILE' run"
  fi
}

doctor() {
  if ! command -v infra-agent >/dev/null 2>&1; then
    fail "Agent não instalado."
    return 1
  fi
  infra-agent --config "$CONFIG_FILE" doctor
}

uninstall_agent() {
  need_root
  banner
  warn "O Agent será removido, mas configurações e dados serão preservados."
  warn "Docker, Dockge e stacks NÃO serão removidos por esta operação."
  read -r -p 'Continuar? [s/N]: ' choice
  [[ "${choice,,}" == "s" || "${choice,,}" == "sim" ]] || return 0
  if command -v systemctl >/dev/null 2>&1; then
    systemctl disable --now "$SERVICE_NAME.service" >/dev/null 2>&1 || true
    rm -f "$SYSTEMD_UNIT"
    systemctl daemon-reload || true
  fi
  rm -f "$PREFIX/bin/infra-agent"
  ok "Binário/serviço do Agent removidos."
  say "Dados preservados em: $CONFIG_DIR e $DATA_DIR"
}

full_install() {
  install_files
  prepare_runtime
  configure_agent
  start_service
  echo
  ok "Instalação concluída."
}

usage() {
  cat <<USAGE
Uso: $0 [--install | --bootstrap-host | --configure | --doctor | --uninstall | --status]

  --install         instala/repara somente o Agent
  --bootstrap-host  prepara Docker/Dockge com escolhas interativas
  --configure       adiciona/atualiza vínculo de Control Plane
  --doctor          diagnóstico do Agent/runtime
  --uninstall       remove somente o Agent, preservando dados e Docker/Dockge
  --status          inventário rápido

Sem argumentos abre o assistente interativo completo.
USAGE
}

case "${1:-}" in
  --install) install_files; exit ;;
  --bootstrap-host) prepare_runtime; exit ;;
  --configure) configure_agent; exit ;;
  --doctor) doctor; exit ;;
  --uninstall) uninstall_agent; exit ;;
  --status) show_environment; exit ;;
  -h|--help) usage; exit ;;
esac

while true; do
  show_environment
  say "${BOLD}Escolha uma operação:${RESET}"
  say "  ${CYAN}1${RESET}) Instalação completa guiada (Agent + runtime + vínculo)"
  say "  ${CYAN}2${RESET}) Preparar Docker/Dockge no host (preserva instalações existentes)"
  say "  ${CYAN}3${RESET}) Instalar/Reparar somente o Agent"
  say "  ${CYAN}4${RESET}) Configurar/Adicionar Control Plane"
  say "  ${CYAN}5${RESET}) Diagnóstico detalhado"
  say "  ${CYAN}6${RESET}) Iniciar/Reiniciar serviço"
  say "  ${CYAN}7${RESET}) Desinstalar somente o Agent (preserva dados/runtime)"
  say "  ${CYAN}0${RESET}) Sair"
  echo
  read -r -p 'Opção: ' choice
  case "$choice" in
    1) full_install; read -r -p 'Pressione Enter para continuar...' _ ;;
    2) prepare_runtime; read -r -p 'Pressione Enter para continuar...' _ ;;
    3) install_files; read -r -p 'Pressione Enter para continuar...' _ ;;
    4) configure_agent; read -r -p 'Pressione Enter para continuar...' _ ;;
    5) banner; doctor || true; read -r -p 'Pressione Enter para continuar...' _ ;;
    6) start_service; read -r -p 'Pressione Enter para continuar...' _ ;;
    7) uninstall_agent; read -r -p 'Pressione Enter para continuar...' _ ;;
    0) exit 0 ;;
    *) warn "Opção inválida."; sleep 1 ;;
  esac
done
