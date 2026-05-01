package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/branch"
	"github.com/ArturGulik/gist/internal/config"
	"github.com/ArturGulik/gist/internal/git"
	"github.com/ArturGulik/gist/internal/remote"
	"github.com/ArturGulik/gist/internal/render"
	"github.com/ArturGulik/gist/internal/update"
)

// Populated at build time via -ldflags. See Makefile and .goreleaser.yaml.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if path, err := config.Bootstrap(); err == nil && path != "" {
		fmt.Fprintf(os.Stderr, "gist: wrote default config to %s\n", path)
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gist: config: %v (using defaults)\n", err)
	}
	a := app.New(&cfg, ansi.DetectColor())

	if err := run(a, os.Args[1:]); err != nil {
		var ee *git.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.Code)
		}
		fmt.Fprintf(a.Err, "gist: %v\n", err)
		os.Exit(1)
	}
}

func run(a *app.App, args []string) error {
	cmd := "status"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "status", "s":
		return render.RunStatus(a, nil)
	case "update", "u":
		return update.RunUpdate(a, nil)
	case "branch", "b":
		return branch.RunBranch(a, args[1:])
	case "switch", "sw":
		return branch.RunSwitch(a, args[1:])
	case "remote", "r":
		return remote.RunRemote(a, nil)
	case "fetch":
		return update.RunFetch(a, args[1:])
	case "config":
		return a.RunConfig(nil)
	case "legend", "l":
		printLegend(a)
		return nil
	case "version", "--version", "-v":
		fmt.Fprintf(a.Out, "gist %s (%s, %s)\n", version, commit, date)
		return nil
	case "help", "--help", "-h":
		printHelp(a)
		return nil
	default:
		return git.Passthrough(args)
	}
}
