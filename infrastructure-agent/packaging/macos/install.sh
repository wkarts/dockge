#!/bin/bash
set -euo pipefail

APP='Generic Infrastructure Agent'
CONFIG_DIR='/Library/Application Support/InfrastructureAgent'
DATA_DIR="$CONFIG_DIR/data"
CONFIG_FILE="$CONFIG_DIR/agent.json"
BIN='/usr/local/bin/infra-agent'
PLIST='/Library/LaunchDaemons/com.infrastructure.agent.plist'
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ -t 1 ]]; then
  BOLD='\033[1m'; GREEN='\033[32m'; CYAN='\033[36m'; YELLOW='\033[33m'; RED='\033[31m'; RESET='\033[0m'
else BOLD=''; GREEN=''; CYAN=''; YELLOW=''; RED=''; RESET=''; fi
say(){ printf '%b\n' "$*"; }
ok(){ say "${GREEN}✓${RESET} $*"; }
warn(){ say "${YELLOW}!${RESET} $*"; }
fail(){ say "${RED}✕${RESET} $*" >&2; }
banner(){ clear 2>/dev/null || true; say "${BOLD}╔══════════════════════════════════════════════════════════════╗${RESET}"; say "${BOLD}║          GENERIC INFRASTRUCTURE AGENT — macOS              ║${RESET}"; say "${BOLD}╚══════════════════════════════════════════════════════════════╝${RESET}"; echo; }
root(){ [[ ${EUID:-$(id -u)} -eq 0 ]] || { fail 'Execute com sudo.'; exit 1; }; }

arch(){ case "$(uname -m)" in arm64) echo arm64;; x86_64) echo amd64;; *) fail "Arquitetura não suportada: $(uname -m)"; exit 2;; esac; }
source_binary(){
  local a="$(arch)"
  for f in "$SCRIPT_DIR/infra-agent-darwin-$a" "$SCRIPT_DIR/../../dist/bin/infra-agent-darwin-$a"; do [[ -f "$f" ]] && { echo "$f"; return; }; done
  command -v infra-agent || true
}

status(){
  banner
  say "${BOLD}Diagnóstico rápido${RESET}"
  printf '  %-22s %s\n' 'macOS' "$(sw_vers -productVersion)"
  printf '  %-22s %s\n' 'Arquitetura' "$(uname -m)"
  printf '  %-22s %s\n' 'Docker' "$(docker --version 2>/dev/null || echo 'não detectado')"
  printf '  %-22s %s\n' 'Compose' "$(docker compose version 2>/dev/null || echo 'não detectado')"
  printf '  %-22s %s\n' 'Agent' "$($BIN version 2>/dev/null || echo 'não instalado')"
  if launchctl print system/com.infrastructure.agent >/dev/null 2>&1; then printf '  %-22s %s\n' 'LaunchDaemon' 'carregado'; else printf '  %-22s %s\n' 'LaunchDaemon' 'não carregado'; fi
}

install_agent(){
  root
  local src="$(source_binary)"
  [[ -n "$src" && -f "$src" ]] || { fail 'Binário do Agent não encontrado.'; return 1; }
  mkdir -p "$CONFIG_DIR/secrets" "$DATA_DIR" /Library/Logs
  chmod 750 "$CONFIG_DIR"; chmod 700 "$CONFIG_DIR/secrets" "$DATA_DIR"
  install -m 0755 "$src" "$BIN"
  install -m 0644 "$SCRIPT_DIR/com.infrastructure.agent.plist" "$PLIST"
  chown root:wheel "$PLIST" "$BIN"
  launchctl bootout system "$PLIST" >/dev/null 2>&1 || true
  ok 'Binário e LaunchDaemon instalados.'
}

prompt(){ local p="$1" d="$2" v; read -r -p "$p [$d]: " v; echo "${v:-$d}"; }
configure(){
  root; [[ -x "$BIN" ]] || { fail 'Instale o Agent primeiro.'; return 1; }
  banner; say "${BOLD}Configuração do Control Plane${RESET}"
  local name url prefixes dockge environment enrollment dockgecred
  name="$(prompt 'Nome lógico do Control Plane' 'control-plane')"
  url="$(prompt 'URL REST/HTTPS' 'https://api.example.com')"
  prefixes="$(prompt 'Prefixos permitidos (vírgula)' '')"
  dockge="$(prompt 'Dockge API local' 'http://127.0.0.1:5001')"
  environment="$(prompt 'Ambiente' 'production')"
  read -r -s -p 'Credencial de enrollment: ' enrollment; echo
  read -r -s -p 'Credencial da Dockge API (Enter se pendente): ' dockgecred; echo
  INFRA_AGENT_CONTROLLER_NAME="$name" INFRA_AGENT_CONTROLLER_URL="$url" INFRA_AGENT_ALLOWED_PREFIXES="$prefixes" INFRA_AGENT_DOCKGE_URL="$dockge" INFRA_AGENT_ENVIRONMENT="$environment" INFRA_AGENT_ENROLLMENT_TOKEN="$enrollment" INFRA_AGENT_DOCKGE_TOKEN="$dockgecred" INFRA_AGENT_DATA_DIR="$DATA_DIR" "$BIN" --config "$CONFIG_FILE" configure
  ok 'Configuração gravada com secrets separados.'
  if [[ -n "$enrollment" ]]; then "$BIN" --config "$CONFIG_FILE" enroll && ok 'Enrollment concluído.' || warn 'Enrollment pendente; configuração preservada.'; fi
}

start_agent(){ root; launchctl bootout system "$PLIST" >/dev/null 2>&1 || true; launchctl bootstrap system "$PLIST"; launchctl enable system/com.infrastructure.agent || true; ok 'Agent iniciado.'; }
doctor(){ "$BIN" --config "$CONFIG_FILE" doctor; }
uninstall_agent(){ root; warn 'Configurações e dados serão preservados.'; read -r -p 'Continuar? [s/N]: ' a; [[ "${a,,}" == s || "${a,,}" == sim ]] || return; launchctl bootout system "$PLIST" >/dev/null 2>&1 || true; rm -f "$PLIST" "$BIN"; ok "Agent removido. Dados preservados em $CONFIG_DIR"; }

case "${1:-}" in --install) install_agent; exit;; --configure) configure; exit;; --doctor) doctor; exit;; --start) start_agent; exit;; --uninstall) uninstall_agent; exit;; esac
while true; do
  status; echo
  say '  1) Instalação completa (recomendado)'; say '  2) Instalar/Reparar Agent'; say '  3) Configurar Control Plane'; say '  4) Diagnóstico detalhado'; say '  5) Iniciar/Reiniciar serviço'; say '  6) Desinstalar (preserva dados)'; say '  0) Sair'; echo
  read -r -p 'Opção: ' c
  case "$c" in 1) install_agent; configure; start_agent; break;; 2) install_agent;; 3) configure;; 4) doctor || true;; 5) start_agent;; 6) uninstall_agent;; 0) exit 0;; *) warn 'Opção inválida.';; esac
  read -r -p 'Pressione Enter para continuar...' _
done
