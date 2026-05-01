# gist — Git Instant State Tool

[![CI](https://github.com/ArturGulik/gist/actions/workflows/ci.yml/badge.svg)](https://github.com/ArturGulik/gist/actions/workflows/ci.yml)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A drop-in `git` superset. Better defaults, smarter wrappers, transparent passthrough — `alias git=gist` and forget about it.

```
  main                         e956f1b             Merge hotfix/quick
  contracts-improvement/artur  13a578d  ↑2 #87     Even more contract work
  hotfix/quick                 7150cc9             Quick hotfix
  i18n/jacek                   560787b             Fixing hardcoded pages
  local-experiment             e11597a  ◦          Trying out a thing
* sso/artur                    594d6e9  ↑1↓1 ~92   Improve email resend
  ⇣ remote-feature             a3b2c1d             Feature only on remote
  1 stash
───────────────────
?? newfile.txt
```

- **Every branch, one screen.** Bare `gist` is a multi-branch status — ahead/behind, stashes, working tree, in-progress merges — sub-100ms.
- **PR/MR state inline.** Number + status from GitHub & GitLab, cached locally. No live network calls on the hot path.
- **Wraps git, doesn't replace it.** Smarter `branch` / `switch` / `fetch` / `remote`; everything else passes straight through. Runs, prints, exits — no daemon, no TUI.

## Install

Linux & macOS, amd64/arm64.

**Prebuilt binary:** grab the archive for your platform from the [latest release](https://github.com/ArturGulik/gist/releases/latest), extract `gist`, drop it on your `PATH`.

**Go ≥ 1.22:**

```sh
go install github.com/ArturGulik/gist@latest
```

**From source:**

```sh
git clone https://github.com/ArturGulik/gist
cd gist
make install        # → ~/.local/bin/gist with version metadata baked in
```

### Use it as `git`

`gist` is a superset — anything it doesn't recognize is forwarded to `git`. Alias it and you'll never type `git` again:

```sh
alias git=gist
```

Or keep `git` and add a single `git st` entry point:

```sh
git config --global alias.st '!gist'
```

## Usage

```
gist [status]   all branches + working-tree at a glance (default)
gist branch …   detailed view for one branch
gist switch …   git switch + status for the new branch
gist fetch …    git fetch + refresh PR/MR cache
gist update     refresh PR/MR cache only (uses gh / glab)
gist remote     remotes with clickable web URLs
gist legend     explain the symbols
gist config     print the fully-commented default config
gist help       full help with every config key + default
gist <cmd> …    anything else → forwarded to git
```

Aliases: `s` `b` `sw` `u` `l` `r`.

## PR / MR state

After `gist fetch` (or any `gist update`), branches show inline PR numbers:

| Symbol | Meaning      |
| ------ | ------------ |
| `#N`   | open         |
| `~N`   | open draft   |
| `✓N`   | merged       |
| `×N`   | closed       |

Backed by [`gh`](https://cli.github.com/) (GitHub) and [`glab`](https://gitlab.com/gitlab-org/cli) (GitLab) — soft no-op if neither is installed or the remote isn't recognized, so `gist` works everywhere.

## Configuration

A fully-commented config is auto-generated on first run. Edit:

- Global: `$XDG_CONFIG_HOME/gist/config` (or `~/.config/gist/config`)
- Per-repo: `<git-dir>/gist/config` *(overrides global)*

You can change symbols (incl. nerd-font alternatives), colors, optional columns (subject, hash, date), section visibility, and PR hyperlinks. `NO_COLOR` and `GIST_COLOR=always|never` are honored.

```sh
gist config         # print the fully-commented defaults
gist help           # every key + default + valid values
```

## License

[GPL-3.0](LICENSE).
