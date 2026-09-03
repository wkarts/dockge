package agent

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "github.com/wkarts/infrastructure-agent/internal/config"
    "github.com/wkarts/infrastructure-agent/internal/controlplane"
    "github.com/wkarts/infrastructure-agent/internal/dockge"
    "github.com/wkarts/infrastructure-agent/internal/inventory"
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

func (r Runner) controller(ctl config.Controller) (*controlplane.Client, string, error) {
    token, err := securefile.Read(ctl.AccessTokenFile)
    if err != nil {
        return nil, "", err
    }
    id, err := securefile.Read(ctl.AgentIDFile)
    if err != nil {
        return nil, "", err
    }
    return controlplane.New(ctl.BaseURL, token), id, nil
}

func (r Runner) Enroll(ctx context.Context) error {
    inv := inventory.Collect(r.Config.Labels, r.Config.Dockge.BaseURL)
    for _, ctl := range r.Config.Controllers {
        if token, err := securefile.Read(ctl.AccessTokenFile); err == nil && token != "" {
            continue
        }
        enrollToken, err := securefile.Read(ctl.EnrollmentTokenFile)
        if err != nil {
            return fmt.Errorf("%s enrollment token: %w", ctl.Name, err)
        }
        cp := controlplane.New(ctl.BaseURL, enrollToken)
        out, err := cp.Enroll(ctx, v.Version, inv)
        if err != nil {
            return fmt.Errorf("%s enroll: %w", ctl.Name, err)
        }
        if out.AgentID == "" || out.AccessToken == "" {
            return fmt.Errorf("%s enroll returned incomplete credentials", ctl.Name)
        }
        if err := securefile.Write(ctl.AgentIDFile, out.AgentID); err != nil {
            return err
        }
        if err := securefile.Write(ctl.AccessTokenFile, out.AccessToken); err != nil {
            return err
        }
        // Bootstrap enrollment tokens are one-time credentials. After a
        // successful exchange they are removed locally; only the scoped
        // access token remains on disk.
        if ctl.EnrollmentTokenFile != "" {
            _ = os.Remove(ctl.EnrollmentTokenFile)
        }
        log.Printf("controller=%s enrolled agent_id=%s", ctl.Name, out.AgentID)
    }
    return nil
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
    token, err := securefile.Read(r.Config.Dockge.TokenFile)
    if err != nil {
        result.Message = "dockge token unavailable: " + err.Error()
        return result
    }
    dc := dockge.New(r.Config.Dockge.BaseURL, token)
    switch action.Type {
    case "dockge.stack.apply":
        err = dc.ApplyStack(ctx, action.Deployment, action.Payload)
    case "dockge.stack.delete":
        err = dc.DeleteStack(ctx, action.Deployment)
    case "dockge.stack.pull", "dockge.stack.up", "dockge.stack.down", "dockge.stack.restart", "dockge.stack.stop", "dockge.stack.start":
        err = dc.Action(ctx, action.Deployment, strings.TrimPrefix(action.Type, "dockge.stack."), action.Payload)
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
    inv := inventory.Collect(r.Config.Labels, r.Config.Dockge.BaseURL)
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
        for _, a := range desired.Actions {
            res := r.execute(ctx, ctl, a)
            if err := cp.Report(ctx, id, res); err != nil {
                errs = append(errs, fmt.Errorf("%s report %s: %w", ctl.Name, a.ID, err))
            }
        }
    }
    return errors.Join(errs...)
}

func (r Runner) Run(ctx context.Context) error {
    if err := r.Enroll(ctx); err != nil {
        return err
    }
    ticker := time.NewTicker(time.Duration(r.Config.PollIntervalSeconds) * time.Second)
    defer ticker.Stop()
    for {
        if err := r.Once(ctx); err != nil {
            log.Printf("cycle error: %v", err)
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
        }
    }
}
