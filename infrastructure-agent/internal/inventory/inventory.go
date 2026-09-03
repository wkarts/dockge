package inventory

import (
    "bufio"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "os/exec"
    "runtime"
    "strconv"
    "strings"
    "time"

    "github.com/wkarts/infrastructure-agent/internal/model"
)

func command(name string, args ...string) string {
    ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
    defer cancel()
    b, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(b))
}

func osRelease() (string, string) {
    f, err := os.Open("/etc/os-release")
    if err != nil {
        return "", ""
    }
    defer f.Close()
    vals := map[string]string{}
    s := bufio.NewScanner(f)
    for s.Scan() {
        parts := strings.SplitN(s.Text(), "=", 2)
        if len(parts) == 2 {
            vals[parts[0]] = strings.Trim(parts[1], "\"")
        }
    }
    return vals["ID"], vals["VERSION_ID"]
}

func memoryBytes() uint64 {
    if runtime.GOOS != "linux" {
        return 0
    }
    f, err := os.Open("/proc/meminfo")
    if err != nil {
        return 0
    }
    defer f.Close()
    s := bufio.NewScanner(f)
    for s.Scan() {
        if strings.HasPrefix(s.Text(), "MemTotal:") {
            fs := strings.Fields(s.Text())
            if len(fs) >= 2 {
                if kb, err := strconv.ParseUint(fs[1], 10, 64); err == nil {
                    return kb * 1024
                }
            }
        }
    }
    return 0
}

type dockgeHealth struct {
    OK         bool   `json:"ok"`
    Service    string `json:"service"`
    API        string `json:"api"`
    Automation bool   `json:"automation"`
    Version    string `json:"version"`
}

func dockgeHTTPStatus(baseURL, credential string) (reachable bool, automationAPI bool, version string) {
    client := http.Client{Timeout: 2 * time.Second}
    req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/automation/health", nil)
    if err != nil {
        return false, false, ""
    }
    req.Header.Set("Accept", "application/json")
    if strings.TrimSpace(credential) != "" {
        req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credential))
    }

    res, err := client.Do(req)
    if err != nil {
        return false, false, ""
    }
    defer res.Body.Close()

    // Any non-5xx response proves something is listening at the configured
    // local endpoint. Only a valid authenticated JSON health payload proves
    // that the API-first automation contract is actually available.
    reachable = res.StatusCode >= 200 && res.StatusCode < 500
    if res.StatusCode != http.StatusOK {
        return reachable, false, ""
    }

    raw, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
    if err != nil {
        return reachable, false, ""
    }
    var health dockgeHealth
    if err := json.Unmarshal(raw, &health); err != nil {
        return reachable, false, ""
    }
    automationAPI = health.OK && strings.EqualFold(health.Service, "dockge") && health.API == "v1" && health.Automation
    if automationAPI {
        version = health.Version
    }
    return reachable, automationAPI, version
}

func discoverDockgeContainers() []model.DockgeContainer {
    // Include stopped containers: an existing provider installation must be
    // visible to the Control Plane even when it is not currently running.
    out := command("docker", "ps", "-a", "--format", "{{.Names}}\t{{.Image}}\t{{.State}}")
    if out == "" {
        return nil
    }

    containers := make([]model.DockgeContainer, 0)
    for _, line := range strings.Split(out, "\n") {
        parts := strings.SplitN(strings.TrimSpace(line), "\t", 3)
        if len(parts) < 2 {
            continue
        }
        candidate := model.DockgeContainer{
            Name: strings.TrimSpace(parts[0]),
            Image: strings.TrimSpace(parts[1]),
        }
        if len(parts) == 3 {
            candidate.State = strings.TrimSpace(parts[2])
        }
        searchable := strings.ToLower(candidate.Name + " " + candidate.Image)
        if strings.Contains(searchable, "dockge") {
            containers = append(containers, candidate)
        }
    }
    return containers
}

func Collect(labels map[string]string, dockgeURL string, dockgeCredential string) model.Inventory {
    hostname, _ := os.Hostname()
    distro, distroVersion := osRelease()
    kernel := command("uname", "-r")
    docker := command("docker", "version", "--format", "{{.Server.Version}}")
    compose := command("docker", "compose", "version", "--short")

    reachable, automationAPI, version := dockgeHTTPStatus(dockgeURL, dockgeCredential)
    containers := discoverDockgeContainers()
    detected := automationAPI || len(containers) > 0

    dockge := model.DockgeInventory{
        Detected: detected,
        Reachable: reachable,
        AutomationAPI: automationAPI,
        Version: version,
        BaseURL: dockgeURL,
        Containers: containers,
    }
    if len(containers) > 0 {
        dockge.ContainerName = containers[0].Name
        dockge.ContainerImage = containers[0].Image
    }

    return model.Inventory{
        Hostname: hostname, OS: runtime.GOOS, Arch: runtime.GOARCH, Kernel: kernel,
        Distro: distro, DistroVersion: distroVersion, CPUs: runtime.NumCPU(), MemoryBytes: memoryBytes(),
        DockerVersion: docker, ComposeVersion: compose, DockgeReachable: reachable, Dockge: dockge, Labels: labels,
    }
}
