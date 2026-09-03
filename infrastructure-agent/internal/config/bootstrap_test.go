package config

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestConfigureFromEnvWritesConfigAndCredentialsSeparately(t *testing.T) {
    root := t.TempDir()
    cfgPath := filepath.Join(root, "etc", "agent.json")
    dataDir := filepath.Join(root, "data")
    t.Setenv("INFRA_AGENT_CONTROLLER_NAME", "pige360")
    t.Setenv("INFRA_AGENT_CONTROLLER_URL", "https://api.pige360.com.br")
    t.Setenv("INFRA_AGENT_ALLOWED_PREFIXES", "pige360-,mailcow-")
    t.Setenv("INFRA_AGENT_ENROLLMENT_TOKEN", "secret-enrollment")
    t.Setenv("INFRA_AGENT_DOCKGE_TOKEN", "secret-dockge")
    t.Setenv("INFRA_AGENT_DOCKGE_URL", "http://127.0.0.1:5001")
    t.Setenv("INFRA_AGENT_DATA_DIR", dataDir)

    cfg, err := ConfigureFromEnv(cfgPath)
    if err != nil {
        t.Fatal(err)
    }
    if len(cfg.Controllers) != 1 || cfg.Controllers[0].Name != "pige360" {
        t.Fatalf("unexpected controllers: %#v", cfg.Controllers)
    }
    raw, err := os.ReadFile(cfgPath)
    if err != nil {
        t.Fatal(err)
    }
    if strings.Contains(string(raw), "secret-enrollment") || strings.Contains(string(raw), "secret-dockge") {
        t.Fatal("credential leaked into agent.json")
    }
    enrollment, err := os.ReadFile(cfg.Controllers[0].EnrollmentFile)
    if err != nil {
        t.Fatal(err)
    }
    if strings.TrimSpace(string(enrollment)) != "secret-enrollment" {
        t.Fatal("enrollment credential was not persisted correctly")
    }
    dockgeCredential, err := os.ReadFile(cfg.Controllers[0].DockgeCredentialFile)
    if err != nil {
        t.Fatal(err)
    }
    if strings.TrimSpace(string(dockgeCredential)) != "secret-dockge" {
        t.Fatal("controller-scoped Dockge credential was not persisted correctly")
    }
    if cfg.Dockge.CredentialFile != "" {
        t.Fatalf("new config should not need legacy global Dockge credential, got %q", cfg.Dockge.CredentialFile)
    }
}

func TestConfigureFromEnvRejectsInsecureControlPlaneByDefault(t *testing.T) {
    root := t.TempDir()
    t.Setenv("INFRA_AGENT_CONTROLLER_NAME", "test")
    t.Setenv("INFRA_AGENT_CONTROLLER_URL", "http://control.example.test")
    t.Setenv("INFRA_AGENT_DATA_DIR", filepath.Join(root, "data"))
    _, err := ConfigureFromEnv(filepath.Join(root, "agent.json"))
    if err == nil || !strings.Contains(err.Error(), "HTTPS") {
        t.Fatalf("expected HTTPS validation error, got %v", err)
    }
}

func TestConfigureFromEnvAddsSecondControllerWithIndependentDockgeCredential(t *testing.T) {
    root := t.TempDir()
    cfgPath := filepath.Join(root, "etc", "agent.json")
    dataDir := filepath.Join(root, "data")

    t.Setenv("INFRA_AGENT_DATA_DIR", dataDir)
    t.Setenv("INFRA_AGENT_CONTROLLER_NAME", "pige360")
    t.Setenv("INFRA_AGENT_CONTROLLER_URL", "https://api.pige360.com.br")
    t.Setenv("INFRA_AGENT_ALLOWED_PREFIXES", "pige360-")
    t.Setenv("INFRA_AGENT_ENROLLMENT_TOKEN", "pige-token")
    t.Setenv("INFRA_AGENT_DOCKGE_TOKEN", "dockge-pige")
    if _, err := ConfigureFromEnv(cfgPath); err != nil {
        t.Fatal(err)
    }

    t.Setenv("INFRA_AGENT_CONTROLLER_NAME", "connect-api")
    t.Setenv("INFRA_AGENT_CONTROLLER_URL", "https://control.connect.example.com")
    t.Setenv("INFRA_AGENT_ALLOWED_PREFIXES", "connect-api-")
    t.Setenv("INFRA_AGENT_ENROLLMENT_TOKEN", "connect-token")
    t.Setenv("INFRA_AGENT_DOCKGE_TOKEN", "dockge-connect")
    cfg, err := ConfigureFromEnv(cfgPath)
    if err != nil {
        t.Fatal(err)
    }
    if len(cfg.Controllers) != 2 {
        t.Fatalf("expected 2 controllers, got %d", len(cfg.Controllers))
    }
    if cfg.Controllers[0].Name != "pige360" || cfg.Controllers[1].Name != "connect-api" {
        t.Fatalf("controllers not preserved: %#v", cfg.Controllers)
    }
    if cfg.Controllers[0].DockgeCredentialFile == cfg.Controllers[1].DockgeCredentialFile {
        t.Fatal("controllers must not share the same Dockge credential file")
    }
    first, err := os.ReadFile(cfg.Controllers[0].DockgeCredentialFile)
    if err != nil {
        t.Fatal(err)
    }
    second, err := os.ReadFile(cfg.Controllers[1].DockgeCredentialFile)
    if err != nil {
        t.Fatal(err)
    }
    if strings.TrimSpace(string(first)) != "dockge-pige" || strings.TrimSpace(string(second)) != "dockge-connect" {
        t.Fatalf("unexpected scoped credentials first=%q second=%q", first, second)
    }
}

func TestConfigureFromEnvPreservesExistingScopedDockgeCredentialWhenBlank(t *testing.T) {
    root := t.TempDir()
    cfgPath := filepath.Join(root, "etc", "agent.json")
    dataDir := filepath.Join(root, "data")

    t.Setenv("INFRA_AGENT_DATA_DIR", dataDir)
    t.Setenv("INFRA_AGENT_CONTROLLER_NAME", "pige360")
    t.Setenv("INFRA_AGENT_CONTROLLER_URL", "https://api.pige360.com.br")
    t.Setenv("INFRA_AGENT_ALLOWED_PREFIXES", "pige360-")
    t.Setenv("INFRA_AGENT_DOCKGE_TOKEN", "dockge-original")
    first, err := ConfigureFromEnv(cfgPath)
    if err != nil {
        t.Fatal(err)
    }

    t.Setenv("INFRA_AGENT_DOCKGE_TOKEN", "")
    t.Setenv("INFRA_AGENT_CONTROLLER_URL", "https://api2.pige360.com.br")
    second, err := ConfigureFromEnv(cfgPath)
    if err != nil {
        t.Fatal(err)
    }
    if second.Controllers[0].DockgeCredentialFile != first.Controllers[0].DockgeCredentialFile {
        t.Fatal("reconfiguration should preserve the existing scoped Dockge credential path")
    }
    raw, err := os.ReadFile(second.Controllers[0].DockgeCredentialFile)
    if err != nil {
        t.Fatal(err)
    }
    if strings.TrimSpace(string(raw)) != "dockge-original" {
        t.Fatal("blank reconfiguration must preserve existing Dockge credential")
    }
}
