package dockge

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type Client struct {
    BaseURL string
    Token   string
    HTTP    *http.Client
}

func New(baseURL, token string) *Client {
    return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{Timeout: 2 * time.Minute}}
}

func (c *Client) do(ctx context.Context, method, path string, in any, out any, idempotencyKey string) error {
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
    if strings.TrimSpace(idempotencyKey) != "" {
        req.Header.Set("Idempotency-Key", idempotencyKey)
    }
    res, err := c.HTTP.Do(req)
    if err != nil {
        return err
    }
    defer res.Body.Close()
    b, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
    if res.StatusCode < 200 || res.StatusCode >= 300 {
        return fmt.Errorf("dockge %s %s: HTTP %d: %s", method, path, res.StatusCode, strings.TrimSpace(string(b)))
    }
    if out != nil && len(b) > 0 {
        return json.Unmarshal(b, out)
    }
    return nil
}

func (c *Client) ApplyStack(ctx context.Context, deployment string, payload map[string]any, idempotencyKey string) error {
    return c.do(ctx, http.MethodPut, "/api/v1/automation/stacks/"+deployment, payload, nil, idempotencyKey)
}

func (c *Client) DeleteStack(ctx context.Context, deployment, idempotencyKey string) error {
    return c.do(ctx, http.MethodDelete, "/api/v1/automation/stacks/"+deployment, nil, nil, idempotencyKey)
}

func (c *Client) Action(ctx context.Context, deployment, action string, payload map[string]any, idempotencyKey string) error {
    return c.do(ctx, http.MethodPost, "/api/v1/automation/stacks/"+deployment+"/actions/"+action, payload, nil, idempotencyKey)
}
