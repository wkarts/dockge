package agent

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/wkarts/infrastructure-agent/internal/config"
    "github.com/wkarts/infrastructure-agent/internal/controlplane"
    "github.com/wkarts/infrastructure-agent/internal/dockge"
    "github.com/wkarts/infrastructure-agent/internal/inventory"
    "github.com/wkarts/infrastructure-agent/internal/journal"
    "github.com/wkarts/infrastructure-agent/internal/model"
    "github.com/wkarts/infrastructure-agent/internal/securefile"
    v "github.com/wkarts/infrastructure-agent/internal/version"
)

type Runner struct{ Config config.Config }

func allowed(name string, prefixes []string) bool {
    if name == "" {
        return false
    }
    for _, p := range prefixes {
        if p != "" && strings.HasPrefix(name, p) {
            return true
        }
    }
    return false
}

func readOptionalCredential(path string) string {
    if strings.TrimSpace(path) == "" {
        return ""
    }
    credential, _ := securefile.Read(path)
    return credential
}

func (r Runner) dockgeCredentialFor(ctl config.Controller) (string, error) {
    path := ctl.DockgeCredentialFile
    if path == "" {
        path = r.Config.Dockge.CredentialFile
    }
    if path == "" {
        return "", errors.New("no Dockge credential configured for this controller")
    }
    return securefile.Read(path)
}

func (r Runner) inventory() model.Inventory {
    credential := ""
    for _, ctl := range r.Config.Controllers {
        if value := readOptionalCredential(ctl.DockgeCredentialFile); value != "" {
            credential = value
            break
        }
    }
    if credential == "" {
        credential = readOptionalCredential(r.Config.Dockge.CredentialFile)
    }
    return inventory.Collect(r.Config.Labels, r.Config.Dockge.BaseURL, credential)
}

func (r Runner) controller(ctl config.Controller) (*controlplane.Client, string, error) {
    credential, err := securefile.Read(ctl.CredentialFile)
    if err != nil {
        return nil, "", err
    }
    id, err := securefile.Read(ctl.AgentIdentityFile)
    if err != nil {
        return nil, "", err
    }
    return controlplane.New(ctl.BaseURL, credential), id, nil
}

func (r Runner) Enroll(ctx context.Context) error {
    inv := r.inventory()
    var errs []error

    for _, ctl := range r.Config.Controllers {
        if credential, err := securefile.Read(ctl.CredentialFile); err == nil && credential != "" {
            continue
        }

        enrollmentCredential, err := securefile.Read(ctl.EnrollmentFile)
        if err != nil {
            errs = append(errs, fmt.Errorf("%s enrollment credential: %w", ctl.Name, err))
            continue
        }

        cp := controlplane.New(ctl.BaseURL, enrollmentCredential)
        out, err := cp.Enroll(ctx, v.Version, inv)
        if err != nil {
            errs = append(errs, fmt.Errorf("%s enroll: %w", ctl.Name, err))
            continue
        }
        if out.AgentID == "" || out.AccessToken == "" {
            errs = append(errs, fmt.Errorf("%s enroll returned incomplete credentials", ctl.Name))
            continue
        }
        if err := securefile.Write(ctl.AgentIdentityFile, out.AgentID); err != nil {
            errs = append(errs, fmt.Errorf("%s agent identity: %w", ctl.Name, err))
            continue
        }
        if err := securefile.Write(ctl.CredentialFile, out.AccessToken); err != nil {
            errs = append(errs, fmt.Errorf("%s access credential: %w", ctl.Name, err))
            continue
        }
        if ctl.EnrollmentFile != "" {
            _ = os.Remove(ctl.EnrollmentFile)
        }
        log.Printf("controller=%s enrolled agent_id=%s", ctl.Name, out.AgentID)
    }

    return errors.Join(errs...)
}

func (r Runner) execute(ctx context.Context, ctl config.Controller, action model.Action) model.ActionResult {
    started := time.Now().UTC()
    result := model.ActionResult{ActionID: action.ID, Status: "failed", StartedAt: started}
    defer func() { result.FinishedAt = time.Now().UTC() }()

    if action.ExpiresAt != nil && time.Now().After(*action.ExpiresAt) {
        result.Status = "expired"
        result.Message = "action expired"
        return result
    }
    if !allowed(action.Deployment, ctl.AllowedDeployments) {
        result.Status = "denied"
        result.Message = "deployment outside controller scope"
        return result
    }

    credential, err := r.dockgeCredentialFor(ctl)
    if err != nil {
        result.Message = "dockge credential unavailable: " + err.Error()
        return result
    }

    dc := dockge.New(r.Config.Dockge.BaseURL, credential)
    switch action.Type {
    case "dockge.stack.apply":
        err = dc.ApplyStack(ctx, action.Deployment, action.Payload, action.ID)
    case "dockge.stack.delete":
        err = dc.DeleteStack(ctx, action.Deployment, action.ID)
    case "dockge.stack.pull", "dockge.stack.up", "dockge.stack.down", "dockge.stack.restart", "dockge.stack.stop", "dockge.stack.start":
        err = dc.Action(ctx, action.Deployment, strings.TrimPrefix(action.Type, "dockge.stack."), action.Payload, action.ID)
    case "noop":
        result.Status = "succeeded"
        result.Message = "noop"
        return result
    default:
        result.Status = "denied"
        result.Message = "unsupported typed action"
        return result
    }

    if err != nil {
        result.Message = err.Error()
        return result
    }
    result.Status = "succeeded"
    return result
}

func (r Runner) Once(ctx context.Context) error {
    inv := r.inventory()
    actionJournal, err := journal.Open(filepath.Join(r.Config.DataDir, "action-journal.json"))
    if err != nil {
        return fmt.Errorf("open action journal: %w", err)
    }

    var errs []error
    for _, ctl := range r.Config.Controllers {
        cp, id, err := r.controller(ctl)
        if err != nil {
            errs = append(errs, fmt.Errorf("%s credentials: %w", ctl.Name, err))
            continue
        }
        if err := cp.Heartbeat(ctx, id, v.Version, inv); err != nil {
            errs = append(errs, fmt.Errorf("%s heartbeat: %w", ctl.Name, err))
            continue
        }

        desired, err := cp.DesiredState(ctx, id)
        if err != nil {
            errs = append(errs, fmt.Errorf("%s desired state: %w", ctl.Name, err))
            continue
        }

        for _, action := range desired.Actions {
            if strings.TrimSpace(action.ID) == "" {
                errs = append(errs, fmt.Errorf("%s desired action without action_id", ctl.Name))
                continue
            }

            journalKey := ctl.Name + ":" + action.ID
            result, alreadyProcessed := actionJournal.Get(journalKey)
            if alreadyProcessed {
                log.Printf("controller=%s action_id=%s replay detected; reporting cached result status=%s", ctl.Name, action.ID, result.Status)
            } else {
                result = r.execute(ctx, ctl, action)
                if err := actionJournal.Put(journalKey, result); err != nil {
                    // The Dockge API also receives action.ID as Idempotency-Key.
                    // If this local journal write fails after the host changed,
                    // the same action can be retried without blindly repeating
                    // a completed destructive mutation.
                    errs = append(errs, fmt.Errorf("%s journal action %s: %w", ctl.Name, action.ID, err))
                }
            }

            if err := cp.Report(ctx, id, result); err != nil {
                errs = append(errs, fmt.Errorf("%s report %s: %w", ctl.Name, action.ID, err))
            }
        }
    }
    return errors.Join(errs...)
}

func (r Runner) Run(ctx context.Context) error {
    ticker := time.NewTicker(time.Duration(r.Config.PollIntervalSeconds) * time.Second)
    defer ticker.Stop()

    for {
        if err := r.Enroll(ctx); err != nil {
            log.Printf("enrollment cycle error: %v", err)
        }
        if err := r.Once(ctx); err != nil {
            log.Printf("reconciliation cycle error: %v", err)
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
        }
    }
}
