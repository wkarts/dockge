package journal

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sort"
    "sync"
    "time"

    "github.com/wkarts/infrastructure-agent/internal/model"
)

const maxEntries = 2048

type Entry struct {
    ActionID    string             `json:"action_id"`
    Result      model.ActionResult `json:"result"`
    ProcessedAt time.Time          `json:"processed_at"`
}

type fileFormat struct {
    Version int     `json:"version"`
    Entries []Entry `json:"entries"`
}

type Journal struct {
    path    string
    mu      sync.Mutex
    entries map[string]Entry
}

func Open(path string) (*Journal, error) {
    j := &Journal{path: path, entries: map[string]Entry{}}
    raw, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return j, nil
        }
        return nil, err
    }
    var data fileFormat
    if err := json.Unmarshal(raw, &data); err != nil {
        return nil, err
    }
    for _, entry := range data.Entries {
        if entry.ActionID != "" {
            j.entries[entry.ActionID] = entry
        }
    }
    return j, nil
}

func (j *Journal) Get(actionID string) (model.ActionResult, bool) {
    j.mu.Lock()
    defer j.mu.Unlock()
    entry, ok := j.entries[actionID]
    return entry.Result, ok
}

func (j *Journal) Put(result model.ActionResult) error {
    j.mu.Lock()
    defer j.mu.Unlock()

    j.entries[result.ActionID] = Entry{
        ActionID: result.ActionID,
        Result: result,
        ProcessedAt: time.Now().UTC(),
    }

    entries := make([]Entry, 0, len(j.entries))
    for _, entry := range j.entries {
        entries = append(entries, entry)
    }
    sort.Slice(entries, func(i, k int) bool {
        return entries[i].ProcessedAt.After(entries[k].ProcessedAt)
    })
    if len(entries) > maxEntries {
        entries = entries[:maxEntries]
    }

    compact := make(map[string]Entry, len(entries))
    for _, entry := range entries {
        compact[entry.ActionID] = entry
    }
    j.entries = compact

    if err := os.MkdirAll(filepath.Dir(j.path), 0700); err != nil {
        return err
    }
    raw, err := json.MarshalIndent(fileFormat{Version: 1, Entries: entries}, "", "  ")
    if err != nil {
        return err
    }
    raw = append(raw, '\n')
    tmp := j.path + ".tmp"
    if err := os.WriteFile(tmp, raw, 0600); err != nil {
        return err
    }
    if err := os.Rename(tmp, j.path); err != nil {
        _ = os.Remove(tmp)
        return err
    }
    _ = os.Chmod(j.path, 0600)
    return nil
}
