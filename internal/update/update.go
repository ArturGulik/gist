package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/git"
)

// CacheEntry is the on-disk shape per branch.
type CacheEntry struct {
	Number  int    `json:"number"`
	State   string `json:"state"` // "open" | "merged" | "closed"
	IsDraft bool   `json:"isDraft,omitempty"`
}

// forge abstracts a hosting provider that exposes pull/merge requests through
// a local CLI. New providers (Bitbucket, Gitea, …) implement this interface.
// Methods stay package-private; only update's own functions consume them.
type forge interface {
	cli() string                 // executable name on PATH
	label() string               // singular noun for messages: "PR" / "MR"
	prPathFmt() string           // appended to the project web URL: "/pull/%d", "/-/merge_requests/%d"
	list() ([]forgeEntry, error) // entries keyed by source/head branch
}

// forgeEntry is the cross-forge intermediate before dedup. State and Number
// must already be normalized — RunUpdate just dedupes by Number.
type forgeEntry struct {
	branch string
	entry  CacheEntry
}

// ghPR matches the JSON fields we ask for from `gh pr list`.
type ghPR struct {
	Number      int    `json:"number"`
	State       string `json:"state"` // OPEN | MERGED | CLOSED
	HeadRefName string `json:"headRefName"`
	IsDraft     bool   `json:"isDraft"`
}

// glabMR matches the JSON fields returned by `glab mr list --output json`.
type glabMR struct {
	IID          int    `json:"iid"`
	State        string `json:"state"` // "opened" | "merged" | "closed"
	SourceBranch string `json:"source_branch"`
	Draft        bool   `json:"draft"`
	WIP          bool   `json:"work_in_progress"` // pre-14.2 alias for Draft
}

// RunFetch runs git fetch with any extra args, then refreshes the PR/MR cache.
// Fetch output is passed through directly so progress and remote messages
// appear normally. If fetch fails, returns a *git.ExitError carrying git's code.
func RunFetch(a *app.App, args []string) error {
	cmd := exec.Command("git", append([]string{"fetch"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = a.Out
	cmd.Stderr = a.Err
	if err := cmd.Run(); err != nil {
		return &git.ExitError{Code: git.ExitCodeFrom(cmd, 1)}
	}
	return RunUpdate(a, nil)
}

// RunUpdate refreshes the PR/MR cache from the detected remote forge.
// It is a no-op with a soft warning when the required CLI is unavailable or
// the remote is not a supported forge — gist must remain useful outside those ecosystems.
func RunUpdate(a *app.App, _ []string) error {
	if !git.InWorkTree() {
		fmt.Fprintln(a.Out, "not a git repository")
		return nil
	}

	f, _ := detectForge()
	if f == nil {
		fmt.Fprintln(a.Err, "gist: remote is not GitHub or GitLab; skipping PR/MR refresh")
		return nil
	}
	if _, err := exec.LookPath(f.cli()); err != nil {
		fmt.Fprintf(a.Err, "gist: %s CLI not found; skipping %s refresh\n", f.cli(), f.label())
		return nil
	}

	entries, err := f.list()
	if err != nil {
		fmt.Fprintf(a.Err, "gist: %v\n", err)
		return nil
	}

	cache := make(map[string]CacheEntry, len(entries))
	for _, e := range entries {
		// Highest number wins on branch reuse — the most recently created
		// PR/MR on a head ref is always the relevant one.
		if existing, ok := cache[e.branch]; ok && existing.Number >= e.entry.Number {
			continue
		}
		cache[e.branch] = e.entry
	}
	fmt.Fprintf(a.Out, "gist: cached %d %ss\n", len(cache), f.label())
	return WriteCache(cache)
}

// detectForge inspects remote URLs to identify the hosting forge, returning
// the matched forge and the URL it was matched from. origin is checked first;
// other remotes (upstream, github, etc.) are checked in the order git lists
// them so users who name their forge remote anything else still get PR/MR
// refresh and web-URL resolution.
func detectForge() (forge, string) {
	out, err := git.Run("remote")
	if err != nil {
		return nil, ""
	}
	names := strings.Split(strings.TrimSpace(out), "\n")
	ordered := make([]string, 0, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n == "origin" {
			ordered = append([]string{n}, ordered...)
		} else if n != "" {
			ordered = append(ordered, n)
		}
	}
	for _, name := range ordered {
		url, err := git.Run("remote", "get-url", name)
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(url, "github.com"):
			return githubForge{}, url
		case strings.Contains(url, "gitlab"):
			return gitlabForge{}, url
		}
	}
	return nil, ""
}

type githubForge struct{}

func (githubForge) cli() string       { return "gh" }
func (githubForge) label() string     { return "PR" }
func (githubForge) prPathFmt() string { return "/pull/%d" }
func (githubForge) list() ([]forgeEntry, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--state", "all",
		"--json", "number,state,headRefName,isDraft",
		"--limit", "200")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh pr list failed: %s", msg)
	}
	var list []ghPR
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return nil, fmt.Errorf("parse gh output: %w", err)
	}
	out := make([]forgeEntry, 0, len(list))
	for _, e := range list {
		out = append(out, forgeEntry{
			branch: e.HeadRefName,
			entry:  CacheEntry{Number: e.Number, State: strings.ToLower(e.State), IsDraft: e.IsDraft},
		})
	}
	return out, nil
}

type gitlabForge struct{}

func (gitlabForge) cli() string       { return "glab" }
func (gitlabForge) label() string     { return "MR" }
func (gitlabForge) prPathFmt() string { return "/-/merge_requests/%d" }
func (gitlabForge) list() ([]forgeEntry, error) {
	cmd := exec.Command("glab", "mr", "list",
		"--all",
		"--output", "json",
		"--per-page", "100")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("glab mr list failed: %s", msg)
	}
	var list []glabMR
	if err := json.Unmarshal(stdout.Bytes(), &list); err != nil {
		return nil, fmt.Errorf("parse glab output: %w", err)
	}
	out := make([]forgeEntry, 0, len(list))
	for _, e := range list {
		out = append(out, forgeEntry{
			branch: e.SourceBranch,
			entry:  CacheEntry{Number: e.IID, State: normalizeGitLabState(e.State), IsDraft: e.Draft || e.WIP},
		})
	}
	return out, nil
}

// normalizeGitLabState maps GitLab MR states to the internal format.
// GitLab uses "opened" where we use "open"; other states match.
func normalizeGitLabState(s string) string {
	if s == "opened" {
		return "open"
	}
	return s
}

func WriteCache(cache map[string]CacheEntry) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// cachePath resolves the cache file path inside the repo's common git dir
// so all worktrees share one cache.
func cachePath() (string, error) {
	d, err := git.Run("rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(d) {
		top, err2 := git.Run("rev-parse", "--show-toplevel")
		if err2 == nil {
			d = filepath.Join(top, d)
		}
	}
	return filepath.Join(d, "gist", "prs.json"), nil
}

func LoadCache() (map[string]CacheEntry, error) {
	path, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cache map[string]CacheEntry
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return cache, nil
}

// prURLCache memoizes PR-web-URL resolution per process: first call to
// PRWebURL shells to git for the origin URL and detects the forge; later
// calls reuse the resolved base + path format.
var prURLCache struct {
	resolved bool
	base     string
	pathFmt  string
}

// PRWebURL builds the forge web URL for a PR/MR number. Returns "" when the
// origin is not a known forge or its base URL can't be derived.
func PRWebURL(n int) string {
	if !prURLCache.resolved {
		prURLCache.resolved = true
		f, raw := detectForge()
		if f == nil {
			return ""
		}
		prURLCache.base = git.WebURL(raw)
		prURLCache.pathFmt = f.prPathFmt()
	}
	if prURLCache.base == "" || prURLCache.pathFmt == "" {
		return ""
	}
	return prURLCache.base + fmt.Sprintf(prURLCache.pathFmt, n)
}
