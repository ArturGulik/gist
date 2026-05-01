package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExitError signals that main should exit with a specific code without
// printing anything additional — used when a child process (git) has already
// produced user-facing output and we just need to propagate its exit code.
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return fmt.Sprintf("exit %d", e.Code) }

// ExitCodeFrom extracts a non-negative exit code from a finished cmd, or
// returns fallback if the process never started.
func ExitCodeFrom(cmd *exec.Cmd, fallback int) int {
	if cmd.ProcessState == nil {
		return fallback
	}
	if ec := cmd.ProcessState.ExitCode(); ec >= 0 {
		return ec
	}
	return fallback
}

// Run runs git with the given args and returns stdout trimmed of the
// trailing newline. On failure stderr is attached to the error.
func Run(args ...string) (string, error) {
	out, err := RunRaw(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// RunRaw returns stdout bytes verbatim — use when preserving color codes
// or any other non-text output.
func RunRaw(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, &gitError{args: args, stderr: errb.String(), err: err}
	}
	return out.Bytes(), nil
}

type gitError struct {
	args   []string
	stderr string
	err    error
}

func (e *gitError) Error() string {
	msg := strings.TrimSpace(e.stderr)
	if msg == "" {
		msg = e.err.Error()
	}
	return "git " + strings.Join(e.args, " ") + ": " + msg
}

// Passthrough runs git with the given args, inheriting stdin/stdout/stderr
// so interactive commands (editors, pagers, prompts) work correctly. Returns
// an *ExitError carrying git's exit code; main propagates it without adding
// any further output.
func Passthrough(args []string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &ExitError{Code: ExitCodeFrom(cmd, 1)}
	}
	return nil
}

func InWorkTree() bool {
	_, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

func HasCommits() bool {
	_, err := Run("rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// CurrentBranch returns the name of the current branch, or "" if HEAD
// is detached.
func CurrentBranch() string {
	s, err := Run("symbolic-ref", "--short", "--quiet", "HEAD")
	if err != nil {
		return ""
	}
	return s
}

// WebURL converts a git remote URL to a browser-openable HTTPS URL.
// Returns "" if the format is not recognized.
func WebURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)

	// SCP-style SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		s := strings.TrimPrefix(rawURL, "git@")
		colon := strings.IndexByte(s, ':')
		if colon < 0 {
			return ""
		}
		host := s[:colon]
		path := strings.TrimSuffix(s[colon+1:], ".git")
		return "https://" + host + "/" + path
	}

	// ssh:// protocol: ssh://git@github.com/owner/repo.git
	if strings.HasPrefix(rawURL, "ssh://") {
		s := strings.TrimPrefix(rawURL, "ssh://")
		if at := strings.IndexByte(s, '@'); at >= 0 {
			s = s[at+1:]
		}
		// strip :port if present before the first /
		if slash := strings.IndexByte(s, '/'); slash >= 0 {
			hostPort := s[:slash]
			if colon := strings.IndexByte(hostPort, ':'); colon >= 0 {
				s = hostPort[:colon] + s[slash:]
			}
		}
		return "https://" + strings.TrimSuffix(s, ".git")
	}

	// HTTPS / HTTP: strip embedded credentials and .git suffix
	if strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://") {
		rest := rawURL
		// normalise to https
		if strings.HasPrefix(rest, "http://") {
			rest = "https://" + strings.TrimPrefix(rest, "http://")
		}
		// strip user:pass@host → host
		schemeEnd := strings.Index(rest, "://") + 3
		hostAndPath := rest[schemeEnd:]
		if at := strings.IndexByte(hostAndPath, '@'); at >= 0 {
			hostAndPath = hostAndPath[at+1:]
		}
		return "https://" + strings.TrimSuffix(hostAndPath, ".git")
	}

	return ""
}
