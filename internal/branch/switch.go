package branch

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/collect"
	"github.com/ArturGulik/gist/internal/git"
	"github.com/ArturGulik/gist/internal/render"
)

// RunSwitch shells out to `git switch` and, on success, replaces git's
// conventional "Switched to branch …" chatter with a single status line for
// the resulting current branch. On failure git's stdout/stderr are passed
// through verbatim and we exit with git's exit code.
func RunSwitch(a *app.App, args []string) error {
	if !git.InWorkTree() {
		fmt.Fprintln(a.Out, "not a git repository")
		return nil
	}

	cmd := exec.Command("git", append([]string{"switch"}, args...)...)
	cmd.Stdin = os.Stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_, _ = a.Out.Write(stdout.Bytes())
		_, _ = a.Err.Write(stderr.Bytes())
		return &git.ExitError{Code: git.ExitCodeFrom(cmd, 1)}
	}

	state, err := collect.RepoState(a.Color)
	if err != nil {
		return err
	}
	r := render.New(a)
	for _, b := range state.Branches {
		if b.Name == state.CurrentBranch {
			r.Branch(b)
			return nil
		}
	}
	// Detached HEAD or other odd case — fall back to git's own output.
	_, _ = a.Out.Write(stdout.Bytes())
	return nil
}
