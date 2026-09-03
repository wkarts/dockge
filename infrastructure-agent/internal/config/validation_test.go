package config

import (
    "strings"
    "testing"
)

func validController(name string) Controller {
    return Controller{
        Name:                name,
        BaseURL:             "https://control.example.test",
        CredentialFile:      "/tmp/access.credential",
        AgentIdentityFile:   "/tmp/agent.id",
        AllowedDeployments:  nil,
        AllowInsecureHTTP:   false,
    }
}

func TestValidateRejectsDuplicateControllerNames(t *testing.T) {
    cfg := Config{
        Controllers: []Controller{
            validController("PIGE360"),
            validController("pige360"),
        },
        Dockge: Dockge{BaseURL: "http://127.0.0.1:5001"},
    }

    err := cfg.Validate()
    if err == nil {
        t.Fatal("expected duplicate controller-name validation error")
    }
    if !strings.Contains(err.Error(), "must be unique") {
        t.Fatalf("unexpected validation error: %v", err)
    }
}

func TestValidateAcceptsDistinctControllerNames(t *testing.T) {
    cfg := Config{
        Controllers: []Controller{
            validController("pige360"),
            validController("connect-api"),
        },
        Dockge: Dockge{BaseURL: "http://127.0.0.1:5001"},
    }

    if err := cfg.Validate(); err != nil {
        t.Fatalf("distinct controller names should validate: %v", err)
    }
}
