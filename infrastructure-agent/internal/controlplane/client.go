package controlplane

import (
    "bytes"
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"

    "github.com/wkarts/infrastructure-agent/internal/model"
)

type Client struct {
    BaseURL string
    Token   string
    HTTP    *http.Client
}

func New(baseURL, token string) *Client {
    return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

func nonce() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    return hex.EncodeToString(b)
}

func (c *Client) do(ctx context.Context, method, path string, in any, out any) error {
    var body io.Reader
    if in != nil {
        b, err := json.Marshal(in)
        if err != nil {
            return err
        }
        body = bytes.NewReader(b)
    }
    req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
    if err != nil {
        return err
    }
    req.Header.Set("Accept", "application/json")
    if in != nil {
        req.Header.Set("Content-Type", "application/json")
    }
    if c.Token != "" {
        req.Header.Set("Authorization", "Bearer "+c.Token)
    }
    res, err := c.HTTP.Do(req)
    if err != nil {
        return err
    }
    defer res.Body.Close()
    data, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
    if res.StatusCode < 200 || res.StatusCode >= 300 {
        return fmt.Errorf("control plane %s %s: HTTP %d: %s", method, path, res.StatusCode, strings.TrimSpace(string(data)))
    }
    if out != nil && len(data) > 0 {
        return json.Unmarshal(data, out)
    }
    return nil
}

func (c *Client) Enroll(ctx context.Context, version string, inv model.Inventory) (model.EnrollmentResponse, error) {
    var out model.EnrollmentResponse
    err := c.do(ctx, http.MethodPost, "/api/v1/infrastructure/agents/enroll", model.EnrollmentRequest{AgentVersion: version, Inventory: inv, Nonce: nonce()}, &out)
    return out, err
}

func (c *Client) Heartbeat(ctx context.Context, agentID, version string, inv model.Inventory) error {
    return c.do(ctx, http.MethodPost, "/api/v1/infrastructure/agents/"+agentID+"/heartbeat", model.HeartbeatRequest{AgentVersion: version, Inventory: inv, Timestamp: time.Now().UTC().Format(time.RFC3339)}, nil)
}

func (c *Client) DesiredState(ctx context.Context, agentID string) (model.DesiredState, error) {
    var out model.DesiredState
    err := c.do(ctx, http.MethodGet, "/api/v1/infrastructure/agents/"+agentID+"/desired-state", nil, &out)
    return out, err
}

func (c *Client) Report(ctx context.Context, agentID string, result model.ActionResult) error {
    return c.do(ctx, http.MethodPost, "/api/v1/infrastructure/agents/"+agentID+"/actions/"+result.ActionID+"/result", result, nil)
}
