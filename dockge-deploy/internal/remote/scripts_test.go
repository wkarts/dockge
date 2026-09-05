package remote

import (
	"strings"
	"testing"
)

func assertOrdered(t *testing.T, script, before, after string) {
	t.Helper()
	beforeIndex := strings.Index(script, before)
	afterIndex := strings.Index(script, after)
	if beforeIndex < 0 {
		t.Fatalf("script missing prerequisite %q", before)
	}
	if afterIndex < 0 {
		t.Fatalf("script missing subsequent step %q", after)
	}
	if beforeIndex >= afterIndex {
		t.Fatalf("expected %q before %q", before, after)
	}
}

func TestInstallScriptRefusesOverwrite(t *testing.T) {
	script := InstallScript("/opt/dockge", "/opt/stacks", "1.6.1", "127.0.0.1", 5001)
	if !strings.Contains(script, "Refusing to overwrite") {
		t.Fatal("install script must refuse an existing installation")
	}
	for _, filename := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if !strings.Contains(script, filename) {
			t.Fatalf("install guard must detect %q", filename)
		}
	}
	if !strings.Contains(script, "docker compose up -d --no-deps dockge") {
		t.Fatal("install must recreate only Dockge")
	}
}

func TestDetectSupportsAllStandardComposeFilenames(t *testing.T) {
	script := DetectScript("/opt/dockge")
	for _, filename := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if !strings.Contains(script, filename) {
			t.Fatalf("detect script missing %q", filename)
		}
	}
}

func TestUpgradeScriptSnapshotsRealDataBindMount(t *testing.T) {
	script := UpgradeScript("/opt/dockge", "1.6.1")
	required := []string{
		"rollback()",
		"docker image tag",
		"trap rollback",
		"docker compose up -d --no-deps dockge",
		"data.tar.gz",
		"data.tar.gz.partial",
		"snapshot-state.txt",
		"data-mount-source.txt",
		"/app/data",
		"data_mount_type",
		"new_data_source",
		"find \"$data_mount_source\" -mindepth 1 -maxdepth 1",
	}
	for _, value := range required {
		if !strings.Contains(script, value) {
			t.Fatalf("upgrade script missing %q", value)
		}
	}
	if !strings.Contains(script, "requires /app/data to be a bind mount") {
		t.Fatal("upgrade must fail closed for unsupported data-volume layouts")
	}
	assertOrdered(t, script, "DOCKGE_IMAGE_TAG=\"$new_tag\" docker compose pull dockge", "docker compose stop dockge\nold_running=")
	assertOrdered(t, script, "docker compose stop dockge\nold_running=", "tar -C \"$data_mount_source\" -czf \"$backup/data.tar.gz.partial\" .")
	assertOrdered(t, script, "data.tar.gz.partial", "mv \"$backup/data.tar.gz.partial\" \"$backup/data.tar.gz\"")
}

func TestRollbackQuiescesBeforeRestoringData(t *testing.T) {
	script := RollbackScript("/opt/dockge", "/opt/dockge/backups/upgrade-test")
	for _, value := range []string{"data-mount-source.txt", "data_restore_path", "find \"$data_restore_path\"", "data.tar.gz", "current_compose"} {
		if !strings.Contains(script, value) {
			t.Fatalf("rollback script missing %q", value)
		}
	}
	assertOrdered(t, script, "docker compose -f \"$current_compose\" stop dockge", "find \"$data_restore_path\" -mindepth 1 -maxdepth 1")
}

func TestDockerInstallSupportsMajorLinuxFamilies(t *testing.T) {
	script := DockerInstallScript()
	for _, value := range []string{"apt-get", "dnf", "yum", "apk", "docker-compose-plugin", "docker compose version"} {
		if !strings.Contains(script, value) {
			t.Fatalf("Docker bootstrap missing %q", value)
		}
	}
}

func TestMigrationPreservesRuntimeMountsStacksAndRollsBack(t *testing.T) {
	script := MigrationScript("/opt/dockge", "/opt/stacks", "1.6.1", "127.0.0.1", 5001)
	required := []string{
		"stacks-before.txt",
		"stacks-after.txt",
		"cmp -s",
		"trap rollback",
		"source-compose",
		"data.tar.gz",
		"data.tar.gz.partial",
		"snapshot-state.txt",
		"data-mount-source.txt",
		"stacks-mount-source.txt",
		"source_stacks_dir",
		"stacks_mount_source",
		"set_env_value DOCKGE_DATA_PATH \"$data_mount_source\"",
		"DOCKGE_ALLOW_DISABLE_AUTH",
		"DOCKGE_TOTP_ISSUER",
		"new_data_source",
		"docker pull \"ghcr.io/wkarts/dockge:$target_tag\"",
		"docker compose -f \"$source_name\" stop dockge",
		"docker compose -f \"$path/compose.yaml\" up -d --no-deps dockge",
		"ghcr.io/wkarts/dockge",
	}
	for _, value := range required {
		if !strings.Contains(script, value) {
			t.Fatalf("migration script missing %q", value)
		}
	}
	for _, filename := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if !strings.Contains(script, filename) {
			t.Fatalf("migration script must support source %q", filename)
		}
	}
	assertOrdered(t, script, "docker pull \"ghcr.io/wkarts/dockge:$target_tag\"", "docker compose -f \"$source_name\" stop dockge\nold_running=")
	assertOrdered(t, script, "docker compose -f \"$source_name\" stop dockge\nold_running=", "tar -C \"$data_mount_source\" -czf \"$backup/data.tar.gz.partial\" .")
	for _, forbidden := range []string{"down -v", "system prune", "rm -rf \"$stacks\""} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("migration script contains forbidden destructive operation %q", forbidden)
		}
	}
}

func TestMigrationPlanIsReadOnlyAndInventoriesRuntimeMounts(t *testing.T) {
	script := MigrationPlanScript("/opt/dockge", "/opt/stacks", "1.6.1", "127.0.0.1", 5001)
	for _, filename := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if !strings.Contains(script, filename) {
			t.Fatalf("migration plan missing %q", filename)
		}
	}
	for _, value := range []string{"read_only=true", "source_data_mount_type", "source_data_mount_source", "source_stacks_mount_source", "stacks_path_match", "snapshot_strategy=quiesce-dockge-before-archive"} {
		if !strings.Contains(script, value) {
			t.Fatalf("migration plan missing %q", value)
		}
	}
	for _, forbidden := range []string{"docker compose stop", "docker compose up", "docker compose pull", "docker pull", "rm -f"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("migration plan contains mutation %q", forbidden)
		}
	}
}

func TestManagerTokenUsesTypedScopesAndNoShellSecretArgument(t *testing.T) {
	script := ManagerTokenScript("/opt/dockge", "dockge-manager", "a,b", false)
	for _, value := range []string{"server:read", "stacks:read", "stacks:write", "stacks:operate", "stacks:adopt", "--json"} {
		if !strings.Contains(script, value) {
			t.Fatalf("manager token script missing %q", value)
		}
	}
}
