package model

import "time"

type DockgeInventory struct {
    Detected       bool   `json:"detected"`
    Reachable      bool   `json:"reachable"`
    AutomationAPI  bool   `json:"automation_api"`
    Version        string `json:"version,omitempty"`
    BaseURL        string `json:"base_url,omitempty"`
    ContainerName  string `json:"container_name,omitempty"`
    ContainerImage string `json:"container_image,omitempty"`
}

type Inventory struct {
    Hostname        string            `json:"hostname"`
    OS              string            `json:"os"`
    Arch            string            `json:"arch"`
    Kernel          string            `json:"kernel,omitempty"`
    Distro          string            `json:"distro,omitempty"`
    DistroVersion   string            `json:"distro_version,omitempty"`
    CPUs            int               `json:"cpus"`
    MemoryBytes     uint64            `json:"memory_bytes,omitempty"`
    DockerVersion   string            `json:"docker_version,omitempty"`
    ComposeVersion  string            `json:"compose_version,omitempty"`
    DockgeReachable bool              `json:"dockge_reachable"` // compatibility field
    Dockge           DockgeInventory   `json:"dockge"`
    Labels          map[string]string `json:"labels,omitempty"`
}

type EnrollmentRequest struct {
    AgentVersion string    `json:"agent_version"`
    Inventory    Inventory `json:"inventory"`
    Nonce        string    `json:"nonce"`
}

type EnrollmentResponse struct {
    AgentID        string `json:"agent_id"`
    AccessToken    string `json:"access_token"`
    PollSeconds    int    `json:"poll_seconds,omitempty"`
    TokenExpiresAt string `json:"token_expires_at,omitempty"`
}

type HeartbeatRequest struct {
    AgentVersion string    `json:"agent_version"`
    Inventory    Inventory `json:"inventory"`
    Timestamp    string    `json:"timestamp"`
}

type DesiredState struct {
    Revision string   `json:"revision"`
    Actions  []Action `json:"actions"`
}

type Action struct {
    ID         string         `json:"id"`
    Type       string         `json:"type"`
    Deployment string         `json:"deployment"`
    Payload    map[string]any `json:"payload,omitempty"`
    ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
}

type ActionResult struct {
    ActionID   string         `json:"action_id"`
    Status     string         `json:"status"`
    Message    string         `json:"message,omitempty"`
    StartedAt  time.Time      `json:"started_at"`
    FinishedAt time.Time      `json:"finished_at"`
    Details    map[string]any `json:"details,omitempty"`
}
