package inventory

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestDockgeHTTPStatusRecognizesAuthenticatedAutomationAPI(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/v1/automation/health" {
            http.NotFound(w, r)
            return
        }
        if r.Header.Get("Authorization") != "Bearer dockge-secret" {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"ok":true,"service":"dockge","api":"v1","automation":true,"version":"1.6.0"}`))
    }))
    defer server.Close()

    reachable, api, version := dockgeHTTPStatus(server.URL, "dockge-secret")
    if !reachable || !api || version != "1.6.0" {
        t.Fatalf("unexpected status reachable=%v api=%v version=%q", reachable, api, version)
    }
}

func TestDockgeHTTPStatusDoesNotMistakeLegacyHTMLForAutomationAPI(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        _, _ = w.Write([]byte("<html><title>Dockge</title></html>"))
    }))
    defer server.Close()

    reachable, api, version := dockgeHTTPStatus(server.URL, "")
    if !reachable {
        t.Fatal("legacy/local endpoint should still be reported as reachable")
    }
    if api || version != "" {
        t.Fatalf("legacy HTML must not be reported as API-first: api=%v version=%q", api, version)
    }
}

func TestDockgeHTTPStatusTreatsServerErrorAsUnavailable(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusServiceUnavailable)
    }))
    defer server.Close()

    reachable, api, _ := dockgeHTTPStatus(server.URL, "")
    if reachable || api {
        t.Fatalf("expected unavailable status, got reachable=%v api=%v", reachable, api)
    }
}
