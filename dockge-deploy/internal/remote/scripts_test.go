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
	required := []string{"rollback()", "docker image tag", "trap rollback", "docker compose up -d --no-deps dockge"}
	for _, value := range required {
		if !strings.Contains(script, value) {
			t.Fatalf("upgrade script missing %q", value)
		}
	}
}
