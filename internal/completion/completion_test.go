package completion

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/config"
)

func newApp() (*app.App, *bytes.Buffer, *bytes.Buffer) {
	cfg := config.Default()
	a := app.New(&cfg, false)
	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	a.Out = out
	a.Err = errBuf
	return a, out, errBuf
}

func TestRun_Bash(t *testing.T) {
	a, out, _ := newApp()
	if err := Run(a, []string{"bash"}); err != nil {
		t.Fatalf("bash: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "_gist()") {
		t.Errorf("bash output missing _gist() function definition")
	}
	if !strings.Contains(got, "complete -F _gist gist") {
		t.Errorf("bash output missing complete registration for gist")
	}
	if strings.Contains(got, "_gist_overlay_git") {
		t.Errorf("bash without --alias=git unexpectedly contains overlay")
	}
}

func TestRun_BashAliasGit(t *testing.T) {
	a, out, _ := newApp()
	if err := Run(a, []string{"bash", "--alias=git"}); err != nil {
		t.Fatalf("bash --alias=git: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "_gist_overlay_git") {
		t.Errorf("bash --alias=git output missing overlay function")
	}
	if !strings.Contains(got, "complete -F _gist_overlay_git git") {
		t.Errorf("bash --alias=git output missing overlay registration")
	}
}

func TestRun_Zsh(t *testing.T) {
	a, out, _ := newApp()
	if err := Run(a, []string{"zsh"}); err != nil {
		t.Fatalf("zsh: %v", err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "#compdef gist") {
		t.Errorf("zsh script must start with #compdef gist (autoload tag)")
	}
	if !strings.Contains(got, "compdef _gist gist") {
		t.Errorf("zsh output missing compdef registration")
	}
}

func TestRun_ZshAliasGit(t *testing.T) {
	a, out, _ := newApp()
	if err := Run(a, []string{"zsh", "--alias=git"}); err != nil {
		t.Fatalf("zsh --alias=git: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "_gist_overlay_git") {
		t.Errorf("zsh --alias=git output missing overlay function")
	}
	if !strings.Contains(got, "compdef _gist_overlay_git git") {
		t.Errorf("zsh --alias=git output missing overlay compdef")
	}
}

func TestRun_BadShell(t *testing.T) {
	a, _, _ := newApp()
	err := Run(a, []string{"fish"})
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected unsupported-shell error, got %v", err)
	}
}

func TestRun_BadAlias(t *testing.T) {
	a, _, _ := newApp()
	err := Run(a, []string{"bash", "--alias=fish"})
	if err == nil || !strings.Contains(err.Error(), "--alias=fish") {
		t.Errorf("expected unsupported-alias error, got %v", err)
	}
}

func TestRun_NoArgs(t *testing.T) {
	a, _, _ := newApp()
	err := Run(a, nil)
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("expected usage error, got %v", err)
	}
}

// Sanity-check that the gist overlay command list in the embedded scripts
// stays in sync with the dispatch in main.go. If you add a new top-level
// subcommand, add it here too.
func TestEmbeddedCommandLists(t *testing.T) {
	wanted := []string{"status", "update", "branch", "switch", "remote", "fetch", "config", "legend", "version"}
	for _, cmd := range wanted {
		if !strings.Contains(bashScript, cmd) {
			t.Errorf("bash.sh missing command %q", cmd)
		}
		if !strings.Contains(zshScript, cmd) {
			t.Errorf("zsh.sh missing command %q", cmd)
		}
	}
}
