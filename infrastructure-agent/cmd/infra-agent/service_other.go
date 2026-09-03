//go:build !windows

package main

import "github.com/wkarts/infrastructure-agent/internal/config"

func runAsPlatformService(_ config.Config) (bool, error) {
    return false, nil
}
