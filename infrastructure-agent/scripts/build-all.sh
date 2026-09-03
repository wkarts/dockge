#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
VERSION="$(tr -d '[:space:]' < VERSION)"
COMMIT="${GITHUB_SHA:-$(git rev-parse --short=12 HEAD 2>/dev/null || echo local)}"
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
mkdir -p dist/bin
LDFLAGS="-s -w -X github.com/wkarts/infrastructure-agent/internal/version.Version=$VERSION -X github.com/wkarts/infrastructure-agent/internal/version.Commit=$COMMIT -X github.com/wkarts/infrastructure-agent/internal/version.Date=$DATE"
build(){ GOOS="$1" GOARCH="$2" CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "dist/bin/$3" ./cmd/infra-agent; }
build linux amd64 infra-agent-linux-amd64
build linux arm64 infra-agent-linux-arm64
build windows amd64 infra-agent-windows-amd64.exe
build windows arm64 infra-agent-windows-arm64.exe
build darwin amd64 infra-agent-darwin-amd64
build darwin arm64 infra-agent-darwin-arm64
(cd dist/bin && sha256sum * > SHA256SUMS)
echo "Built $(find dist/bin -maxdepth 1 -type f | wc -l) files"
