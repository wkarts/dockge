package config

import (
    "encoding/json"
    "errors"
    "fmt"
    "net/url"
    "os"
    "path/filepath"
    "runtime"
    "strings"
)

type Controller struct {
    Name                 string   `json:"name"`
    BaseURL              string   `json:"base_url"`
    EnrollmentFile       string   `json:"enrollment_file,omitempty"`
    CredentialFile       string   `json:"credential_file"`
    AgentIdentityFile    string   `json:"agent_identity_file"`
    DockgeCredentialFile string   `json:"dockge_credential_file,omitempty"`
    AllowedDeployments   []string `json:"allowed_deployment_prefixes,omitempty"`
    AllowInsecureHTTP    bool     `json:"allow_insecure_http,omitempty"`
}

type Dockge struct {
    BaseURL          string `json:"base_url"`
    CredentialFile   string `json:"credential_file,omitempty"` // legacy/global fallback
    AllowNonLoopback bool   `json:"allow_non_loopback,omitempty"`
}

type Config struct {
    DataDir             string            `json:"data_dir"`
    PollIntervalSeconds int               `json:"poll_interval_seconds"`
    Labels              map[string]string `json:"labels,omitempty"`
    Controllers         []Controller      `json:"controllers"`
    Dockge              Dockge            `json:"dockge"`
}

func defaultConfigPath(goos, programData string) string {
    switch goos {
    case "windows":
        if programData != "" {
            return filepath.Join(programData, "InfrastructureAgent", "agent.json")
        }
        return filepath.Join(".", "agent.json")
    case "darwin":
        return "/Library/Application Support/InfrastructureAgent/agent.json"
    default:
        return "/etc/infrastructure-agent/agent.json"
    }
}

func defaultDataDir(goos, programData string) string {
    switch goos {
    case "windows":
        if programData != "" {
            return filepath.Join(programData, "InfrastructureAgent", "data")
        }
        return filepath.Join(".", "data")
    case "darwin":
        return "/Library/Application Support/InfrastructureAgent/data"
    default:
        return "/var/lib/infrastructure-agent"
    }
}

func DefaultPath() string {
    if v := os.Getenv("INFRA_AGENT_CONFIG"); v != "" {
        return v
    }
    return defaultConfigPath(runtime.GOOS, os.Getenv("ProgramData"))
}

func DefaultDataDir() string {
    return defaultDataDir(runtime.GOOS, os.Getenv("ProgramData"))
}

func Load(path string) (Config, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return Config{}, err
    }
    var cfg Config
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return Config{}, err
    }
    if cfg.DataDir == "" {
        cfg.DataDir = DefaultDataDir()
    }
    if cfg.PollIntervalSeconds <= 0 {
        cfg.PollIntervalSeconds = 30
    }
    if cfg.Dockge.BaseURL == "" {
        cfg.Dockge.BaseURL = "http://127.0.0.1:5001"
    }
    if err := cfg.Validate(); err != nil {
        return Config{}, err
    }
    return cfg, nil
}

func normalizedSensitivePath(value string) string {
    clean := filepath.Clean(strings.TrimSpace(value))
    if runtime.GOOS == "windows" {
        clean = strings.ToLower(clean)
    }
    return clean
}

func registerSensitivePath(seen map[string]string, value, label string) error {
    if strings.TrimSpace(value) == "" {
        return nil
    }
    normalized := normalizedSensitivePath(value)
    if previous, exists := seen[normalized]; exists {
        return fmt.Errorf("%s reuses sensitive file already assigned to %s; controller credentials and identities must be isolated", label, previous)
    }
    seen[normalized] = label
    return nil
}

func (cfg Config) Validate() error {
    if len(cfg.Controllers) == 0 {
        return errors.New("at least one controller is required")
    }

    seenControllerNames := make(map[string]int, len(cfg.Controllers))
    seenSensitivePaths := make(map[string]string, len(cfg.Controllers)*4)
    for i, ctl := range cfg.Controllers {
        name := strings.TrimSpace(ctl.Name)
        if name == "" || ctl.BaseURL == "" {
            return fmt.Errorf("controllers[%d] name and base_url are required", i)
        }
        normalizedName := strings.ToLower(name)
        if previousIndex, exists := seenControllerNames[normalizedName]; exists {
            return fmt.Errorf("controllers[%d].name duplicates controllers[%d].name (%q); controller names must be unique", i, previousIndex, name)
        }
        seenControllerNames[normalizedName] = i

        u, err := url.Parse(ctl.BaseURL)
        if err != nil || u.Host == "" {
            return fmt.Errorf("controllers[%d].base_url is invalid", i)
        }
        if u.Scheme != "https" {
            if !ctl.AllowInsecureHTTP || u.Scheme != "http" {
                return fmt.Errorf("controllers[%d] must use HTTPS; HTTP requires allow_insecure_http=true", i)
            }
        }
        if ctl.CredentialFile == "" || ctl.AgentIdentityFile == "" {
            return fmt.Errorf("controllers[%d] credential and identity files are required", i)
        }
        if len(ctl.AllowedDeployments) > 0 && ctl.DockgeCredentialFile == "" && cfg.Dockge.CredentialFile == "" {
            return fmt.Errorf("controllers[%d] needs a Dockge credential for deployment actions", i)
        }

        sensitiveFiles := []struct {
            value string
            role  string
        }{
            {ctl.EnrollmentFile, "enrollment_file"},
            {ctl.CredentialFile, "credential_file"},
            {ctl.AgentIdentityFile, "agent_identity_file"},
            {ctl.DockgeCredentialFile, "dockge_credential_file"},
        }
        for _, sensitive := range sensitiveFiles {
            if err := registerSensitivePath(seenSensitivePaths, sensitive.value, fmt.Sprintf("controllers[%d].%s", i, sensitive.role)); err != nil {
                return err
            }
        }
    }

    u, err := url.Parse(cfg.Dockge.BaseURL)
    if err != nil || u.Host == "" {
        return errors.New("dockge.base_url is invalid")
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return errors.New("dockge.base_url must use HTTP or HTTPS")
    }
    host := strings.ToLower(u.Hostname())
    if !cfg.Dockge.AllowNonLoopback && host != "127.0.0.1" && host != "localhost" && host != "::1" {
        return errors.New("dockge.base_url must be loopback unless allow_non_loopback=true")
    }
    return nil
}
