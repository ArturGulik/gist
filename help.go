package main

import (
	"fmt"

	"github.com/ArturGulik/gist/internal/app"
)

func printHelp(a *app.App) {
	fmt.Fprint(a.Out, `gist — Git Instant State Tool

Usage:
  gist [status]        Show branches and working-tree state (default)
  gist branch          Detailed info for the current branch
  gist branch …        Pass through to git branch
  gist remote          Show remotes with clickable web URLs
  gist remote …        Pass through to git remote
  gist fetch …         git fetch, then refresh PR/MR cache
  gist switch …        git switch, then print a status line for the new branch
  gist update          Refresh PR/MR cache (GitHub via gh, GitLab via glab)
  gist config          Print the fully-documented default config to stdout
  gist completion …    Print, install, or uninstall shell completion.
                       `+"`gist completion install [--alias=git]`"+` drops
                       the file in the right place; `+"`uninstall`"+` undoes it.
  gist legend          Explain the symbols used by status
  gist version         Print version
  gist help            Print this help
  gist <cmd> …         Any other command is passed through to git

Env:
  NO_COLOR        Disable color output
  GIST_COLOR      "always" or "never" to override TTY detection

Config:
  Global: $XDG_CONFIG_HOME/gist/config (or ~/.config/gist/config) —
          auto-generated on first run with all defaults + comments.
  Repo:   <git-dir>/gist/config (overrides global)

  Run `+"`gist config`"+` to print the fully-documented defaults; redirect
  to the path above to reset. Format is git-config-style INI. Keys:
    [status]
        show-subject = false   # append commit subject column
        show-hash = false      # append short hash column
        show-date = false      # append relative committer date column
        hyperlink-prs = true   # OSC-8 hyperlink PR numbers to forge URL
    [sections]
        stash = true
        status-footer = true
        in-progress-banner = true
    [symbol]
        ahead = "↑"   behind = "↓"   no-upstream = "◦"
        remote-only = "⇣"   in-progress = "⚠"
        pr-open = "#"   pr-draft = "~"   pr-merged = "✓"   pr-closed = "×"
    [color]
        # values: any of bold dim italic strike + a foreground name
        # (red green yellow blue magenta cyan), space-separated.
        branch-current = "bold green"     branch-default = "bold"
        branch-gone = "strike dim"        branch-remote-only = "dim"
        branch-pr-merged = "dim"
        pr-open = "cyan"                  pr-draft = "dim cyan"
        pr-merged = "dim green"           pr-closed = "red"
        sync-ahead = "yellow"             sync-behind = "red"
        sync-no-upstream = "dim"
        in-progress = "bold yellow"
        divider = "dim"                   status-meta = "dim"
`)
}

func printLegend(a *app.App) {
	sy := a.Cfg.Symbols
	fmt.Fprintf(a.Out, `Symbols appear before the branch name. Multiple can combine,
separated by a space (e.g. "%s81 ↑2").

PR/MR state (from `+"`gist update`"+` via gh or glab):
  %sN       open pull request
  %sN       open draft pull request
  %sN       merged pull request
  %sN       closed / rejected pull request

Sync state (local vs. upstream):
  %s %s    ahead / behind upstream
  %s        no upstream (local only — never pushed)
  (strike) branch name is struck through when its upstream is gone
           (remote branch was deleted — cleanup candidate). This is
           distinct from "never pushed" (%s), which git tracks separately.
  %s        remote-only (exists on origin, not checked out locally)
  (none)   in sync with upstream

Other:
  %s        rebase / merge / cherry-pick / revert / bisect in progress

Symbols and colors are configurable — see `+"`gist help`"+` for the config
file location and keys. Rows are sorted default branch first, then local
branches alphabetically, then remote-only branches alphabetically.
`,
		sy.PROpen,
		sy.PROpen, sy.PRDraft, sy.PRMerged, sy.PRClosed,
		sy.Ahead, sy.Behind,
		sy.NoUpstream, sy.NoUpstream,
		sy.RemoteOnly,
		sy.InProgress,
	)
}
