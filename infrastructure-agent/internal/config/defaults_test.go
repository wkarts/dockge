package config

import (
    "path/filepath"
    "testing"
)

func TestNativeDefaultPaths(t *testing.T) {
    tests := []struct {
        name       string
        goos       string
        programData string
        configPath string
        dataDir    string
    }{
        {
            name: "linux",
            goos: "linux",
            configPath: "/etc/infrastructure-agent/agent.json",
            dataDir: "/var/lib/infrastructure-agent",
        },
        {
            name: "macos",
            goos: "darwin",
            configPath: "/Library/Application Support/InfrastructureAgent/agent.json",
            dataDir: "/Library/Application Support/InfrastructureAgent/data",
        },
        {
            name: "windows",
            goos: "windows",
            programData: `C:\ProgramData`,
            configPath: filepath.Join(`C:\ProgramData`, "InfrastructureAgent", "agent.json"),
            dataDir: filepath.Join(`C:\ProgramData`, "InfrastructureAgent", "data"),
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := defaultConfigPath(tt.goos, tt.programData); got != tt.configPath {
                t.Fatalf("config path: got %q want %q", got, tt.configPath)
            }
            if got := defaultDataDir(tt.goos, tt.programData); got != tt.dataDir {
                t.Fatalf("data dir: got %q want %q", got, tt.dataDir)
            }
        })
    }
}
