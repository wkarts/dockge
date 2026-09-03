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

func discoverDockgeContainer() (name string, image string) {
    out := command("docker", "ps", "--format", "{{.Names}}\t{{.Image}}")
    if out == "" {
        return "", ""
    }
    for _, line := range strings.Split(out, "\n") {
        parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
        if len(parts) != 2 {
            continue
        }
        candidateName := strings.TrimSpace(parts[0])
        candidateImage := strings.TrimSpace(parts[1])
        searchable := strings.ToLower(candidateName + " " + candidateImage)
        if strings.Contains(searchable, "dockge") {
            return candidateName, candidateImage
        }
    }
    return "", ""
}

func Collect(labels map[string]string, dockgeURL string, dockgeCredential string) model.Inventory {
    hostname, _ := os.Hostname()
    distro, distroVersion := osRelease()
    kernel := command("uname", "-r")
    docker := command("docker", "version", "--format", "{{.Server.Version}}")
    compose := command("docker", "compose", "version", "--short")

    reachable, automationAPI, version := dockgeHTTPStatus(dockgeURL, dockgeCredential)
    containerName, containerImage := discoverDockgeContainer()
    detected := automationAPI || containerName != "" || containerImage != ""

    dockge := model.DockgeInventory{
        Detected: detected,
        Reachable: reachable,
        AutomationAPI: automationAPI,
        Version: version,
        BaseURL: dockgeURL,
        ContainerName: containerName,
        ContainerImage: containerImage,
    }

    return model.Inventory{
        Hostname: hostname, OS: runtime.GOOS, Arch: runtime.GOARCH, Kernel: kernel,
        Distro: distro, DistroVersion: distroVersion, CPUs: runtime.NumCPU(), MemoryBytes: memoryBytes(),
        DockerVersion: docker, ComposeVersion: compose, DockgeReachable: reachable, Dockge: dockge, Labels: labels,
    }
}
