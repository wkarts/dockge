package config

import (
    "strings"
    "testing"
)

func validController(name string) Controller {
    slug := strings.ToLower(name)
    return Controller{
        Name:               name,
        BaseURL:            "https://control.example.test",
        CredentialFile:     "/tmp/" + slug + "-access.credential",
        AgentIdentityFile:  "/tmp/" + slug + "-agent.id",
        AllowedDeployments: nil,
        AllowInsecureHTTP:  false,
    }
}

func validConfig(controllers ...Controller) Config {
    return Config{
        Controllers: controllers,
        Dockge:      Dockge{BaseURL: "http://127.0.0.1:5001"},
    }
}

func TestValidateRejectsDuplicateControllerNames(t *testing.T) {
    cfg := validConfig(
        validController("PIGE360"),
        validController("pige360"),
    )

    err := cfg.Validate()
    if err == nil {
        t.Fatal("expected duplicate controller-name validation error")
    }
    if !strings.Contains(err.Error(), "must be unique") {
        t.Fatalf("unexpected validation error: %v", err)
    }
}

func TestValidateAcceptsDistinctControllerNames(t *testing.T) {
    cfg := validConfig(
        validController("pige360"),
        validController("connect-api"),
    )

    if err := cfg.Validate(); err != nil {
        t.Fatalf("distinct controller names should validate: %v", err)
    }
}

func TestValidateRejectsControllerCredentialPathReuse(t *testing.T) {
    first := validController("pige360")
    second := validController("connect-api")
    second.CredentialFile = first.CredentialFile

    err := validConfig(first, second).Validate()
    if err == nil {
        t.Fatal("expected credential-path isolation validation error")
    }
    if !strings.Contains(err.Error(), "must be isolated") {
        t.Fatalf("unexpected validation error: %v", err)
    }
}

func TestValidateRejectsCrossRoleSensitivePathReuse(t *testing.T) {
    first := validController("pige360")
    second := validController("connect-api")
    second.AgentIdentityFile = first.CredentialFile

    err := validConfig(first, second).Validate()
    if err == nil {
        t.Fatal("expected cross-role sensitive-path validation error")
    }
    if !strings.Contains(err.Error(), "must be isolated") {
        t.Fatalf("unexpected validation error: %v", err)
    }
}

func TestValidateRejectsUnsupportedControllerScheme(t *testing.T) {
    ctl := validController("pige360")
    ctl.BaseURL = "ftp://control.example.test"
    ctl.AllowInsecureHTTP = true

    err := validConfig(ctl).Validate()
    if err == nil {
        t.Fatal("expected unsupported controller scheme validation error")
    }
    if !strings.Contains(err.Error(), "must use HTTPS") {
        t.Fatalf("unexpected validation error: %v", err)
    }
}

func TestValidateAllowsExplicitInsecureHTTPController(t *testing.T) {
    ctl := validController("lab")
    ctl.BaseURL = "http://127.0.0.1:8080"
    ctl.AllowInsecureHTTP = true

    if err := validConfig(ctl).Validate(); err != nil {
        t.Fatalf("explicit insecure HTTP controller should validate: %v", err)
    }
}

func TestValidateRejectsUnsupportedDockgeScheme(t *testing.T) {
    cfg := validConfig(validController("pige360"))
    cfg.Dockge.BaseURL = "ftp://127.0.0.1:5001"

    err := cfg.Validate()
    if err == nil {
        t.Fatal("expected unsupported Dockge scheme validation error")
    }
    if !strings.Contains(err.Error(), "HTTP or HTTPS") {
        t.Fatalf("unexpected validation error: %v", err)
    }
}
