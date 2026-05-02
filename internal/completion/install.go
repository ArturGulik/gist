package completion

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ArturGulik/gist/internal/app"
)

// rcDelim wraps the auto-managed block in .bashrc / .zshrc so subsequent
// installs can find and replace the block in place rather than appending
// duplicates.
const rcDelim = "####-gist-command-completion-####"

// Install handles `gist completion install [--shell=<bash|zsh>] [--alias=git]`.
// File-drop lands at the user-level XDG completion dir (bash) or a $fpath
// candidate (zsh), so no shell-rcfile edit is needed for the base completion.
// `--alias=git` additionally appends a delimited block to the user's rcfile
// that registers the git-overlay completer.
func Install(a *app.App, args []string) error {
	var shell, alias string
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--shell="):
			shell = strings.TrimPrefix(arg, "--shell=")
		case strings.HasPrefix(arg, "--alias="):
			alias = strings.TrimPrefix(arg, "--alias=")
		case arg == "-h", arg == "--help":
			return installUsage()
		default:
			return fmt.Errorf("completion install: unknown argument %q", arg)
		}
	}
	if alias != "" && alias != "git" {
		return fmt.Errorf("completion install: --alias=%s not supported (only --alias=git)", alias)
	}
	if shell == "" {
		shell = detectShell()
		if shell == "" {
			return errors.New("completion install: cannot detect shell from $SHELL; pass --shell=bash or --shell=zsh")
		}
	}

	switch shell {
	case "bash":
		return installBash(a, alias == "git")
	case "zsh":
		return installZsh(a, alias == "git")
	default:
		return fmt.Errorf("completion install: unsupported shell %q (supported: bash, zsh)", shell)
	}
}

func installUsage() error {
	return fmt.Errorf("usage: gist completion install [--shell=bash|zsh] [--alias=git]")
}

// detectShell returns "bash", "zsh", or "" based on $SHELL.
func detectShell() string {
	switch filepath.Base(os.Getenv("SHELL")) {
	case "bash":
		return "bash"
	case "zsh":
		return "zsh"
	}
	return ""
}

func installBash(a *app.App, withGitOverlay bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("completion install: %w", err)
	}
	dest := filepath.Join(home, ".local", "share", "bash-completion", "completions", "gist")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("completion install: %w", err)
	}
	if err := os.WriteFile(dest, []byte(bashScript), 0o644); err != nil {
		return fmt.Errorf("completion install: %w", err)
	}
	fmt.Fprintf(a.Out, "wrote bash completion → %s\n", dest)

	if withGitOverlay {
		rc := filepath.Join(home, ".bashrc")
		block := []string{
			"# Eagerly load git's completion so the overlay can wrap it.",
			"declare -F _completion_loader >/dev/null && _completion_loader git",
			"source <(gist completion bash --alias=git)",
		}
		action, err := upsertRcBlock(rc, block)
		if err != nil {
			return fmt.Errorf("completion install: %w", err)
		}
		fmt.Fprintf(a.Out, "%s git-overlay block in %s\n", action, rc)
	}

	fmt.Fprintln(a.Out, "Restart your shell (or open a new one) to activate.")
	return nil
}

func installZsh(a *app.App, withGitOverlay bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("completion install: %w", err)
	}
	dir, inFpath := chooseZshDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("completion install: %w", err)
	}
	dest := filepath.Join(dir, "_gist")
	if err := os.WriteFile(dest, []byte(zshScript), 0o644); err != nil {
		return fmt.Errorf("completion install: %w", err)
	}
	fmt.Fprintf(a.Out, "wrote zsh completion → %s\n", dest)
	if !inFpath {
		fmt.Fprintf(a.Out, "note: %s is not in $fpath. Add this to your zshrc BEFORE compinit:\n", dir)
		fmt.Fprintf(a.Out, "  fpath=(%s $fpath)\n", dir)
	}

	if withGitOverlay {
		rc := zshRcPath(home)
		block := []string{
			"(( ${+functions[compdef]} )) && source <(gist completion zsh --alias=git)",
		}
		action, err := upsertRcBlock(rc, block)
		if err != nil {
			return fmt.Errorf("completion install: %w", err)
		}
		fmt.Fprintf(a.Out, "%s git-overlay block in %s\n", action, rc)
	}

	fmt.Fprintln(a.Out, "Restart your shell (or open a new one) to activate.")
	return nil
}

// zshRcPath honors $ZDOTDIR if set, falling back to ~/.zshrc.
func zshRcPath(home string) string {
	if z := os.Getenv("ZDOTDIR"); z != "" {
		return filepath.Join(z, ".zshrc")
	}
	return filepath.Join(home, ".zshrc")
}

// chooseZshDir picks an fpath dir to write _gist into. If we can probe
// $fpath via `zsh -c` and find a writable user dir already on it, we use
// that; otherwise we default to ${ZDOTDIR:-$HOME}/.zsh/completions and
// flag inFpath=false so the caller can warn the user.
func chooseZshDir(home string) (dir string, inFpath bool) {
	defaultDir := filepath.Join(home, ".zsh", "completions")
	if z := os.Getenv("ZDOTDIR"); z != "" {
		defaultDir = filepath.Join(z, "completions")
	}

	out, err := exec.Command("zsh", "-c", "print -lr -- $fpath").Output()
	if err != nil {
		return defaultDir, false
	}
	candidates := []string{defaultDir}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, home) {
			continue
		}
		candidates = append(candidates, line)
	}
	for _, c := range candidates {
		if c == defaultDir {
			continue
		}
		if writable(c) {
			return c, true
		}
	}
	return defaultDir, dirInFpathOutput(defaultDir, string(out))
}

func writable(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	probe := filepath.Join(dir, ".gist-write-probe")
	f, err := os.Create(probe)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(probe)
	return true
}

func dirInFpathOutput(dir, fpathOut string) bool {
	for _, line := range strings.Split(fpathOut, "\n") {
		if strings.TrimSpace(line) == dir {
			return true
		}
	}
	return false
}

// upsertRcBlock writes lines into path, wrapped in rcDelim markers. If a
// previous gist-managed block (delimited by rcDelim on its own line) is
// present, replace it in place; otherwise append a fresh one.
//
// Returns "wrote", "updated", or "appended" depending on what happened.
func upsertRcBlock(path string, lines []string) (string, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	var newBlock strings.Builder
	newBlock.WriteString(rcDelim + "\n")
	for _, l := range lines {
		newBlock.WriteString(l + "\n")
	}
	newBlock.WriteString(rcDelim + "\n")

	start, end, ok := findRcBlock(string(existing))
	var result []byte
	var action string
	switch {
	case len(existing) == 0:
		result = []byte(newBlock.String())
		action = "wrote"
	case ok:
		result = append(append([]byte{}, existing[:start]...), newBlock.String()...)
		result = append(result, existing[end:]...)
		action = "updated"
	default:
		var prefix string
		if existing[len(existing)-1] != '\n' {
			prefix = "\n"
		}
		prefix += "\n"
		result = append(append([]byte{}, existing...), []byte(prefix+newBlock.String())...)
		action = "appended"
	}
	if err := os.WriteFile(path, result, 0o644); err != nil {
		return "", err
	}
	return action, nil
}

// Uninstall handles `gist completion uninstall [--shell=<bash|zsh>] [--alias=git]`.
// Removes the dropped completion file and, if present, the delimited rc block.
// The --alias=git flag is accepted for symmetry with `install` but is a no-op:
// uninstall always also strips the rc block when it finds one.
func Uninstall(a *app.App, args []string) error {
	var shell, alias string
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--shell="):
			shell = strings.TrimPrefix(arg, "--shell=")
		case strings.HasPrefix(arg, "--alias="):
			alias = strings.TrimPrefix(arg, "--alias=")
		case arg == "-h", arg == "--help":
			return uninstallUsage()
		default:
			return fmt.Errorf("completion uninstall: unknown argument %q", arg)
		}
	}
	if alias != "" && alias != "git" {
		return fmt.Errorf("completion uninstall: --alias=%s not supported (only --alias=git)", alias)
	}
	if shell == "" {
		shell = detectShell()
		if shell == "" {
			return errors.New("completion uninstall: cannot detect shell from $SHELL; pass --shell=bash or --shell=zsh")
		}
	}
	switch shell {
	case "bash":
		return uninstallBash(a)
	case "zsh":
		return uninstallZsh(a)
	default:
		return fmt.Errorf("completion uninstall: unsupported shell %q (supported: bash, zsh)", shell)
	}
}

func uninstallUsage() error {
	return fmt.Errorf("usage: gist completion uninstall [--shell=bash|zsh] [--alias=git]")
}

func uninstallBash(a *app.App) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("completion uninstall: %w", err)
	}
	file := filepath.Join(home, ".local", "share", "bash-completion", "completions", "gist")
	if err := removeFile(a, file, "bash completion"); err != nil {
		return err
	}
	return removeRcBlock(a, filepath.Join(home, ".bashrc"))
}

func uninstallZsh(a *app.App) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("completion uninstall: %w", err)
	}
	// _gist could live in any fpath dir we previously chose, so probe the
	// same set of candidates as install does plus all writable HOME-rooted
	// fpath entries we can see.
	dirs := zshCandidateDirs(home)
	removed := 0
	for _, dir := range dirs {
		path := filepath.Join(dir, "_gist")
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("completion uninstall: %w", err)
			}
			fmt.Fprintf(a.Out, "removed zsh completion ← %s\n", path)
			removed++
		}
	}
	if removed == 0 {
		fmt.Fprintf(a.Out, "no zsh completion file found under %s (skipped)\n", home)
	}
	return removeRcBlock(a, zshRcPath(home))
}

// zshCandidateDirs returns dirs to search for a previously-installed _gist
// file: the install default plus any HOME-rooted entries from the live
// $fpath. Order doesn't matter — we remove any matches.
func zshCandidateDirs(home string) []string {
	defaultDir := filepath.Join(home, ".zsh", "completions")
	if z := os.Getenv("ZDOTDIR"); z != "" {
		defaultDir = filepath.Join(z, "completions")
	}
	dirs := []string{defaultDir}
	out, err := exec.Command("zsh", "-c", "print -lr -- $fpath").Output()
	if err != nil {
		return dirs
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, home) || line == defaultDir {
			continue
		}
		dirs = append(dirs, line)
	}
	return dirs
}

func removeFile(a *app.App, path, label string) error {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(a.Out, "no %s at %s (skipped)\n", label, path)
		return nil
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("completion uninstall: %w", err)
	}
	fmt.Fprintf(a.Out, "removed %s ← %s\n", label, path)
	return nil
}

// removeRcBlock strips the gist-managed delimited block from path. Also
// peels off one preceding blank line (the one install adds to separate
// the block from existing content) so repeated install/uninstall cycles
// don't accumulate empty lines.
func removeRcBlock(a *app.App, path string) error {
	existing, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(a.Out, "no rcfile at %s (skipped)\n", path)
		return nil
	}
	if err != nil {
		return fmt.Errorf("completion uninstall: %w", err)
	}
	start, end, ok := findRcBlock(string(existing))
	if !ok {
		fmt.Fprintf(a.Out, "no gist completion block in %s (skipped)\n", path)
		return nil
	}
	cutStart := start
	if cutStart >= 2 && existing[cutStart-1] == '\n' && existing[cutStart-2] == '\n' {
		cutStart--
	}
	result := append(append([]byte{}, existing[:cutStart]...), existing[end:]...)
	if err := os.WriteFile(path, result, 0o644); err != nil {
		return fmt.Errorf("completion uninstall: %w", err)
	}
	fmt.Fprintf(a.Out, "removed gist completion block ← %s\n", path)
	return nil
}

// findRcBlock locates the first delimited block in text. Delimiters must
// each occupy their own line. Returns byte offsets [start, end) covering
// both delimiters and everything between them, including the trailing
// newline after the closing delimiter if present.
func findRcBlock(text string) (start, end int, ok bool) {
	first := -1
	pos := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		if strings.TrimRight(line, "\n") == rcDelim {
			if first < 0 {
				first = pos
			} else {
				return first, pos + len(line), true
			}
		}
		pos += len(line)
	}
	return -1, -1, false
}
