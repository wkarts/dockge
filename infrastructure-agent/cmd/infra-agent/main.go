package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/wkarts/infrastructure-agent/internal/agent"
    "github.com/wkarts/infrastructure-agent/internal/config"
    "github.com/wkarts/infrastructure-agent/internal/inventory"
    v "github.com/wkarts/infrastructure-agent/internal/version"
)

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
    r := agent.Runner{Config: cfg}
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    switch cmd {
    case "run":
        err = r.Run(ctx)
    case "enroll":
        err = r.Enroll(ctx)
    case "once":
        err = r.Once(ctx)
    case "inventory":
        b, _ := json.MarshalIndent(inventory.Collect(cfg.Labels, cfg.Dockge.BaseURL), "", "  ")
        fmt.Println(string(b))
        return
    case "doctor":
        inv := inventory.Collect(cfg.Labels, cfg.Dockge.BaseURL)
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
