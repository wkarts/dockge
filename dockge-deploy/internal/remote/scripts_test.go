package remote

import (
	"strings"
	"testing"
)

func TestInstallScriptRefusesOverwrite(t *testing.T) {
	script := InstallScript("/opt/dockge", "/opt/stacks", "1.6.1", "127.0.0.1", 5001)
	if !strings.Contains(script, "Refusing to overwrite") {
		t.Fatal("install script must refuse an existing installation")
	}
	if !strings.Contains(script, "docker compose up -d --no-deps dockge") {
		t.Fatal("install must recreate only Dockge")
	}
}

func TestUpgradeScriptContainsRollback(t *testing.T) {
	script := UpgradeScript("/opt/dockge", "1.6.1")
	required := []string{"rollback()", "docker image tag", "trap rollback", "docker compose up -d --no-deps dockge", "data.tar.gz"}
	for _, value := range required {
		if !strings.Contains(script, value) {
			t.Fatalf("upgrade script missing %q", value)
		}
	}
}

func TestDockerInstallSupportsMajorLinuxFamilies(t *testing.T) {
	script := DockerInstallScript()
	for _, value := range []string{"apt-get", "dnf", "yum", "apk", "docker-compose-plugin", "docker compose version"} {
		if !strings.Contains(script, value) {
			t.Fatalf("Docker bootstrap missing %q", value)
		}
	}
}

func TestMigrationPreservesStacksAndRollsBack(t *testing.T) {
	script := MigrationScript("/opt/dockge", "/opt/stacks", "1.6.1", "127.0.0.1", 5001)
	required := []string{
		"stacks-before.txt",
		"stacks-after.txt",
		"cmp -s",
		"trap rollback",
		"source-compose",
		"data.tar.gz",
		"docker compose stop dockge",
		"docker compose up -d --no-deps dockge",
		"ghcr.io/wkarts/dockge",
	}
	for _, value := range required {
		if !strings.Contains(script, value) {
			t.Fatalf("migration script missing %q", value)
		}
	}
	for _, forbidden := range []string{"down -v", "system prune", "rm -rf \"$stacks\""} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("migration script contains forbidden destructive operation %q", forbidden)
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
