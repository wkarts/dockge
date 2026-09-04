package main

import (
	"fmt"
	"os"

	"github.com/wkarts/dockge/dockge-deploy/internal/commands"
)

func main() {
	if err := commands.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
