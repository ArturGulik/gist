package model

// Branch is the per-branch data the renderer consumes.
// New fields can be added as collectors grow (e.g. Author, PRState).
type Branch struct {
	Name         string
	Hash         string // short hash
	Upstream     string // empty if not tracking anything
	Ahead        int
	Behind       int
	Gone         bool // upstream was configured but the remote ref is gone
	IsCurrent    bool
	IsDefault    bool
	IsRemoteOnly bool // exists on origin only, no local ref
	Subject      string
	LastActivity string // committer date, relative (e.g. "2 days ago"); empty when not requested
	PRNumber     int    // 0 if no associated PR on the remote
	PRState      string // "open" | "merged" | "closed" | "" (no PR)
	PRIsDraft    bool   // true if the PR is a draft
}

// RepoState is the assembled input to the renderer. Collectors populate it;
// the renderer has no knowledge of git.
type RepoState struct {
	DefaultBranch string // "main", "master", "develop", or "" if unknown
	CurrentBranch string // empty if HEAD is detached
	DetachedHead  bool
	Branches      []Branch // includes both local and remote-only branches
	StashCount    int
	InProgress    string // "rebase" | "merge" | "cherry-pick" | "revert" | "bisect" | ""
	StatusRaw     []byte // output of `git status -s`, color codes preserved when appropriate
}
