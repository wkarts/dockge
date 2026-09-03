#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"; cd "$ROOT"
VERSION="$(tr -d '[:space:]' < VERSION)"
ARCH="${1:-$(uname -m)}"
case "$ARCH" in arm64) BINARCH=arm64;; x86_64|amd64) ARCH=x86_64; BINARCH=amd64;; *) echo "unsupported architecture: $ARCH" >&2; exit 2;; esac
BIN="dist/bin/infra-agent-darwin-$BINARCH"
[[ -f "$BIN" ]] || { echo "missing $BIN" >&2; exit 3; }
command -v pkgbuild >/dev/null || { echo 'pkgbuild is required' >&2; exit 4; }
PKGROOT="$(mktemp -d)"; SCRIPTS="$(mktemp -d)"; trap 'rm -rf "$PKGROOT" "$SCRIPTS"' EXIT
mkdir -p "$PKGROOT/usr/local/bin" "$PKGROOT/Library/LaunchDaemons" "$PKGROOT/Library/Application Support/InfrastructureAgent/secrets" "$PKGROOT/Library/Application Support/InfrastructureAgent/data"
install -m 0755 "$BIN" "$PKGROOT/usr/local/bin/infra-agent"
install -m 0644 packaging/macos/com.infrastructure.agent.plist "$PKGROOT/Library/LaunchDaemons/com.infrastructure.agent.plist"
install -m 0755 packaging/macos/install.sh "$PKGROOT/usr/local/bin/infra-agent-installer"
cat > "$SCRIPTS/postinstall" <<'POST'
#!/bin/bash
set -e
CONFIG_DIR='/Library/Application Support/InfrastructureAgent'
mkdir -p "$CONFIG_DIR/secrets" "$CONFIG_DIR/data" /Library/Logs
chmod 750 "$CONFIG_DIR" || true
chmod 700 "$CONFIG_DIR/secrets" "$CONFIG_DIR/data" || true
chown root:wheel /Library/LaunchDaemons/com.infrastructure.agent.plist /usr/local/bin/infra-agent /usr/local/bin/infra-agent-installer || true
launchctl bootout system /Library/LaunchDaemons/com.infrastructure.agent.plist >/dev/null 2>&1 || true
if [ -f "$CONFIG_DIR/agent.json" ]; then launchctl bootstrap system /Library/LaunchDaemons/com.infrastructure.agent.plist || true; fi
printf '\nGeneric Infrastructure Agent instalado. Execute:\n  sudo infra-agent-installer\n\n'
POST
chmod 0755 "$SCRIPTS/postinstall"
mkdir -p dist/packages
pkgbuild --root "$PKGROOT" --scripts "$SCRIPTS" --identifier com.infrastructure.agent --version "$VERSION" --install-location / "dist/packages/infrastructure-agent_${VERSION}_macos_${BINARCH}.pkg"
