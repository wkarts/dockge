package agent

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/wkarts/infrastructure-agent/internal/config"
)

func writeCredential(t *testing.T, root, name, value string) string {
    t.Helper()
    path := filepath.Join(root, name)
    if err := os.WriteFile(path, []byte(value+"\n"), 0600); err != nil {
        t.Fatal(err)
    }
    return path
}

func TestDockgeCredentialIsScopedPerController(t *testing.T) {
    root := t.TempDir()
    pigePath := writeCredential(t, root, "pige.credential", "dockge-pige")
    connectPath := writeCredential(t, root, "connect.credential", "dockge-connect")
    globalPath := writeCredential(t, root, "global.credential", "dockge-global")

    runner := Runner{Config: config.Config{
        Dockge: config.Dockge{CredentialFile: globalPath},
    }}

    pige, err := runner.dockgeCredentialFor(config.Controller{DockgeCredentialFile: pigePath})
    if err != nil {
        t.Fatal(err)
    }
    connect, err := runner.dockgeCredentialFor(config.Controller{DockgeCredentialFile: connectPath})
    if err != nil {
        t.Fatal(err)
    }
    if pige != "dockge-pige" || connect != "dockge-connect" || pige == connect {
        t.Fatalf("scoped credentials not isolated: pige=%q connect=%q", pige, connect)
    }

    legacy, err := runner.dockgeCredentialFor(config.Controller{})
    if err != nil {
        t.Fatal(err)
    }
    if legacy != "dockge-global" {
        t.Fatalf("expected legacy global fallback, got %q", legacy)
    }
}

func TestDockgeCredentialMissingIsExplicitFailure(t *testing.T) {
    runner := Runner{Config: config.Config{}}
    if _, err := runner.dockgeCredentialFor(config.Controller{}); err == nil {
        t.Fatal("expected missing Dockge credential to fail explicitly")
    }
}
