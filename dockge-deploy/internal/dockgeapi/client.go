package dockgeapi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiPath = "/api/v1/automation"

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type Error struct {
	StatusCode int
	Body       []byte
}

func (e *Error) Error() string {
	return fmt.Sprintf("Dockge API returned HTTP %d: %s", e.StatusCode, strings.TrimSpace(string(e.Body)))
}

func New(baseURL, token string, allowHTTP, insecureTLS bool) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("Dockge bearer token is required; set DOCKGE_DEPLOY_DOCKGE_TOKEN")
	}
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return nil, errors.New("Dockge URL must be an absolute http(s) URL")
	}
	if parsed.Scheme == "http" && !allowHTTP && !isLoopbackHost(parsed.Hostname()) {
		return nil, errors.New("plain HTTP Dockge URLs are blocked; use HTTPS or pass --allow-http explicitly for a trusted private network")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecureTLS {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		transport.TLSClientConfig.InsecureSkipVerify = true // explicit CLI opt-in for labs only
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/") + apiPath,
		token:   token,
		http: &http.Client{
			Timeout:   45 * time.Second,
			Transport: transport,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func NewIdempotencyKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate idempotency key: %w", err)
	}
	return "dockge-deploy-" + hex.EncodeToString(buf), nil
}

func (c *Client) request(method, path string, body any, idempotencyKey string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Dockge API request failed: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, fmt.Errorf("Dockge API redirect blocked: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, &Error{StatusCode: resp.StatusCode, Body: payload}
	}
	return payload, nil
}

func (c *Client) Health() ([]byte, error) { return c.request(http.MethodGet, "/health", nil, "") }
func (c *Client) Info() ([]byte, error) { return c.request(http.MethodGet, "/info", nil, "") }
func (c *Client) Stacks() ([]byte, error) { return c.request(http.MethodGet, "/stacks", nil, "") }
func (c *Client) Stack(name string) ([]byte, error) {
	return c.request(http.MethodGet, "/stacks/"+url.PathEscape(name), nil, "")
}
func (c *Client) Logs(name string, tail int) ([]byte, error) {
	return c.request(http.MethodGet, fmt.Sprintf("/stacks/%s/logs?tail=%d", url.PathEscape(name), tail), nil, "")
}
func (c *Client) Action(name, action, idempotencyKey string) ([]byte, error) {
	return c.request(http.MethodPost, "/stacks/"+url.PathEscape(name)+"/actions/"+url.PathEscape(action), map[string]any{}, idempotencyKey)
}
func (c *Client) Apply(name, composeYAML string, composeEnv *string, adopt bool, idempotencyKey string) ([]byte, error) {
	body := map[string]any{"compose_yaml": composeYAML, "adopt": adopt, "owner": "dockge-deploy"}
	if composeEnv != nil {
		body["compose_env"] = *composeEnv
	}
	return c.request(http.MethodPut, "/stacks/"+url.PathEscape(name), body, idempotencyKey)
}
