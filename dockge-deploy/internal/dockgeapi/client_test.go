package dockgeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSendsBearerAndIdempotency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/automation/stacks/demo/actions/restart" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("unexpected authorization: %q", got)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "op-123" {
			t.Fatalf("unexpected idempotency key: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "secret-token", true, false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := client.Action("demo", "restart", "op-123")
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil || body["ok"] != true {
		t.Fatalf("unexpected body: %s (%v)", payload, err)
	}
}

func TestPlainHTTPRequiresExplicitOptIn(t *testing.T) {
	if _, err := New("http://example.test:5001", "secret-token", false, false); err == nil {
		t.Fatal("expected plain HTTP to be rejected")
	}
	if _, err := New("http://127.0.0.1:5001", "secret-token", false, false); err != nil {
		t.Fatalf("loopback HTTP should be allowed: %v", err)
	}
}

func TestRedirectIsBlocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://example.invalid/steal", http.StatusFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "secret-token", true, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Health()
	if err == nil || !strings.Contains(err.Error(), "redirect blocked") {
		t.Fatalf("expected redirect to be blocked, got %v", err)
	}
}

func TestIdempotencyKeysAreUnique(t *testing.T) {
	one, err := NewIdempotencyKey()
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewIdempotencyKey()
	if err != nil {
		t.Fatal(err)
	}
	if one == two || !strings.HasPrefix(one, "dockge-deploy-") {
		t.Fatalf("unexpected keys: %q %q", one, two)
	}
}
