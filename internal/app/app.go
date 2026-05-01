package app

import (
	"fmt"
	"io"
	"os"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/config"
)

// App is the ambient context every subcommand receives. main() builds one
// at startup; subcommands read Cfg/Color/Pen and write to Out/Err. Tests
// build their own with a bytes.Buffer for Out.
type App struct {
	Cfg   *config.Config
	Color bool
	Pen   ansi.Pen
	Out   io.Writer
	Err   io.Writer
}

// New constructs an App with stdout/stderr as the default writers and the
// pen color flag derived from the bool. Callers can override Out/Err in tests.
func New(cfg *config.Config, color bool) *App {
	return &App{
		Cfg:   cfg,
		Color: color,
		Pen:   ansi.Pen{Color: color},
		Out:   os.Stdout,
		Err:   os.Stderr,
	}
}

// RunConfig prints the canonical default config to Out. Useful as
// `gist config > ~/.config/gist/config` to reset to defaults, or to inspect
// the full surface of available knobs.
func (a *App) RunConfig(_ []string) error {
	fmt.Fprint(a.Out, config.DefaultText)
	return nil
}
