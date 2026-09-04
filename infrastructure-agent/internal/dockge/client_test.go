package dockge

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestApplyStackSendsAuthenticationAndIdempotencyKey(t *testing.T) {
    var received map[string]any
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPut {
            t.Errorf("unexpected method: %s", r.Method)
        }
        if r.URL.Path != "/api/v1/automation/stacks/pige360-school" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        if got := r.Header.Get("Authorization"); got != "Bearer dockge-secret" {
            t.Errorf("unexpected authorization header: %q", got)
        }
        if got := r.Header.Get("Idempotency-Key"); got != "act-001" {
            t.Errorf("unexpected idempotency key: %q", got)
        }
        if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
            t.Errorf("decode body: %v", err)
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"ok":true}`))
    }))
    defer server.Close()

    client := New(server.URL, "dockge-secret")
    err := client.ApplyStack(context.Background(), "pige360-school", map[string]any{
        "compose_yaml": "services:\n  app:\n    image: example/app:1\n",
    }, "act-001")
    if err != nil {
        t.Fatal(err)
    }
    if received["compose_yaml"] == nil {
        t.Fatalf("compose payload was not sent: %#v", received)
    }
}

func TestActionOmitsIdempotencyHeaderWhenKeyIsBlank(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if got := r.Header.Get("Idempotency-Key"); got != "" {
            t.Errorf("blank key must not emit Idempotency-Key header, got %q", got)
        }
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    client := New(server.URL, "dockge-secret")
    if err := client.Action(context.Background(), "stack-a", "up", map[string]any{}, ""); err != nil {
        t.Fatal(err)
    }
}
