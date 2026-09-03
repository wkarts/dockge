#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
VERSION="$(tr -d '[:space:]' < VERSION)"
ARCH="${1:-amd64}"
BIN="dist/bin/infra-agent-linux-$ARCH"
PKG="dist/pkg/deb-$ARCH"
rm -rf "$PKG"
mkdir -p \
  "$PKG/DEBIAN" \
  "$PKG/usr/bin" \
  "$PKG/usr/lib/infrastructure-agent" \
  "$PKG/etc/infrastructure-agent" \
  "$PKG/lib/systemd/system" \
  "$PKG/var/lib/infrastructure-agent" \
  dist/packages
chmod g-s "$PKG/DEBIAN"
chmod 0755 "$PKG/DEBIAN"
cp "$BIN" "$PKG/usr/bin/infra-agent"; chmod 0755 "$PKG/usr/bin/infra-agent"
cp scripts/install.sh "$PKG/usr/bin/infra-agent-installer"; chmod 0755 "$PKG/usr/bin/infra-agent-installer"
cp scripts/bootstrap-host.sh "$PKG/usr/lib/infrastructure-agent/bootstrap-host.sh"; chmod 0644 "$PKG/usr/lib/infrastructure-agent/bootstrap-host.sh"
cp examples/agent.json "$PKG/etc/infrastructure-agent/agent.json.example"
cp packaging/systemd/infrastructure-agent.service "$PKG/lib/systemd/system/"
cat > "$PKG/DEBIAN/control" <<CTRL
Package: infrastructure-agent
Version: $VERSION
Section: admin
Priority: optional
Architecture: $ARCH
Maintainer: wkarts
Depends: ca-certificates, curl
Description: Generic REST infrastructure enrollment and deployment agent
 Cross-platform infrastructure agent for scoped Control Plane integration.
 Includes an optional interactive Linux host bootstrap for Docker/Dockge.
CTRL
cat > "$PKG/DEBIAN/postinst" <<'POST'
#!/bin/sh
set -e
mkdir -p /etc/infrastructure-agent/secrets /var/lib/infrastructure-agent
chmod 0750 /etc/infrastructure-agent || true
chmod 0700 /etc/infrastructure-agent/secrets /var/lib/infrastructure-agent || true
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload || true
  systemctl enable infrastructure-agent.service || true
fi
printf '\n╔══════════════════════════════════════════════════════════════╗\n'
printf '║ Generic Infrastructure Agent instalado                     ║\n'
printf '╚══════════════════════════════════════════════════════════════╝\n'
printf 'Próximo passo: sudo infra-agent-installer\n'
printf 'O assistente pode preservar runtime atual ou preparar Docker/Dockge.\n\n'
POST
cat > "$PKG/DEBIAN/prerm" <<'PRE'
#!/bin/sh
if command -v systemctl >/dev/null 2>&1; then systemctl disable --now infrastructure-agent.service >/dev/null 2>&1 || true; fi
exit 0
PRE
chmod 0755 "$PKG/DEBIAN/postinst" "$PKG/DEBIAN/prerm"
dpkg-deb --root-owner-group --build "$PKG" "dist/packages/infrastructure-agent_${VERSION}_${ARCH}.deb"
