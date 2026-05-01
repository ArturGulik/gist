package collect

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ArturGulik/gist/internal/git"
	"github.com/ArturGulik/gist/internal/model"
	"github.com/ArturGulik/gist/internal/update"
)

// RepoState gathers everything needed to render the status view.
// All operations here must be local (no network), cheap, and tolerant of
// partial failure — missing pieces degrade gracefully rather than erroring.
// The color flag controls whether `git status -s` output preserves color codes.
func RepoState(color bool) (*model.RepoState, error) {
	s := &model.RepoState{}

	s.DefaultBranch = DetectDefaultBranch()
	s.CurrentBranch = git.CurrentBranch()
	s.DetachedHead = s.CurrentBranch == ""

	branches, err := Branches(s.DefaultBranch, s.CurrentBranch)
	if err != nil {
		return nil, err
	}
	s.Branches = branches

	s.Branches = append(s.Branches, RemoteOnly(s.Branches)...)
	ApplyPRCache(s.Branches)

	s.StashCount = StashCount()
	s.InProgress = DetectInProgress()

	colorArg := "color.status=never"
	if color {
		colorArg = "color.status=always"
	}
	if raw, err := git.RunRaw("-c", colorArg, "status", "-s"); err == nil {
		s.StatusRaw = raw
	}

	return s, nil
}

// DetectDefaultBranch resolves the repo's default branch by consulting
// origin/HEAD first (set automatically on clone, or by `git remote set-head`),
// then falling back to common local branch names.
func DetectDefaultBranch() string {
	if s, err := git.Run("symbolic-ref", "--short", "--quiet", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(s, "origin/")
	}
	for _, name := range []string{"main", "master", "develop", "trunk"} {
		if _, err := git.Run("show-ref", "--verify", "--quiet", "refs/heads/"+name); err == nil {
			return name
		}
	}
	return ""
}

// Branches runs a single for-each-ref and parses all local branches.
// Using NUL as a field separator avoids any conflict with characters that
// might appear in commit subjects. The 6th field (relative committer date)
// is always collected; the renderer decides whether to display it.
func Branches(defaultBranch, currentBranch string) ([]model.Branch, error) {
	format := strings.Join([]string{
		"%(refname:short)",
		"%(objectname:short)",
		"%(upstream:short)",
		"%(upstream:track)",
		"%(contents:subject)",
		"%(committerdate:relative)",
	}, "%00")

	out, err := git.Run("for-each-ref", "refs/heads/", "--format="+format)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var branches []model.Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 6)
		if len(parts) < 6 {
			continue
		}
		b := model.Branch{
			Name:         parts[0],
			Hash:         parts[1],
			Upstream:     parts[2],
			Subject:      parts[4],
			LastActivity: parts[5],
			IsCurrent:    parts[0] == currentBranch,
			IsDefault:    parts[0] == defaultBranch,
		}
		b.Ahead, b.Behind, b.Gone = ParseTrack(parts[3])
		branches = append(branches, b)
	}
	return branches, nil
}

// ParseTrack decodes `%(upstream:track)` output. Examples:
//
//	""                          -> in sync or no upstream (distinguish via Upstream)
//	"[gone]"                    -> upstream deleted
//	"[ahead 2]"                 -> ahead only
//	"[behind 1]"                -> behind only
//	"[ahead 2, behind 1]"       -> diverged
func ParseTrack(s string) (ahead, behind int, gone bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "gone" {
		return 0, 0, true
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "ahead "):
			ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
		case strings.HasPrefix(part, "behind "):
			behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
		}
	}
	return
}

// ApplyPRCache fills PRNumber/PRState from the on-disk cache written by
// `gist update`. Missing cache is silently ignored — `status` must remain a
// local, offline operation.
func ApplyPRCache(branches []model.Branch) {
	cache, err := update.LoadCache()
	if err != nil || cache == nil {
		return
	}
	for i := range branches {
		if info, ok := cache[branches[i].Name]; ok {
			branches[i].PRNumber = info.Number
			branches[i].PRState = info.State
			branches[i].PRIsDraft = info.IsDraft
		}
	}
}

// RemoteOnly lists branches that exist on any remote but have no local
// counterpart, returning them as Branch rows so they render in the main
// list. This reads the already-cached refs/remotes/, so no network I/O
// and no credential prompt — the data is as fresh as the last `git fetch`.
//
// If the same branch name appears under multiple remotes, the first one
// encountered wins (order: git's for-each-ref traversal).
func RemoteOnly(locals []model.Branch) []model.Branch {
	format := strings.Join([]string{
		"%(refname)",
		"%(objectname:short)",
		"%(contents:subject)",
		"%(committerdate:relative)",
	}, "%00")
	out, err := git.Run("for-each-ref", "refs/remotes/", "--format="+format)
	if err != nil {
		return nil
	}
	localSet := map[string]bool{}
	for _, b := range locals {
		localSet[b.Name] = true
	}
	seen := map[string]bool{}
	var remoteOnly []model.Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) < 4 {
			continue
		}
		// parts[0] is e.g. "refs/remotes/upstream/feature/x". Strip the
		// "refs/remotes/<remote>/" prefix to get the bare branch name.
		full := strings.TrimPrefix(parts[0], "refs/remotes/")
		slash := strings.IndexByte(full, '/')
		if slash < 0 {
			continue
		}
		name := full[slash+1:]
		if name == "" || name == "HEAD" {
			continue
		}
		if localSet[name] || seen[name] {
			continue
		}
		seen[name] = true
		remoteOnly = append(remoteOnly, model.Branch{
			Name:         name,
			Hash:         parts[1],
			Subject:      parts[2],
			LastActivity: parts[3],
			IsRemoteOnly: true,
		})
	}
	return remoteOnly
}

func StashCount() int {
	out, err := git.Run("stash", "list")
	if err != nil || out == "" {
		return 0
	}
	return strings.Count(out, "\n") + 1
}

// DetectInProgress reports any mid-flight git operation by checking for
// well-known marker files in the git dir. These are cheap stat calls.
func DetectInProgress() string {
	gitDir, err := git.Run("rev-parse", "--git-dir")
	if err != nil {
		return ""
	}
	exists := func(p string) bool {
		_, err := os.Stat(filepath.Join(gitDir, p))
		return err == nil
	}
	switch {
	case exists("rebase-merge") || exists("rebase-apply"):
		return "rebase"
	case exists("MERGE_HEAD"):
		return "merge"
	case exists("CHERRY_PICK_HEAD"):
		return "cherry-pick"
	case exists("REVERT_HEAD"):
		return "revert"
	case exists("BISECT_LOG"):
		return "bisect"
	}
	return ""
}

// CommitInfo is one row of `gist branch` ahead-of-default commits.
type CommitInfo struct {
	Hash    string
	Date    string
	Author  string
	Subject string
}

// CommitsAhead returns commits on `branch` that are not on `base`, oldest
// first (git log default order).
func CommitsAhead(base, branch string) ([]CommitInfo, error) {
	out, err := git.Run("log", "--no-merges",
		"--format=%h%x00%as%x00%an%x00%s",
		base+".."+branch)
	if err != nil || out == "" {
		return nil, err
	}
	var commits []CommitInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		p := strings.SplitN(line, "\x00", 4)
		if len(p) < 4 {
			continue
		}
		commits = append(commits, CommitInfo{Hash: p[0], Date: p[1], Author: p[2], Subject: p[3]})
	}
	return commits, nil
}

// FilesChanged returns the three-dot diff (vs. merge base) name-status lines.
func FilesChanged(base, branch string) ([]string, error) {
	out, err := git.Run("diff", "--name-status", base+"..."+branch)
	if err != nil || out == "" {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
