package config

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"

    "github.com/wkarts/infrastructure-agent/internal/atomicfile"
    "github.com/wkarts/infrastructure-agent/internal/securefile"
)

// ConfigureFromEnv adds or updates one Control Plane binding while preserving
// all other bindings already configured on the host. Credentials are stored in
// dedicated files and are never embedded in agent.json.
func ConfigureFromEnv(configPath string) (Config, error) {
    name := strings.TrimSpace(os.Getenv("INFRA_AGENT_CONTROLLER_NAME"))
    baseURL := strings.TrimSpace(os.Getenv("INFRA_AGENT_CONTROLLER_URL"))
    if name == "" || baseURL == "" {
        return Config{}, errors.New("INFRA_AGENT_CONTROLLER_NAME and INFRA_AGENT_CONTROLLER_URL are required")
    }
    if !validName(name) {
        return Config{}, errors.New("controller name may contain only letters, numbers, dot, dash and underscore")
    }

    existing := Config{}
    if raw, err := os.ReadFile(configPath); err == nil {
        if err := json.Unmarshal(raw, &existing); err != nil {
            return Config{}, fmt.Errorf("existing config is invalid: %w", err)
        }
    } else if !os.IsNotExist(err) {
        return Config{}, err
    }

    var previousBinding *Controller
    for i := range existing.Controllers {
        if existing.Controllers[i].Name == name {
            copy := existing.Controllers[i]
            previousBinding = &copy
            break
        }
    }

    configDir := filepath.Dir(configPath)
    dataDir := strings.TrimSpace(os.Getenv("INFRA_AGENT_DATA_DIR"))
    if dataDir == "" {
        dataDir = existing.DataDir
    }
    if dataDir == "" {
        dataDir = DefaultDataDir()
    }

    secretDir := filepath.Join(configDir, "secrets")
    controllerDir := filepath.Join(dataDir, "controllers", name)
    enrollmentFile := filepath.Join(secretDir, name+"-enrollment.credential")
    credentialFile := filepath.Join(controllerDir, "access.credential")
    identityFile := filepath.Join(controllerDir, "agent.id")
    controllerDockgeCredentialFile := filepath.Join(controllerDir, "dockge.credential")

    if value := strings.TrimSpace(os.Getenv("INFRA_AGENT_ENROLLMENT_TOKEN")); value != "" {
        if err := securefile.Write(enrollmentFile, value); err != nil {
            return Config{}, err
        }
    } else if previousBinding != nil && previousBinding.EnrollmentFile != "" {
        enrollmentFile = previousBinding.EnrollmentFile
    }

    if value := strings.TrimSpace(os.Getenv("INFRA_AGENT_DOCKGE_TOKEN")); value != "" {
        if err := securefile.Write(controllerDockgeCredentialFile, value); err != nil {
            return Config{}, err
        }
    } else if previousBinding != nil && previousBinding.DockgeCredentialFile != "" {
        controllerDockgeCredentialFile = previousBinding.DockgeCredentialFile
    } else {
        // Existing installations may still rely on the legacy global Dockge
        // credential. Do not duplicate or expose it; runner fallback handles it.
        controllerDockgeCredentialFile = ""
    }

    binding := Controller{
        Name:                 name,
        BaseURL:              baseURL,
        EnrollmentFile:       enrollmentFile,
        CredentialFile:       credentialFile,
        AgentIdentityFile:    identityFile,
        DockgeCredentialFile: controllerDockgeCredentialFile,
        AllowedDeployments:   splitCSV(os.Getenv("INFRA_AGENT_ALLOWED_PREFIXES")),
        AllowInsecureHTTP:    envBool("INFRA_AGENT_ALLOW_INSECURE_HTTP", false),
    }
    controllers := make([]Controller, 0, len(existing.Controllers)+1)
    replaced := false
    for _, current := range existing.Controllers {
        if current.Name == name {
            controllers = append(controllers, binding)
            replaced = true
        } else {
            controllers = append(controllers, current)
        }
    }
    if !replaced {
        controllers = append(controllers, binding)
    }

    dockgeURL := strings.TrimSpace(os.Getenv("INFRA_AGENT_DOCKGE_URL"))
    if dockgeURL == "" {
        dockgeURL = existing.Dockge.BaseURL
    }
    if dockgeURL == "" {
        dockgeURL = "http://127.0.0.1:5001"
    }

    labels := map[string]string{}
    for key, value := range existing.Labels {
        labels[key] = value
    }
    labels["environment"] = envDefault("INFRA_AGENT_ENVIRONMENT", defaultValue(labels["environment"], "production"))
    labels["managed_by"] = "generic-infrastructure-agent"

    cfg := Config{
        DataDir:             dataDir,
        PollIntervalSeconds: envInt("INFRA_AGENT_POLL_SECONDS", defaultInt(existing.PollIntervalSeconds, 30)),
        Labels:              labels,
        Controllers:         controllers,
        Dockge: Dockge{
            BaseURL:          dockgeURL,
            CredentialFile:   existing.Dockge.CredentialFile,
            AllowNonLoopback: envBool("INFRA_AGENT_DOCKGE_ALLOW_NON_LOOPBACK", existing.Dockge.AllowNonLoopback),
        },
    }
    if err := cfg.Validate(); err != nil {
        return Config{}, err
    }
    if err := os.MkdirAll(configDir, 0750); err != nil {
        return Config{}, err
    }
    if err := os.MkdirAll(dataDir, 0700); err != nil {
        return Config{}, err
    }
    raw, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return Config{}, err
    }
    raw = append(raw, '\n')

    tempPath := configPath + ".tmp"
    if err := os.WriteFile(tempPath, raw, 0640); err != nil {
        return Config{}, err
    }
    if err := atomicfile.Replace(tempPath, configPath); err != nil {
        _ = os.Remove(tempPath)
        return Config{}, err
    }
    _ = os.Chmod(configPath, 0640)
    return cfg, nil
}

func validName(value string) bool {
    if value == "" {
        return false
    }
    for _, r := range value {
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
            continue
        }
        return false
    }
    return true
}

func splitCSV(value string) []string {
    out := make([]string, 0)
    for _, item := range strings.Split(value, ",") {
        item = strings.TrimSpace(item)
        if item != "" {
            out = append(out, item)
        }
    }
    return out
}

func envDefault(name, fallback string) string {
    if value := strings.TrimSpace(os.Getenv(name)); value != "" {
        return value
    }
    return fallback
}

func defaultValue(value, fallback string) string {
    if strings.TrimSpace(value) != "" {
        return value
    }
    return fallback
}

func defaultInt(value, fallback int) int {
    if value > 0 {
        return value
    }
    return fallback
}

func envBool(name string, fallback bool) bool {
    value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
    if value == "" {
        return fallback
    }
    return value == "1" || value == "true" || value == "yes" || value == "on"
}

func envInt(name string, fallback int) int {
    value := strings.TrimSpace(os.Getenv(name))
    if value == "" {
        return fallback
    }
    parsed, err := strconv.Atoi(value)
    if err != nil || parsed <= 0 {
        return fallback
    }
    return parsed
}
