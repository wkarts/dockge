package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"

    "github.com/wkarts/infrastructure-agent/internal/agent"
    "github.com/wkarts/infrastructure-agent/internal/config"
    "github.com/wkarts/infrastructure-agent/internal/inventory"
    "github.com/wkarts/infrastructure-agent/internal/model"
    "github.com/wkarts/infrastructure-agent/internal/securefile"
    v "github.com/wkarts/infrastructure-agent/internal/version"
)

func collectInventory(cfg config.Config) model.Inventory {
    credential, _ := securefile.Read(cfg.Dockge.CredentialFile)
    return inventory.Collect(cfg.Labels, cfg.Dockge.BaseURL, credential)
}

func main() {
    cfgPath := flag.String("config", config.DefaultPath(), "path to agent.json")
    flag.Parse()
    cmd := "run"
    if flag.NArg() > 0 {
        cmd = flag.Arg(0)
    }
    if cmd == "version" {
        fmt.Printf("infra-agent %s commit=%s date=%s\n", v.Version, v.Commit, v.Date)
        return
    }
    if cmd == "configure" {
        cfg, err := config.ConfigureFromEnv(*cfgPath)
        if err != nil {
            log.Fatalf("configure: %v", err)
        }
        fmt.Printf("Configuration written: %s\nControllers: %d\nDockge: %s\n", *cfgPath, len(cfg.Controllers), cfg.Dockge.BaseURL)
        return
    }
    cfg, err := config.Load(*cfgPath)
    if err != nil {
        log.Fatalf("config: %v", err)
    }

    // On Windows the same binary acts as a real SCM service when started by
    // Service Control Manager, while remaining a normal CLI when launched by
    // a human terminal or installer.
    if cmd == "run" {
        handled, serviceErr := runAsPlatformService(cfg)
        if handled {
            if serviceErr != nil {
                log.Fatalf("service: %v", serviceErr)
            }
            return
        }
    }

    r := agent.Runner{Config: cfg}
    ctx, cancel := signal.NotifyContext(context.Background(), platformSignals()...)
    defer cancel()
    switch cmd {
    case "run":
        err = r.Run(ctx)
    case "enroll":
        err = r.Enroll(ctx)
    case "once":
        err = r.Once(ctx)
    case "inventory":
        b, _ := json.MarshalIndent(collectInventory(cfg), "", "  ")
        fmt.Println(string(b))
        return
    case "doctor":
        inv := collectInventory(cfg)
        b, _ := json.MarshalIndent(inv, "", "  ")
        fmt.Println(string(b))
        if inv.DockerVersion == "" {
            os.Exit(2)
        }
        return
    default:
        log.Fatalf("unknown command %q; use run|configure|enroll|once|inventory|doctor|version", cmd)
    }
    if err != nil && err != context.Canceled {
        log.Fatal(err)
    }
}
