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
    Name               string   `json:"name"`
    BaseURL            string   `json:"base_url"`
    EnrollmentFile     string   `json:"enrollment_file,omitempty"`
    CredentialFile     string   `json:"credential_file"`
    AgentIdentityFile  string   `json:"agent_identity_file"`
    AllowedDeployments []string `json:"allowed_deployment_prefixes,omitempty"`
    AllowInsecureHTTP  bool     `json:"allow_insecure_http,omitempty"`
}

type Dockge struct {
    BaseURL          string `json:"base_url"`
    CredentialFile   string `json:"credential_file"`
    AllowNonLoopback bool   `json:"allow_non_loopback,omitempty"`
}

type Config struct {
    DataDir             string            `json:"data_dir"`
    PollIntervalSeconds int               `json:"poll_interval_seconds"`
    Labels              map[string]string `json:"labels,omitempty"`
    Controllers         []Controller      `json:"controllers"`
    Dockge              Dockge            `json:"dockge"`
}

func DefaultPath() string {
    if v := os.Getenv("INFRA_AGENT_CONFIG"); v != "" {
        return v
    }
    if runtime.GOOS == "windows" {
        if pd := os.Getenv("ProgramData"); pd != "" {
            return filepath.Join(pd, "InfrastructureAgent", "agent.json")
        }
        return filepath.Join(".", "agent.json")
    }
    return "/etc/infrastructure-agent/agent.json"
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
        cfg.DataDir = "/var/lib/infrastructure-agent"
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

func (cfg Config) Validate() error {
    if len(cfg.Controllers) == 0 {
        return errors.New("at least one controller is required")
    }
    for i, ctl := range cfg.Controllers {
        if ctl.Name == "" || ctl.BaseURL == "" {
            return fmt.Errorf("controllers[%d] name and base_url are required", i)
        }
        u, err := url.Parse(ctl.BaseURL)
        if err != nil || u.Host == "" {
            return fmt.Errorf("controllers[%d].base_url is invalid", i)
        }
        if u.Scheme != "https" && !ctl.AllowInsecureHTTP {
            return fmt.Errorf("controllers[%d] must use HTTPS", i)
        }
        if ctl.CredentialFile == "" || ctl.AgentIdentityFile == "" {
            return fmt.Errorf("controllers[%d] credential and identity files are required", i)
        }
    }
    u, err := url.Parse(cfg.Dockge.BaseURL)
    if err != nil || u.Host == "" {
        return errors.New("dockge.base_url is invalid")
    }
    host := strings.ToLower(u.Hostname())
    if !cfg.Dockge.AllowNonLoopback && host != "127.0.0.1" && host != "localhost" && host != "::1" {
        return errors.New("dockge.base_url must be loopback unless allow_non_loopback=true")
    }
    return nil
}
