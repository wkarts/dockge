//go:build windows

package main

import (
    "context"
    "errors"
    "time"

    "golang.org/x/sys/windows/svc"

    "github.com/wkarts/infrastructure-agent/internal/agent"
    "github.com/wkarts/infrastructure-agent/internal/config"
)

const windowsServiceName = "InfrastructureAgent"

type windowsService struct {
    cfg config.Config
}

func (s *windowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
    const accepted = svc.AcceptStop | svc.AcceptShutdown
    status <- svc.Status{State: svc.StartPending}

    ctx, cancel := context.WithCancel(context.Background())
    done := make(chan error, 1)
    go func() {
        done <- (agent.Runner{Config: s.cfg}).Run(ctx)
    }()

    status <- svc.Status{State: svc.Running, Accepts: accepted}

    for {
        select {
        case change := <-requests:
            switch change.Cmd {
            case svc.Interrogate:
                status <- change.CurrentStatus
            case svc.Stop, svc.Shutdown:
                status <- svc.Status{State: svc.StopPending}
                cancel()
                select {
                case <-done:
                case <-time.After(20 * time.Second):
                }
                return false, 0
            default:
                // Unsupported SCM controls are intentionally ignored.
            }
        case err := <-done:
            cancel()
            if err != nil && !errors.Is(err, context.Canceled) {
                return false, 1
            }
            return false, 0
        }
    }
}

func runAsPlatformService(cfg config.Config) (bool, error) {
    isService, err := svc.IsWindowsService()
    if err != nil {
        return false, err
    }
    if !isService {
        return false, nil
    }
    return true, svc.Run(windowsServiceName, &windowsService{cfg: cfg})
}
