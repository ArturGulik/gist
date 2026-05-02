package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/branch"
	"github.com/ArturGulik/gist/internal/completion"
	"github.com/ArturGulik/gist/internal/config"
	"github.com/ArturGulik/gist/internal/git"
	"github.com/ArturGulik/gist/internal/remote"
	"github.com/ArturGulik/gist/internal/render"
	"github.com/ArturGulik/gist/internal/update"
)

// Populated at build time via -ldflags (see Makefile and .goreleaser.yaml).
// When ldflags weren't applied — e.g. `go install github.com/ArturGulik/gist@vX.Y.Z`
// or a plain `go build .` from a clone — the init() below fills these from the
// build info embedded by the Go toolchain so `gist version` still prints
// something useful.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	if version != "dev" {
		return // ldflags applied; trust them
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		version = strings.TrimPrefix(v, "v")
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				commit = s.Value[:7]
			}
		case "vcs.time":
			date = s.Value
		case "vcs.modified":
			// Go itself stamps "+dirty" onto Main.Version when the working
			// tree is dirty at a tagged commit. Only append manually if it
			// isn't there already (the no-tag-at-HEAD case).
			if s.Value == "true" && !strings.HasSuffix(version, "+dirty") {
				version += "+dirty"
			}
		}
	}
}

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
	case "status":
		return render.RunStatus(a, nil)
	case "update":
		return update.RunUpdate(a, nil)
	case "branch":
		return branch.RunBranch(a, args[1:])
	case "switch":
		return branch.RunSwitch(a, args[1:])
	case "remote":
		if len(args) > 1 {
			return git.Passthrough(append([]string{"remote"}, args[1:]...))
		}
		return remote.RunRemote(a, nil)
	case "fetch":
		return update.RunFetch(a, args[1:])
	case "config":
		return a.RunConfig(nil)
	case "completion":
		return completion.Run(a, args[1:])
	case "legend":
		printLegend(a)
		return nil
	case "version", "--version", "-v":
		// `go install <module>@<version>` builds from a proxy-served source
		// zip that has no `.git`, so the toolchain can't populate vcs.revision
		// or vcs.time. In that case print the version on its own rather than
		// the noisy "(none, unknown)" parenthetical.
		if commit == "none" || date == "unknown" {
			fmt.Fprintf(a.Out, "gist %s\n", version)
		} else {
			fmt.Fprintf(a.Out, "gist %s (%s, %s)\n", version, commit, date)
		}
		return nil
	case "help", "--help", "-h":
		printHelp(a)
		return nil
	default:
		return git.Passthrough(args)
	}
}
