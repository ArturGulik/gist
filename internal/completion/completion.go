// Package completion serves the static shell-completion scripts shipped
// with gist. The scripts are embedded at build time; this package just
// chooses which one(s) to print based on `gist completion <shell> [--alias=<name>]`.
package completion

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/ArturGulik/gist/internal/app"
)

//go:embed bash.sh
var bashScript string

//go:embed bash_overlay.sh
var bashOverlay string

//go:embed zsh.sh
var zshScript string

//go:embed zsh_overlay.sh
var zshOverlay string

// Run handles `gist completion <shell> [--alias=<name>]`. The only supported
// alias today is `git`, which appends an overlay block that registers a
// completer for the `git` command.
func Run(a *app.App, args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "install":
		return Install(a, args[1:])
	case "uninstall":
		return Uninstall(a, args[1:])
	}
	shell := args[0]
	var alias string
	for _, arg := range args[1:] {
		switch {
		case strings.HasPrefix(arg, "--alias="):
			alias = strings.TrimPrefix(arg, "--alias=")
		case arg == "-h" || arg == "--help":
			return usage()
		default:
			return fmt.Errorf("completion: unknown argument %q", arg)
		}
	}
	if alias != "" && alias != "git" {
		return fmt.Errorf("completion: --alias=%s not supported (only --alias=git)", alias)
	}

	var script, overlay string
	switch shell {
	case "bash":
		script, overlay = bashScript, bashOverlay
	case "zsh":
		script, overlay = zshScript, zshOverlay
	default:
		return fmt.Errorf("completion: unsupported shell %q (supported: bash, zsh)", shell)
	}

	fmt.Fprint(a.Out, script)
	if alias == "git" {
		fmt.Fprint(a.Out, overlay)
	}
	return nil
}

func usage() error {
	return fmt.Errorf("usage: gist completion <bash|zsh> [--alias=git]\n" +
		"       gist completion install   [--shell=bash|zsh] [--alias=git]\n" +
		"       gist completion uninstall [--shell=bash|zsh]")
}
