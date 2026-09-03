#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
VERSION="$(tr -d '[:space:]' < VERSION)"
ARCH="${1:-amd64}"
case "$ARCH" in
  amd64) RPMARCH=x86_64 ;;
  arm64) RPMARCH=aarch64 ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 2 ;;
esac
command -v rpmbuild >/dev/null || { echo "rpmbuild is required" >&2; exit 3; }
TOP="$(mktemp -d)"
trap 'rm -rf "$TOP"' EXIT
mkdir -p "$TOP"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
cp "dist/bin/infra-agent-linux-$ARCH" "$TOP/SOURCES/infra-agent"
cp scripts/install.sh "$TOP/SOURCES/infra-agent-installer"
cp scripts/bootstrap-host.sh "$TOP/SOURCES/bootstrap-host.sh"
cp examples/agent.json "$TOP/SOURCES/agent.json.example"
cp packaging/systemd/infrastructure-agent.service "$TOP/SOURCES/"
cp packaging/rpm/infrastructure-agent.spec "$TOP/SPECS/"
rpmbuild --define "_topdir $TOP" --define "agent_version $VERSION" --target "$RPMARCH" -bb "$TOP/SPECS/infrastructure-agent.spec"
mkdir -p dist/packages
cp "$TOP"/RPMS/*/*.rpm dist/packages/
