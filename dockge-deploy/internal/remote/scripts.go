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
for candidate in compose.yaml compose.yml docker-compose.yaml docker-compose.yml; do
  if [ -f "$path/$candidate" ]; then compose="$path/$candidate"; break; fi
done
if [ -n "$compose" ]; then
  printf '%%s\n' 'dockge_installation=found'
  printf 'compose_file=%%s\n' "$(basename "$compose")"
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
for candidate in compose.yaml compose.yml docker-compose.yaml docker-compose.yml; do
  if [ -e "$path/$candidate" ]; then
    echo "Refusing to overwrite an existing Dockge installation ($candidate)." >&2
    exit 40
  fi
done
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
	return fmt.Sprintf("PLAN: validate %s, pre-pull Dockge %s while the current instance remains online, quiesce only the Dockge container, snapshot the real /app/data bind mount consistently, recreate only Dockge, verify, and rollback automatically on failure; application stacks/volumes remain untouched", path, imageTag)
}

func UpgradeScript(path, imageTag string) string {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	return fmt.Sprintf(`set -eu
path=%s
new_tag=%s
stamp=%s
cd "$path"
[ -f compose.yaml ] || { echo 'compose.yaml not found; use dockge migrate for a legacy/non-canonical Compose filename.' >&2; exit 50; }
docker compose config >/dev/null
old_container="$(docker compose ps -q dockge)"
[ -n "$old_container" ] || { echo 'Dockge container is not running' >&2; exit 51; }
old_image_id="$(docker inspect "$old_container" --format '{{.Image}}')"
data_mount_type="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $1; exit}')"
data_mount_source="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $2; exit}')"
[ "$data_mount_type" = bind ] || { echo 'Automatic upgrade requires /app/data to be a bind mount; create a reviewed backup/migration plan for named volumes.' >&2; exit 53; }
case "$data_mount_source" in ''|'/') echo 'Unsafe /app/data mount source; refusing automatic upgrade.' >&2; exit 54;; esac
[ -d "$data_mount_source" ] || { echo "Dockge data bind source does not exist: $data_mount_source" >&2; exit 55; }
rollback_tag="rollback-$stamp"
docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag"
backup="$path/backups/upgrade-$stamp"
case "$backup/" in "$data_mount_source/"*) echo 'Backup directory cannot be inside the Dockge data mount.' >&2; exit 56;; esac
mkdir -p "$backup"
cp -a compose.yaml "$backup/compose.yaml"
[ -f .env ] && { cp -a .env "$backup/.env"; touch "$backup/had-env"; } || true
printf '%%s\n' "$old_image_id" > "$backup/old-image-id.txt"
printf '%%s\n' "$data_mount_type" > "$backup/data-mount-type.txt"
printf '%%s\n' "$data_mount_source" > "$backup/data-mount-source.txt"

# Fetch the target image before the maintenance window. Shell environment has
# precedence over .env interpolation, so the running configuration is not
# changed yet and application stacks remain untouched.
DOCKGE_IMAGE_TAG="$new_tag" docker compose pull dockge

rollback() {
  rc=$?
  trap - EXIT INT TERM
  echo 'Upgrade failed; rolling back Dockge container only.' >&2
  docker compose stop dockge >/dev/null 2>&1 || true
  cp -a "$backup/compose.yaml" "$path/compose.yaml" || true
  if [ -f "$backup/had-env" ]; then cp -a "$backup/.env" "$path/.env" || true; else rm -f "$path/.env"; fi
  if [ -f "$backup/data.tar.gz" ]; then
    find "$data_mount_source" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} + || true
    tar -xzf "$backup/data.tar.gz" -C "$data_mount_source" || true
  fi
  DOCKGE_IMAGE_TAG="$rollback_tag" docker compose up -d --no-deps dockge || true
  docker compose ps dockge || true
  exit "$rc"
}
trap rollback EXIT INT TERM

# Quiesce Dockge before copying persistent data. This matters for SQLite and
# other files that must not change while the archive is being produced.
docker compose stop dockge
old_running="$(docker inspect "$old_container" --format '{{.State.Running}}')"
[ "$old_running" = false ] || { echo 'Dockge did not quiesce before the data snapshot.' >&2; exit 57; }
printf 'quiesced=true\n' > "$backup/snapshot-state.txt"
tar -C "$data_mount_source" -czf "$backup/data.tar.gz.partial" .
mv "$backup/data.tar.gz.partial" "$backup/data.tar.gz"
printf 'snapshot=consistent\n' >> "$backup/snapshot-state.txt"

if [ -f .env ] && grep -q '^DOCKGE_IMAGE_TAG=' .env; then
  sed -i "s|^DOCKGE_IMAGE_TAG=.*$|DOCKGE_IMAGE_TAG=$new_tag|" .env
else
  printf 'DOCKGE_IMAGE_TAG=%%s\n' "$new_tag" >> .env
fi
docker compose config >/dev/null
docker compose up -d --no-deps dockge
sleep 2
docker compose ps dockge
new_container="$(docker compose ps -q dockge)"
[ -n "$new_container" ] || { echo 'Dockge container was not created after upgrade.' >&2; exit 58; }
running="$(docker inspect "$new_container" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'Dockge container is not running after upgrade.' >&2; exit 52; }
new_data_source="$(docker inspect "$new_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $2; exit}')"
[ "$new_data_source" = "$data_mount_source" ] || { echo "Dockge data mount changed from $data_mount_source to $new_data_source; rolling back." >&2; exit 59; }
trap - EXIT INT TERM
printf 'upgrade=committed backup=%%s data_mount=%%s rollback_image=ghcr.io/wkarts/dockge:%%s\n' "$backup" "$data_mount_source" "$rollback_tag"
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

# Best effort quiesce of the currently installed Dockge must happen before a
# persistent-data restore. Application stacks are separate Compose projects
# and are intentionally not stopped.
current_compose=''
for candidate in compose.yaml compose.yml docker-compose.yaml docker-compose.yml; do
  if [ -f "$path/$candidate" ]; then current_compose="$path/$candidate"; break; fi
done
if [ -n "$current_compose" ]; then
  docker compose -f "$current_compose" stop dockge >/dev/null 2>&1 || true
fi

restored_compose=''
if [ -f "$backup/source-compose-name.txt" ]; then
  source_name="$(cat "$backup/source-compose-name.txt")"
  [ -f "$backup/source-compose" ] || { echo 'backup source compose not found' >&2; exit 71; }
  rm -f "$path/compose.yaml" "$path/compose.yml" "$path/docker-compose.yaml" "$path/docker-compose.yml"
  cp -a "$backup/source-compose" "$path/$source_name"
  restored_compose="$path/$source_name"
else
  [ -f "$backup/compose.yaml" ] || { echo 'backup compose.yaml not found' >&2; exit 71; }
  cp -a "$backup/compose.yaml" "$path/compose.yaml"
  restored_compose="$path/compose.yaml"
fi
if [ -f "$backup/had-env" ]; then cp -a "$backup/.env" "$path/.env"; else rm -f "$path/.env"; fi
data_restore_path="$path/data"
[ -f "$backup/data-mount-source.txt" ] && data_restore_path="$(cat "$backup/data-mount-source.txt")"
case "$data_restore_path" in ''|'/') echo 'Unsafe data restore path in backup.' >&2; exit 73;; esac
if [ -f "$backup/data.tar.gz" ]; then
  mkdir -p "$data_restore_path"
  find "$data_restore_path" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
  tar -xzf "$backup/data.tar.gz" -C "$data_restore_path"
fi
rollback_tag="manual-rollback-$stamp"
if [ -f "$backup/old-image-id.txt" ]; then
  old_image_id="$(cat "$backup/old-image-id.txt")"
  docker image inspect "$old_image_id" >/dev/null
  docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag" || true
fi
docker compose -f "$restored_compose" config >/dev/null
docker compose -f "$restored_compose" up -d --no-deps dockge
docker compose -f "$restored_compose" ps dockge
running="$(docker inspect "$(docker compose -f "$restored_compose" ps -q dockge)" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'Dockge container is not running after rollback.' >&2; exit 72; }
printf 'rollback=committed backup=%%s data_restore_path=%%s\n' "$backup" "$data_restore_path"
`, quote(path), quote(backup), quote(stamp))
}

func MigrationPlan(path, stacksPath, imageTag, bindHost string, port int) string {
	return fmt.Sprintf("PLAN: inspect existing Dockge at %s, verify the real /app/data and stacks bind mounts, pre-pull ghcr.io/wkarts/dockge:%s while the source remains online, then quiesce only Dockge and snapshot its persistent data consistently; preserve stacks at %s, recreate only Dockge bound to %s:%d, verify, and restore the original installation automatically on failure. Re-run with --apply only after reviewing dockge plan-migration output.", path, imageTag, stacksPath, bindHost, port)
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
for candidate in compose.yaml compose.yml docker-compose.yaml docker-compose.yml; do
  if [ -f "$candidate" ]; then compose="$candidate"; break; fi
done
if [ -z "$compose" ]; then echo 'source_compose=missing'; exit 0; fi
printf 'source_compose=%%s\n' "$compose"
docker compose -f "$compose" config >/dev/null 2>&1 && echo 'compose_config=valid' || echo 'compose_config=invalid'
printf 'source_images='; docker compose -f "$compose" config --images 2>/dev/null | paste -sd ',' - || true
old_container="$(docker compose -f "$compose" ps -q dockge 2>/dev/null || true)"
printf 'dockge_container=%%s\n' "$old_container"
if [ -n "$old_container" ]; then
  data_mount_type="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $1; exit}')"
  data_mount_source="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $2; exit}')"
  source_stacks_dir="$(docker inspect "$old_container" --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^DOCKGE_STACKS_DIR=//p' | head -n1)"
  stacks_mount_type="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' -v dest="$source_stacks_dir" '$3==dest {print $1; exit}')"
  stacks_mount_source="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' -v dest="$source_stacks_dir" '$3==dest {print $2; exit}')"
  printf 'source_data_mount_type=%%s\nsource_data_mount_source=%%s\n' "$data_mount_type" "$data_mount_source"
  printf 'source_stacks_dir=%%s\nsource_stacks_mount_type=%%s\nsource_stacks_mount_source=%%s\n' "$source_stacks_dir" "$stacks_mount_type" "$stacks_mount_source"
  if [ "$source_stacks_dir" = "$stacks" ] && [ "$stacks_mount_source" = "$stacks" ]; then echo 'stacks_path_match=true'; else echo 'stacks_path_match=false'; fi
  printf 'source_data_bytes='; du -sb "$data_mount_source" 2>/dev/null | awk '{print $1}' || echo 0
  echo 'snapshot_strategy=quiesce-dockge-before-archive'
else
  echo 'source_data_mount_type=unknown'
  echo 'source_data_mount_source=unknown'
  echo 'source_stacks_dir=unknown'
  echo 'stacks_path_match=unknown'
  echo 'snapshot_strategy=unavailable-until-dockge-is-running'
fi
printf 'stack_count='; find "$stacks" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l
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
target_tag=%s
stamp=%s
cd "$path"
source_name=''
for candidate in compose.yaml compose.yml docker-compose.yaml docker-compose.yml; do
  if [ -f "$candidate" ]; then source_name="$candidate"; break; fi
done
[ -n "$source_name" ] || { echo 'No existing Dockge compose file found.' >&2; exit 80; }
docker compose -f "$source_name" config >/dev/null
old_container="$(docker compose -f "$source_name" ps -q dockge)"
[ -n "$old_container" ] || { echo 'Existing Dockge container is not running.' >&2; exit 81; }
old_image_id="$(docker inspect "$old_container" --format '{{.Image}}')"
source_images="$(docker compose -f "$source_name" config --images 2>/dev/null | paste -sd ',' - || true)"
if printf '%%s' "$source_images" | grep -q 'ghcr.io/wkarts/dockge'; then
  echo 'This installation already uses ghcr.io/wkarts/dockge; use dockge upgrade instead.' >&2
  exit 82
fi
data_mount_type="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $1; exit}')"
data_mount_source="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $2; exit}')"
[ "$data_mount_type" = bind ] || { echo 'Automatic migration requires /app/data to be a bind mount; named-volume layouts require a reviewed manual migration.' >&2; exit 87; }
case "$data_mount_source" in ''|'/') echo 'Unsafe /app/data mount source; refusing automatic migration.' >&2; exit 88;; esac
[ -d "$data_mount_source" ] || { echo "Dockge data bind source does not exist: $data_mount_source" >&2; exit 89; }
source_stacks_dir="$(docker inspect "$old_container" --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^DOCKGE_STACKS_DIR=//p' | head -n1)"
[ -n "$source_stacks_dir" ] || { echo 'Running Dockge does not expose DOCKGE_STACKS_DIR; refusing automatic migration.' >&2; exit 90; }
[ "$source_stacks_dir" = "$stacks" ] || { echo "Requested stacks path $stacks differs from running Dockge path $source_stacks_dir; review plan-migration and retry with the real path." >&2; exit 91; }
stacks_mount_type="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' -v dest="$source_stacks_dir" '$3==dest {print $1; exit}')"
stacks_mount_source="$(docker inspect "$old_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' -v dest="$source_stacks_dir" '$3==dest {print $2; exit}')"
[ "$stacks_mount_type" = bind ] || { echo 'Automatic migration requires the stacks directory to be a bind mount.' >&2; exit 92; }
[ "$stacks_mount_source" = "$stacks" ] || { echo "Stacks bind source $stacks_mount_source differs from container path $stacks; current canonical Compose requires identical host/container paths." >&2; exit 93; }
[ -d "$stacks" ] || { echo "Stacks path does not exist: $stacks" >&2; exit 83; }
backup="$path/backups/migration-$stamp"
case "$backup/" in "$data_mount_source/"*) echo 'Backup directory cannot be inside the Dockge data mount.' >&2; exit 94;; esac
mkdir -p "$backup"
printf '%%s\n' "$source_name" > "$backup/source-compose-name.txt"
cp -a "$path/$source_name" "$backup/source-compose"
[ -f .env ] && { cp -a .env "$backup/.env"; touch "$backup/had-env"; } || true
printf '%%s\n' "$old_image_id" > "$backup/old-image-id.txt"
printf '%%s\n' "$source_images" > "$backup/source-images.txt"
printf '%%s\n' "$data_mount_type" > "$backup/data-mount-type.txt"
printf '%%s\n' "$data_mount_source" > "$backup/data-mount-source.txt"
printf '%%s\n' "$source_stacks_dir" > "$backup/stacks-dir.txt"
printf '%%s\n' "$stacks_mount_source" > "$backup/stacks-mount-source.txt"
find "$stacks" -mindepth 1 -maxdepth 1 -type d -printf '%%f\n' 2>/dev/null | sort > "$backup/stacks-before.txt" || true
rollback_tag="migration-rollback-$stamp"
docker image tag "$old_image_id" "ghcr.io/wkarts/dockge:$rollback_tag" || true

# Pull target before quiescing the old UI/API to keep the maintenance window as
# short as possible.
docker pull "ghcr.io/wkarts/dockge:$target_tag"

rollback() {
  rc=$?
  trap - EXIT INT TERM
  echo 'Migration failed; restoring original Dockge installation. Application stacks remain untouched.' >&2
  if [ -f "$path/compose.yaml" ]; then
    docker compose -f "$path/compose.yaml" stop dockge >/dev/null 2>&1 || true
  else
    docker compose -f "$source_name" stop dockge >/dev/null 2>&1 || true
  fi
  restore_name="$(cat "$backup/source-compose-name.txt")"
  rm -f "$path/compose.yaml" "$path/compose.yml" "$path/docker-compose.yaml" "$path/docker-compose.yml"
  cp -a "$backup/source-compose" "$path/$restore_name" || true
  if [ -f "$backup/had-env" ]; then cp -a "$backup/.env" "$path/.env" || true; else rm -f "$path/.env"; fi
  if [ -f "$backup/data.tar.gz" ]; then
    find "$data_mount_source" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} + || true
    tar -xzf "$backup/data.tar.gz" -C "$data_mount_source" || true
  fi
  cd "$path"
  docker compose -f "$restore_name" up -d --no-deps dockge || true
  docker compose -f "$restore_name" ps dockge || true
  exit "$rc"
}
trap rollback EXIT INT TERM

container_env_value() {
  docker inspect "$old_container" --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n "s/^$1=//p" | head -n1
}
set_env_value() {
  key="$1"; value="$2"; tmp="$path/.env.tmp"
  awk -v key="$key" -v value="$value" '
    BEGIN { prefix = key "="; done = 0 }
    index($0, prefix) == 1 { if (!done) { print prefix value; done = 1 }; next }
    { print }
    END { if (!done) print prefix value }
  ' "$path/.env" > "$tmp"
  mv "$tmp" "$path/.env"
}

# Stop only Dockge, verify the old process is quiesced, then archive the real
# persistent bind atomically. Application containers keep running throughout.
docker compose -f "$source_name" stop dockge
old_running="$(docker inspect "$old_container" --format '{{.State.Running}}')"
[ "$old_running" = false ] || { echo 'Source Dockge did not quiesce before the data snapshot.' >&2; exit 96; }
printf 'quiesced=true\n' > "$backup/snapshot-state.txt"
tar -C "$data_mount_source" -czf "$backup/data.tar.gz.partial" .
mv "$backup/data.tar.gz.partial" "$backup/data.tar.gz"
printf 'snapshot=consistent\n' >> "$backup/snapshot-state.txt"

printf '%%s' %s | base64 -d > "$path/compose.yaml"
printf '%%s' %s | base64 -d > "$path/.env"
set_env_value DOCKGE_DATA_PATH "$data_mount_source"
for key in DOCKGE_API_TOKENS_FILE DOCKGE_API_AUDIT_FILE DOCKGE_API_IDEMPOTENCY_FILE DOCKGE_ENABLE_CONSOLE DOCKGE_ALLOW_DISABLE_AUTH DOCKGE_TOTP_ISSUER; do
  value="$(container_env_value "$key")"
  [ -n "$value" ] && set_env_value "$key" "$value"
done
chmod 600 "$path/.env"
for candidate in compose.yml docker-compose.yaml docker-compose.yml; do rm -f "$path/$candidate"; done
mkdir -p "$data_mount_source" "$stacks"
docker compose -f "$path/compose.yaml" config >/dev/null
# The image was pre-pulled before quiescing, but Compose pull is cheap when
# already present and validates the exact canonical reference.
docker compose -f "$path/compose.yaml" pull dockge
docker compose -f "$path/compose.yaml" up -d --no-deps dockge
sleep 2
new_container="$(docker compose -f "$path/compose.yaml" ps -q dockge)"
[ -n "$new_container" ] || { echo 'New Dockge container was not created.' >&2; exit 84; }
running="$(docker inspect "$new_container" --format '{{.State.Running}}')"
[ "$running" = true ] || { echo 'New Dockge container is not running.' >&2; exit 85; }
new_data_source="$(docker inspect "$new_container" --format '{{range .Mounts}}{{.Type}}{{printf "\t"}}{{.Source}}{{printf "\t"}}{{.Destination}}{{println}}{{end}}' | awk -F '\t' '$3=="/app/data" {print $2; exit}')"
[ "$new_data_source" = "$data_mount_source" ] || { echo "Dockge data mount changed from $data_mount_source to $new_data_source; rolling back." >&2; exit 95; }
find "$stacks" -mindepth 1 -maxdepth 1 -type d -printf '%%f\n' 2>/dev/null | sort > "$backup/stacks-after.txt" || true
cmp -s "$backup/stacks-before.txt" "$backup/stacks-after.txt" || { echo 'Stacks directory set changed during migration; rolling back.' >&2; exit 86; }
trap - EXIT INT TERM
printf 'migration=committed backup=%%s source_compose=%%s source_images=%%s data_mount=%%s target=ghcr.io/wkarts/dockge:%%s\n' "$backup" "$source_name" "$source_images" "$data_mount_source" "$target_tag"
`, quote(path), quote(stacksPath), quote(imageTag), quote(stamp), quote(compose64), quote(env64))
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
