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
printf '%s\n' '== init =='
(command -v systemctl >/dev/null 2>&1 && systemctl is-active docker 2>/dev/null) || true
`
}

func DockerInstallScript() string {
	return `set -eu
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  echo 'docker=already_ready'
  docker --version
  docker compose version
  exit 0
fi

[ -r /etc/os-release ] || { echo 'Unsupported Linux: /etc/os-release not found.' >&2; exit 30; }
. /etc/os-release
id="${ID:-}"
like="${ID_LIKE:-}"

install_debian() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y ca-certificates curl
  install -m 0755 -d /etc/apt/keyrings
  case "$id" in
    ubuntu|debian) docker_os="$id" ;;
    *)
      if printf '%s' "$like" | grep -qw ubuntu; then docker_os=ubuntu
      elif printf '%s' "$like" | grep -qw debian; then docker_os=debian
      else echo "Unsupported apt-family distribution: $id ($like)" >&2; exit 31; fi
      ;;
  esac
  curl -fsSL "https://download.docker.com/linux/$docker_os/gpg" -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  arch="$(dpkg --print-architecture)"
  codename="${VERSION_CODENAME:-}"
  [ -n "$codename" ] || codename="$(. /etc/os-release; printf '%s' "${UBUNTU_CODENAME:-}")"
  [ -n "$codename" ] || { echo 'Unable to resolve Debian/Ubuntu codename.' >&2; exit 32; }
  printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/%s %s stable\n' "$arch" "$docker_os" "$codename" > /etc/apt/sources.list.d/docker.list
  apt-get update
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
}

install_rpm() {
  if command -v dnf >/dev/null 2>&1; then pm=dnf; else pm=yum; fi
  "$pm" -y install ca-certificates curl || true
  case "$id" in
    fedora) repo_os=fedora ;;
    *) repo_os=centos ;;
  esac
  repo="https://download.docker.com/linux/$repo_os/docker-ce.repo"
  if command -v dnf >/dev/null 2>&1; then
    dnf -y install dnf-plugins-core || true
    dnf config-manager --add-repo "$repo" >/dev/null 2>&1 || \
      dnf config-manager addrepo --from-repofile="$repo" >/dev/null 2>&1 || \
      { echo 'Unable to configure Docker CE repository with dnf.' >&2; exit 33; }
  else
    yum -y install yum-utils
    yum-config-manager --add-repo "$repo"
  fi
  "$pm" -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
}

install_alpine() {
  apk add --no-cache docker docker-cli-compose
}

if command -v apt-get >/dev/null 2>&1; then
  install_debian
elif command -v dnf >/dev/null 2>&1 || command -v yum >/dev/null 2>&1; then
  install_rpm
elif command -v apk >/dev/null 2>&1; then
  install_alpine
else
  echo "Unsupported package manager for distribution: $id ($like)" >&2
  exit 34
fi

if command -v systemctl >/dev/null 2>&1; then
  systemctl enable --now docker
elif command -v rc-update >/dev/null 2>&1; then
  rc-update add docker default >/dev/null 2>&1 || true
  rc-service docker start
elif command -v service >/dev/null 2>&1; then
  service docker start || true
fi

docker --version
docker compose version
docker info >/dev/null
echo 'docker=ready'
`
}

func DetectScript(path string) string {
	p := quote(path)
	return fmt.Sprintf(`set -eu
path=%s
printf 'path=%%s\n' "$path"
compose=''
if [ -f "$path/compose.yaml" ]; then compose="$path/compose.yaml"
elif [ -f "$path/docker-compose.yaml" ]; then compose="$path/docker-compose.yaml"
fi
if [ -n "$compose" ]; then
  printf '%%s\n' 'dockge_installation=found'
  cd "$path"
  docker compose config >/dev/null 2>&1 && printf '%%s\n' 'compose_config=valid' || printf '%%s\n' 'compose_config=invalid'
  printf 'images='; docker compose config --images 2>/dev/null | paste -sd ',' - || true
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
	env := canonicalEnv(stacksPath, imageTag, bindHost, port)
	env64 := base64.StdEncoding.EncodeToString([]byte(env))
	return fmt.Sprintf(`set -eu
path=%s
stacks=%s
if [ -e "$path/compose.yaml" ] || [ -e "$path/docker-compose.yaml" ]; then
  echo 'Refusing to overwrite an existing Dockge installation.' >&2
  exit 40
fi
command -v docker >/dev/null 2>&1 || { echo 'Docker Engine is required. Run: dockge-deploy docker install --apply' >&2; exit 41; }
docker compose version >/dev/null 2>&1 || { echo 'Docker Compose v2 is required. Run: dockge-deploy docker install --apply' >&2; exit 42; }
mkdir -p "$path/data" "$stacks"
printf '%%s' %s | base64 -d > "$path/compose.yaml"
printf '%%s' %s | base64 -d > "$path/.env"
chmod 600 "$path/.env"
cd "$path"
docker compose config >/dev/null
docker compose pull dockge
docker compose up -d --no-deps dockge
docker compose ps dockge
running="$(docker inspect "$(docker compose ps -q dockge)" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'Dockge container is not running after installation.' >&2; exit 43; }
`, quote(path), quote(stacksPath), quote(compose), quote(env64))
}

func UpgradePlan(path, imageTag string) string {
	return fmt.Sprintf("PLAN: validate %s, snapshot compose/.env/data, preserve the current image locally, set DOCKGE_IMAGE_TAG=%s, pull, recreate only Dockge, verify, and rollback automatically on failure; application stacks/volumes remain untouched", path, imageTag)
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
[ -f .env ] && { cp -a .env "$backup/.env"; touch "$backup/had-env"; } || true
printf '%%s\n' "$old_image_id" > "$backup/old-image-id.txt"
if [ -d data ]; then tar -czf "$backup/data.tar.gz" data; fi
rollback() {
  rc=$?
  trap - EXIT INT TERM
  echo 'Upgrade failed; rolling back Dockge container only.' >&2
  cp -a "$backup/compose.yaml" "$path/compose.yaml" || true
  if [ -f "$backup/had-env" ]; then cp -a "$backup/.env" "$path/.env" || true; else rm -f "$path/.env"; fi
  if [ -f "$backup/data.tar.gz" ]; then rm -rf "$path/data"; tar -xzf "$backup/data.tar.gz" -C "$path" || true; fi
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
if [ -f "$backup/source-compose-name.txt" ]; then
  source_name="$(cat "$backup/source-compose-name.txt")"
  [ -f "$backup/source-compose" ] || { echo 'backup source compose not found' >&2; exit 71; }
  docker compose stop dockge >/dev/null 2>&1 || true
  rm -f "$path/compose.yaml" "$path/docker-compose.yaml"
  cp -a "$backup/source-compose" "$path/$source_name"
else
  [ -f "$backup/compose.yaml" ] || { echo 'backup compose.yaml not found' >&2; exit 71; }
  cp -a "$backup/compose.yaml" "$path/compose.yaml"
fi
if [ -f "$backup/had-env" ]; then cp -a "$backup/.env" "$path/.env"; else rm -f "$path/.env"; fi
if [ -f "$backup/data.tar.gz" ]; then rm -rf "$path/data"; tar -xzf "$backup/data.tar.gz" -C "$path"; fi
rollback_tag="manual-rollback-$stamp"
if [ -f "$backup/old-image-id.txt" ]; then
  old_image_id="$(cat "$backup/old-image-id.txt")"
  docker image inspect "$old_image_id" >/dev/null
  docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag" || true
fi
docker compose config >/dev/null
docker compose up -d --no-deps dockge
docker compose ps dockge
running="$(docker inspect "$(docker compose ps -q dockge)" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'Dockge container is not running after rollback.' >&2; exit 72; }
printf 'rollback=committed backup=%%s\n' "$backup"
`, quote(path), quote(backup), quote(stamp))
}

func MigrationPlan(path, stacksPath, imageTag, bindHost string, port int) string {
	return fmt.Sprintf("PLAN: inspect existing Dockge at %s, preserve stacks at %s, snapshot compose/.env/data and current image, stop/recreate only Dockge using ghcr.io/wkarts/dockge:%s bound to %s:%d, verify, and restore the original installation automatically on failure. Re-run with --apply only after reviewing dockge plan-migration output.", path, stacksPath, imageTag, bindHost, port)
}

func MigrationPlanScript(path, stacksPath, imageTag, bindHost string, port int) string {
	return fmt.Sprintf(`set -eu
path=%s
stacks=%s
target_tag=%s
printf 'path=%%s\nstacks_path=%%s\ntarget_image=ghcr.io/wkarts/dockge:%%s\nbind=%%s:%%s\n' "$path" "$stacks" "$target_tag" %s %d
if [ ! -d "$path" ]; then echo 'source=not_found'; exit 0; fi
cd "$path"
compose=''
if [ -f compose.yaml ]; then compose=compose.yaml
elif [ -f docker-compose.yaml ]; then compose=docker-compose.yaml
else echo 'source_compose=missing'; exit 0
fi
printf 'source_compose=%%s\n' "$compose"
docker compose config >/dev/null 2>&1 && echo 'compose_config=valid' || echo 'compose_config=invalid'
printf 'source_images='; docker compose config --images 2>/dev/null | paste -sd ',' - || true
printf 'source_data_bytes='; du -sb "$path/data" 2>/dev/null | awk '{print $1}' || echo 0
printf 'stack_count='; find "$stacks" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l
printf 'dockge_container='; docker compose ps -q dockge 2>/dev/null || true
printf '%%s\n' 'read_only=true'
printf '%%s\n' 'No files, stacks, volumes or containers were changed.'
`, quote(path), quote(stacksPath), quote(imageTag), quote(bindHost), port)
}

func MigrationScript(path, stacksPath, imageTag, bindHost string, port int) string {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	compose64 := base64.StdEncoding.EncodeToString([]byte(ComposeYAML))
	env64 := base64.StdEncoding.EncodeToString([]byte(canonicalEnv(stacksPath, imageTag, bindHost, port)))
	return fmt.Sprintf(`set -eu
path=%s
stacks=%s
stamp=%s
cd "$path"
source_name=''
if [ -f compose.yaml ]; then source_name=compose.yaml
elif [ -f docker-compose.yaml ]; then source_name=docker-compose.yaml
else echo 'No existing Dockge compose file found.' >&2; exit 80
fi
docker compose config >/dev/null
old_container="$(docker compose ps -q dockge)"
[ -n "$old_container" ] || { echo 'Existing Dockge container is not running.' >&2; exit 81; }
old_image_id="$(docker inspect "$old_container" --format '{{.Image}}')"
source_images="$(docker compose config --images 2>/dev/null | paste -sd ',' - || true)"
if printf '%%s' "$source_images" | grep -q 'ghcr.io/wkarts/dockge'; then
  echo 'This installation already uses ghcr.io/wkarts/dockge; use dockge upgrade instead.' >&2
  exit 82
fi
[ -d "$stacks" ] || { echo "Stacks path does not exist: $stacks" >&2; exit 83; }
backup="$path/backups/migration-$stamp"
mkdir -p "$backup"
printf '%%s\n' "$source_name" > "$backup/source-compose-name.txt"
cp -a "$path/$source_name" "$backup/source-compose"
[ -f .env ] && { cp -a .env "$backup/.env"; touch "$backup/had-env"; } || true
printf '%%s\n' "$old_image_id" > "$backup/old-image-id.txt"
printf '%%s\n' "$source_images" > "$backup/source-images.txt"
find "$stacks" -mindepth 1 -maxdepth 1 -type d -printf '%%f\n' 2>/dev/null | sort > "$backup/stacks-before.txt" || true
if [ -d data ]; then tar -czf "$backup/data.tar.gz" data; fi
rollback_tag="migration-rollback-$stamp"
docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag" || true
rollback() {
  rc=$?
  trap - EXIT INT TERM
  echo 'Migration failed; restoring original Dockge installation. Application stacks remain untouched.' >&2
  docker compose stop dockge >/dev/null 2>&1 || true
  rm -f "$path/compose.yaml" "$path/docker-compose.yaml"
  cp -a "$backup/source-compose" "$path/$(cat "$backup/source-compose-name.txt")" || true
  if [ -f "$backup/had-env" ]; then cp -a "$backup/.env" "$path/.env" || true; else rm -f "$path/.env"; fi
  if [ -f "$backup/data.tar.gz" ]; then rm -rf "$path/data"; tar -xzf "$backup/data.tar.gz" -C "$path" || true; fi
  cd "$path"
  docker compose up -d --no-deps dockge || true
  docker compose ps dockge || true
  exit "$rc"
}
trap rollback EXIT INT TERM

docker compose stop dockge
printf '%%s' %s | base64 -d > "$path/compose.yaml"
printf '%%s' %s | base64 -d > "$path/.env"
chmod 600 "$path/.env"
[ "$source_name" = compose.yaml ] || rm -f "$path/$source_name"
mkdir -p "$path/data" "$stacks"
docker compose config >/dev/null
docker compose pull dockge
docker compose up -d --no-deps dockge
sleep 2
new_container="$(docker compose ps -q dockge)"
[ -n "$new_container" ] || { echo 'New Dockge container was not created.' >&2; exit 84; }
running="$(docker inspect "$new_container" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'New Dockge container is not running.' >&2; exit 85; }
find "$stacks" -mindepth 1 -maxdepth 1 -type d -printf '%%f\n' 2>/dev/null | sort > "$backup/stacks-after.txt" || true
cmp -s "$backup/stacks-before.txt" "$backup/stacks-after.txt" || { echo 'Stacks directory set changed during migration; rolling back.' >&2; exit 86; }
trap - EXIT INT TERM
printf 'migration=committed backup=%%s source_images=%%s target=ghcr.io/wkarts/dockge:%s\n' "$backup" "$source_images"
`, quote(path), quote(stacksPath), quote(stamp), quote(compose64), quote(env64), imageTag)
}

func ManagerTokenScript(path, name, prefixes string, replace bool) string {
	replaceArg := ""
	if replace {
		replaceArg = " --replace --replace-grace-seconds 0"
	}
	scopes := "server:read,stacks:read,stacks:write,stacks:delete,stacks:operate,stacks:adopt"
	return fmt.Sprintf(`set -eu
path=%s
cd "$path"
docker compose config >/dev/null
docker compose ps -q dockge >/dev/null
exec docker compose exec -T dockge npm run --silent api-token:create -- --name %s --scopes %s --prefixes %s --json%s
`, quote(path), quote(name), quote(scopes), quote(prefixes), replaceArg)
}

func canonicalEnv(stacksPath, imageTag, bindHost string, port int) string {
	return fmt.Sprintf("DOCKGE_IMAGE=ghcr.io/wkarts/dockge\nDOCKGE_IMAGE_TAG=%s\nDOCKGE_DATA_PATH=./data\nDOCKGE_STACKS_PATH=%s\nDOCKGE_BIND_HOST=%s\nDOCKGE_PORT=%d\nDOCKGE_API_TOKENS_FILE=/app/data/api-tokens.json\nDOCKGE_API_AUDIT_FILE=/app/data/api-audit.jsonl\nDOCKGE_API_IDEMPOTENCY_FILE=/app/data/api-idempotency.json\n", imageTag, stacksPath, bindHost, port)
}
