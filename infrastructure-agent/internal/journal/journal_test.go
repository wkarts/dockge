package journal

import (
    "path/filepath"
    "testing"
    "time"

    "github.com/wkarts/infrastructure-agent/internal/model"
)

func TestJournalPersistsAndReplaysResult(t *testing.T) {
    path := filepath.Join(t.TempDir(), "action-journal.json")
    first, err := Open(path)
    if err != nil {
        t.Fatal(err)
    }

    result := model.ActionResult{
        ActionID: "action-123",
        Status: "succeeded",
        Message: "done",
        StartedAt: time.Unix(100, 0).UTC(),
        FinishedAt: time.Unix(101, 0).UTC(),
    }
    if err := first.Put("control-a:action-123", result); err != nil {
        t.Fatal(err)
    }

    reopened, err := Open(path)
    if err != nil {
        t.Fatal(err)
    }
    cached, ok := reopened.Get("control-a:action-123")
    if !ok {
        t.Fatal("expected action result to be persisted")
    }
    if cached.ActionID != result.ActionID || cached.Status != "succeeded" || cached.Message != "done" {
        t.Fatalf("unexpected cached result: %#v", cached)
    }
    if _, ok := reopened.Get("control-b:action-123"); ok {
        t.Fatal("journal must isolate equal action IDs from different control planes")
    }
}
