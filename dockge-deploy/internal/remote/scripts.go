package remote

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func InspectScript() string {
	return `set -eu
printf '%s\n' '== host =='
uname -a
printf '%s\n' '== os-release =='
cat /etc/os-release 2>/dev/null || true
printf '%s\n' '== resources =='
df -h / || true
(command -v free >/dev/null 2>&1 && free -h) || true
printf '%s\n' '== docker =='
docker --version 2>/dev/null || true
docker compose version 2>/dev/null || true
`
}

func DetectScript(path string) string {
	p := quote(path)
	return fmt.Sprintf(`set -eu
path=%s
printf 'path=%%s\n' "$path"
if [ -f "$path/compose.yaml" ] || [ -f "$path/docker-compose.yaml" ]; then
  printf '%%s\n' 'dockge_installation=found'
  cd "$path"
  docker compose config >/dev/null 2>&1 && printf '%%s\n' 'compose_config=valid' || printf '%%s\n' 'compose_config=invalid'
  docker compose ps dockge 2>/dev/null || true
else
  printf '%%s\n' 'dockge_installation=not_found'
fi
printf '%%s\n' 'stacks:'
find /opt/stacks -mindepth 1 -maxdepth 1 -type d -printf '  %%f\n' 2>/dev/null | sort || true
`, p)
}

func InstallScript(path, stacksPath, imageTag string, bindHost string, port int) string {
	compose := base64.StdEncoding.EncodeToString([]byte(ComposeYAML))
	env := fmt.Sprintf("DOCKGE_IMAGE=ghcr.io/wkarts/dockge\nDOCKGE_IMAGE_TAG=%s\nDOCKGE_DATA_PATH=./data\nDOCKGE_STACKS_PATH=%s\nDOCKGE_BIND_HOST=%s\nDOCKGE_PORT=%d\n", imageTag, stacksPath, bindHost, port)
	env64 := base64.StdEncoding.EncodeToString([]byte(env))
	return fmt.Sprintf(`set -eu
path=%s
stacks=%s
if [ -e "$path/compose.yaml" ] || [ -e "$path/docker-compose.yaml" ]; then
  echo 'Refusing to overwrite an existing Dockge installation.' >&2
  exit 40
fi
command -v docker >/dev/null 2>&1 || { echo 'Docker Engine is required.' >&2; exit 41; }
docker compose version >/dev/null 2>&1 || { echo 'Docker Compose v2 is required.' >&2; exit 42; }
mkdir -p "$path/data" "$stacks"
printf '%%s' %s | base64 -d > "$path/compose.yaml"
printf '%%s' %s | base64 -d > "$path/.env"
chmod 600 "$path/.env"
cd "$path"
docker compose config >/dev/null
docker compose pull dockge
docker compose up -d --no-deps dockge
docker compose ps dockge
`, quote(path), quote(stacksPath), quote(compose), quote(env64))
}

func UpgradePlan(path, imageTag string) string {
	return fmt.Sprintf("PLAN: validate %s, snapshot compose/.env/data metadata, preserve current image locally, set DOCKGE_IMAGE_TAG=%s, pull, recreate only Dockge, verify, rollback automatically on failure", path, imageTag)
}

func UpgradeScript(path, imageTag string) string {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf(`set -eu
path=%s
new_tag=%s
stamp=%s
cd "$path"
[ -f compose.yaml ] || { echo 'compose.yaml not found' >&2; exit 50; }
docker compose config >/dev/null
old_container="$(docker compose ps -q dockge)"
[ -n "$old_container" ] || { echo 'Dockge container is not running' >&2; exit 51; }
old_image_id="$(docker inspect "$old_container" --format '{{.Image}}')"
rollback_tag="rollback-$stamp"
docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag"
backup="$path/backups/upgrade-$stamp"
mkdir -p "$backup"
cp -a compose.yaml "$backup/compose.yaml"
[ -f .env ] && cp -a .env "$backup/.env" || true
printf '%%s\n' "$old_image_id" > "$backup/old-image-id.txt"
if [ -d data ]; then tar -czf "$backup/data.tar.gz" data; fi
rollback() {
  rc=$?
  trap - EXIT INT TERM
  echo 'Upgrade failed; rolling back Dockge container only.' >&2
  [ -f "$backup/.env" ] && cp -a "$backup/.env" "$path/.env" || true
  DOCKGE_IMAGE_TAG="$rollback_tag" docker compose up -d --no-deps dockge || true
  docker compose ps dockge || true
  exit "$rc"
}
trap rollback EXIT INT TERM
if [ -f .env ] && grep -q '^DOCKGE_IMAGE_TAG=' .env; then
  sed -i "s|^DOCKGE_IMAGE_TAG=.*$|DOCKGE_IMAGE_TAG=$new_tag|" .env
else
  printf 'DOCKGE_IMAGE_TAG=%%s\n' "$new_tag" >> .env
fi
docker compose config >/dev/null
docker compose pull dockge
docker compose up -d --no-deps dockge
sleep 2
docker compose ps dockge
running="$(docker inspect "$(docker compose ps -q dockge)" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'Dockge container is not running after upgrade.' >&2; exit 52; }
trap - EXIT INT TERM
printf 'upgrade=committed backup=%%s rollback_image=ghcr.io/wkarts/dockge:%%s\n' "$backup" "$rollback_tag"
`, quote(path), quote(imageTag), quote(stamp))
}

func RollbackScript(path, backup string) string {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf(`set -eu
path=%s
backup=%s
stamp=%s
cd "$path"
[ -d "$backup" ] || { echo 'backup directory not found' >&2; exit 70; }
[ -f "$backup/compose.yaml" ] || { echo 'backup compose.yaml not found' >&2; exit 71; }
cp -a "$backup/compose.yaml" "$path/compose.yaml"
[ -f "$backup/.env" ] && cp -a "$backup/.env" "$path/.env" || true
rollback_tag="manual-rollback-$stamp"
if [ -f "$backup/old-image-id.txt" ]; then
  old_image_id="$(cat "$backup/old-image-id.txt")"
  docker image inspect "$old_image_id" >/dev/null
  docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag"
  DOCKGE_IMAGE_TAG="$rollback_tag" docker compose up -d --no-deps dockge
else
  docker compose pull dockge
  docker compose up -d --no-deps dockge
fi
docker compose config >/dev/null
docker compose ps dockge
running="$(docker inspect "$(docker compose ps -q dockge)" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'Dockge container is not running after rollback.' >&2; exit 72; }
printf 'rollback=committed backup=%%s\n' "$backup"
`, quote(path), quote(backup), quote(stamp))
}

func MigrationPlanScript(source, target string) string {
	return fmt.Sprintf(`set -eu
source=%s
target=%s
printf 'source=%%s\ntarget=%%s\n' "$source" "$target"
[ "$source" != "$target" ] || { echo 'source and target must differ' >&2; exit 60; }
if [ ! -d "$source" ]; then echo 'source=not_found'; exit 0; fi
printf 'source_compose='; if [ -f "$source/compose.yaml" ] || [ -f "$source/docker-compose.yaml" ]; then echo found; else echo missing; fi
printf 'source_data_bytes='; du -sb "$source/data" 2>/dev/null | awk '{print $1}' || echo 0
printf 'stack_count='; find /opt/stacks -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l
printf '%%s\n' 'No files were changed. Migration execution requires an explicit reviewed plan.'
`, quote(source), quote(target))
}
