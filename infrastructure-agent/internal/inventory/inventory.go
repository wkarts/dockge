package inventory

import (
    "bufio"
    "context"
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

func dockgeReachable(baseURL string) bool {
    client := http.Client{Timeout: 2 * time.Second}
    req, _ := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/automation/health", nil)
    res, err := client.Do(req)
    if err != nil {
        return false
    }
    defer res.Body.Close()
    // 401/403 still proves that the local Dockge API is alive; availability
    // is distinct from authentication state.
    return res.StatusCode >= 200 && res.StatusCode < 500
}

func Collect(labels map[string]string, dockgeURL string) model.Inventory {
    hostname, _ := os.Hostname()
    distro, distroVersion := osRelease()
    kernel := command("uname", "-r")
    docker := command("docker", "version", "--format", "{{.Server.Version}}")
    compose := command("docker", "compose", "version", "--short")
    return model.Inventory{
        Hostname: hostname, OS: runtime.GOOS, Arch: runtime.GOARCH, Kernel: kernel,
        Distro: distro, DistroVersion: distroVersion, CPUs: runtime.NumCPU(), MemoryBytes: memoryBytes(),
        DockerVersion: docker, ComposeVersion: compose, DockgeReachable: dockgeReachable(dockgeURL), Labels: labels,
    }
}
