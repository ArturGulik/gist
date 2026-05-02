package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempHome points HOME (and ZDOTDIR) at a fresh tmpdir for the duration
// of one test. SHELL is also set so detectShell() works deterministically.
func withTempHome(t *testing.T, shell string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("ZDOTDIR", "")
	t.Setenv("SHELL", "/usr/bin/"+shell)
	return dir
}

func TestInstall_BashWritesFile(t *testing.T) {
	home := withTempHome(t, "bash")
	a, _, _ := newApp()
	if err := Install(a, nil); err != nil {
		t.Fatalf("Install: %v", err)
	}
	dest := filepath.Join(home, ".local", "share", "bash-completion", "completions", "gist")
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read %s: %v", dest, err)
	}
	if !strings.Contains(string(body), "complete -F _gist gist") {
		t.Errorf("dropped file is not bash completion")
	}
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Errorf(".bashrc was touched without --alias=git")
	}
}

func TestInstall_BashAliasGitAppendsRc(t *testing.T) {
	home := withTempHome(t, "bash")
	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, []byte("# preexisting\nexport FOO=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a, _, _ := newApp()
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "# preexisting") {
		t.Errorf("preexisting content lost")
	}
	if !strings.Contains(string(got), rcDelim) {
		t.Errorf("rcfile missing delimiter")
	}
	if !strings.Contains(string(got), "_completion_loader git") {
		t.Errorf("rcfile missing eager git loader")
	}
	if !strings.Contains(string(got), "gist completion bash --alias=git") {
		t.Errorf("rcfile missing source line")
	}
	if strings.Count(string(got), rcDelim) != 2 {
		t.Errorf("expected exactly 2 delimiter lines, got %d", strings.Count(string(got), rcDelim))
	}
}

func TestInstall_BashAliasGitIdempotent(t *testing.T) {
	withTempHome(t, "bash")

	a, _, _ := newApp()
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatal(err)
	}
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatal(err)
	}
	rc := filepath.Join(os.Getenv("HOME"), ".bashrc")
	got, _ := os.ReadFile(rc)
	if c := strings.Count(string(got), rcDelim); c != 2 {
		t.Errorf("expected 2 delimiter lines after re-install, got %d", c)
	}
}

func TestInstall_ZshWritesFile(t *testing.T) {
	home := withTempHome(t, "zsh")
	a, _, _ := newApp()
	if err := Install(a, nil); err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Either default dir or an fpath dir under HOME — locate _gist by searching.
	var found string
	_ = filepath.Walk(home, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Base(path) == "_gist" {
			found = path
		}
		return nil
	})
	if found == "" {
		t.Fatalf("did not find _gist under %s", home)
	}
	body, _ := os.ReadFile(found)
	if !strings.HasPrefix(string(body), "#compdef gist") {
		t.Errorf("zsh _gist file missing #compdef tag")
	}
}

func TestInstall_ZshAliasGitAppendsRc(t *testing.T) {
	home := withTempHome(t, "zsh")
	a, _, _ := newApp()
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	rc := filepath.Join(home, ".zshrc")
	got, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "gist completion zsh --alias=git") {
		t.Errorf("rcfile missing zsh source line")
	}
	if strings.Count(string(got), rcDelim) != 2 {
		t.Errorf("expected 2 delimiter lines, got %d", strings.Count(string(got), rcDelim))
	}
}

func TestInstall_BadAlias(t *testing.T) {
	withTempHome(t, "bash")
	a, _, _ := newApp()
	err := Install(a, []string{"--alias=fish"})
	if err == nil || !strings.Contains(err.Error(), "fish") {
		t.Errorf("expected error mentioning fish, got %v", err)
	}
}

func TestInstall_BadShell(t *testing.T) {
	withTempHome(t, "fish")
	a, _, _ := newApp()
	err := Install(a, []string{"--shell=fish"})
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected unsupported-shell error, got %v", err)
	}
}

func TestInstall_NoShellNoEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "")
	a, _, _ := newApp()
	err := Install(a, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot detect shell") {
		t.Errorf("expected detect-shell error, got %v", err)
	}
}

func TestFindRcBlock(t *testing.T) {
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"empty", "", false},
		{"none", "export FOO=1\n", false},
		{"only-one", rcDelim + "\nfoo\n", false},
		{"valid", "before\n" + rcDelim + "\nfoo\n" + rcDelim + "\nafter\n", true},
		{"no-newline-after", "x\n" + rcDelim + "\nfoo\n" + rcDelim, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, ok := findRcBlock(tc.in)
			if ok != tc.ok {
				t.Errorf("findRcBlock(%q): want ok=%v, got %v", tc.in, tc.ok, ok)
			}
		})
	}
}

func TestUninstall_BashRemovesFileAndBlock(t *testing.T) {
	home := withTempHome(t, "bash")
	a, _, _ := newApp()

	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatal(err)
	}
	rc := filepath.Join(home, ".bashrc")
	file := filepath.Join(home, ".local", "share", "bash-completion", "completions", "gist")
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("setup: completion file missing: %v", err)
	}

	if err := Uninstall(a, nil); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("expected completion file removed, stat err = %v", err)
	}
	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), rcDelim) {
		t.Errorf("rcfile still contains delimiter: %q", got)
	}
}

func TestUninstall_PreservesSurroundingRcContent(t *testing.T) {
	home := withTempHome(t, "bash")
	a, _, _ := newApp()
	rc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(rc, []byte("alpha=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatal(err)
	}
	// Append more content after the block to make sure we don't truncate it.
	more := []byte("\nomega=9\n")
	f, _ := os.OpenFile(rc, os.O_APPEND|os.O_WRONLY, 0o644)
	f.Write(more)
	f.Close()

	if err := Uninstall(a, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(rc)
	want := "alpha=1\n\nomega=9\n"
	if string(got) != want {
		t.Errorf("rcfile after uninstall:\n got: %q\nwant: %q", got, want)
	}
}

func TestUninstall_Idempotent(t *testing.T) {
	withTempHome(t, "bash")
	a, _, _ := newApp()
	if err := Uninstall(a, nil); err != nil {
		t.Fatalf("first uninstall on clean home: %v", err)
	}
	if err := Uninstall(a, nil); err != nil {
		t.Fatalf("second uninstall: %v", err)
	}
}

func TestUninstall_AliasGitFlagIsAccepted(t *testing.T) {
	withTempHome(t, "bash")
	a, _, _ := newApp()
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(a, []string{"--alias=git"}); err != nil {
		t.Fatalf("uninstall --alias=git: %v", err)
	}
	rc := filepath.Join(os.Getenv("HOME"), ".bashrc")
	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), rcDelim) {
		t.Errorf("--alias=git uninstall did not remove block: %q", got)
	}
}

func TestUninstall_Zsh(t *testing.T) {
	home := withTempHome(t, "zsh")
	a, _, _ := newApp()
	if err := Install(a, []string{"--alias=git"}); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(a, nil); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	// _gist should be gone from the install location.
	dest := filepath.Join(home, ".zsh", "completions", "_gist")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("zsh _gist still present: %v", err)
	}
	rc := filepath.Join(home, ".zshrc")
	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), rcDelim) {
		t.Errorf("zshrc still contains delimiter: %q", got)
	}
}

func TestUninstall_BadShell(t *testing.T) {
	withTempHome(t, "fish")
	a, _, _ := newApp()
	err := Uninstall(a, []string{"--shell=fish"})
	if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("expected unsupported-shell error, got %v", err)
	}
}

func TestUpsertRcBlock_ReplacesInPlace(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, "rc")
	os.WriteFile(rc, []byte("alpha\n"+rcDelim+"\nold-content\n"+rcDelim+"\nomega\n"), 0o644)

	action, err := upsertRcBlock(rc, []string{"new-content"})
	if err != nil {
		t.Fatal(err)
	}
	if action != "updated" {
		t.Errorf("action = %q, want updated", action)
	}
	got, _ := os.ReadFile(rc)
	if strings.Contains(string(got), "old-content") {
		t.Errorf("old content not removed: %q", got)
	}
	if !strings.Contains(string(got), "new-content") {
		t.Errorf("new content not present: %q", got)
	}
	if !strings.HasPrefix(string(got), "alpha\n") || !strings.HasSuffix(string(got), "omega\n") {
		t.Errorf("surrounding content mangled: %q", got)
	}
}
