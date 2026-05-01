package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gistBinary is the path to a freshly built gist binary, populated by TestMain.
// Integration tests exec it against tempdir repos so the full main()
// dispatch + collectors + render path is exercised.
var gistBinary string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "gist-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir tmp: %v\n", err)
		os.Exit(2)
	}
	gistBinary = filepath.Join(tmp, "gist")
	build := exec.Command("go", "build", "-o", gistBinary, "github.com/ArturGulik/gist")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build gist: %v\n%s\n", err, out)
		os.Exit(2)
	}
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// makeRepo initializes a fresh git repo in t.TempDir() with a single commit
// on "main" and returns the repo path. The repo has user.name / user.email
// set locally so commits succeed without a global git identity.
func makeRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.email", "test@example.com")
	gitIn(t, repo, "config", "user.name", "Test")
	gitIn(t, repo, "config", "commit.gpgsign", "false")
	writeFile(t, filepath.Join(repo, "README"), "init\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "initial")
	return repo
}

// gitIn runs a git command inside repo, failing the test on non-zero exit.
func gitIn(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitFile creates (or rewrites) a file in repo and commits it.
func commitFile(t *testing.T, repo, name, content, msg string) {
	t.Helper()
	writeFile(t, filepath.Join(repo, name), content)
	gitIn(t, repo, "add", name)
	gitIn(t, repo, "commit", "-q", "-m", msg)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// runGist execs the built gist binary in repo with the given args. Returns
// combined stdout/stderr so error messages from gist surface in test output.
// Color is disabled so assertions can match plain text.
func runGist(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command(gistBinary, args...)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gist %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func TestIntegration_StatusSingleBranch(t *testing.T) {
	repo := makeRepo(t)
	out := runGist(t, repo, "status")
	if !strings.Contains(out, "main") {
		t.Errorf("expected current branch 'main' in output, got:\n%s", out)
	}
}

func TestIntegration_StatusMultipleBranches(t *testing.T) {
	repo := makeRepo(t)
	gitIn(t, repo, "checkout", "-q", "-b", "feature")
	commitFile(t, repo, "feature.txt", "x\n", "add feature")
	gitIn(t, repo, "checkout", "-q", "main")
	gitIn(t, repo, "checkout", "-q", "-b", "another")
	commitFile(t, repo, "another.txt", "y\n", "add another")

	out := runGist(t, repo, "status")
	for _, want := range []string{"main", "feature", "another"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestIntegration_StatusAhead(t *testing.T) {
	repo := makeRepo(t)
	// Create a bare "remote" and push main so feature has a real upstream
	// to be ahead of.
	bare := t.TempDir()
	cmd := exec.Command("git", "init", "-q", "--bare", bare)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init bare: %v\n%s", err, out)
	}
	gitIn(t, repo, "remote", "add", "origin", bare)
	gitIn(t, repo, "push", "-q", "-u", "origin", "main")
	commitFile(t, repo, "extra.txt", "1\n", "extra 1")
	commitFile(t, repo, "extra2.txt", "2\n", "extra 2")

	out := runGist(t, repo, "status")
	if !strings.Contains(out, "↑2") {
		t.Errorf("expected '↑2' indicator in output, got:\n%s", out)
	}
}

func TestIntegration_RebaseInProgress(t *testing.T) {
	repo := makeRepo(t)
	commitFile(t, repo, "shared.txt", "base\n", "base")
	gitIn(t, repo, "checkout", "-q", "-b", "feature")
	commitFile(t, repo, "shared.txt", "feature change\n", "feature change")
	gitIn(t, repo, "checkout", "-q", "main")
	commitFile(t, repo, "shared.txt", "main change\n", "main change")

	// Force a conflicting rebase that halts mid-flight; ignore exit code.
	cmd := exec.Command("git", "rebase", "main", "feature")
	cmd.Dir = repo
	_ = cmd.Run()

	out := runGist(t, repo, "status")
	if !strings.Contains(out, "rebase in progress") {
		t.Errorf("expected 'rebase in progress' banner, got:\n%s", out)
	}

	// Clean up the rebase so t.TempDir cleanup doesn't trip on locked refs.
	abort := exec.Command("git", "rebase", "--abort")
	abort.Dir = repo
	_ = abort.Run()
}

// TestIntegration_UpdateWithFakeGh exercises the gh-backed PR cache by
// dropping a fake `gh` script in PATH that prints canned JSON. After
// `gist update`, `gist status` should show the PR number on the matching
// branch even though no real GitHub API was contacted.
func TestIntegration_UpdateWithFakeGh(t *testing.T) {
	repo := makeRepo(t)
	gitIn(t, repo, "remote", "add", "origin", "https://github.com/test/test.git")
	gitIn(t, repo, "checkout", "-q", "-b", "feature")
	commitFile(t, repo, "feature.txt", "x\n", "add feature")

	// Fake gh: ignore arguments, print one PR matching the feature branch.
	fakeBin := t.TempDir()
	fakeGh := filepath.Join(fakeBin, "gh")
	if err := os.WriteFile(fakeGh, []byte(`#!/bin/sh
cat <<'EOF'
[{"number":87,"state":"OPEN","headRefName":"feature","isDraft":false}]
EOF
`), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	// Run gist update with PATH prepended so the fake gh is discovered first.
	cmd := exec.Command(gistBinary, "update")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(),
		"NO_COLOR=1",
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gist update: %v\n%s", err, out)
	}

	// Cache file should now exist.
	cachePath := filepath.Join(repo, ".git", "gist", "prs.json")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected PR cache at %s: %v", cachePath, err)
	}

	out := runGist(t, repo, "status")
	if !strings.Contains(out, "#87") {
		t.Errorf("expected '#87' PR indicator on feature branch, got:\n%s", out)
	}
}
