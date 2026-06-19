# gist — Git Instant State Tool

[![CI](https://github.com/ArturGulik/gist/actions/workflows/ci.yml/badge.svg)](https://github.com/ArturGulik/gist/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ArturGulik/gist.svg)](https://pkg.go.dev/github.com/ArturGulik/gist)
[![Version](https://img.shields.io/github/v/tag/ArturGulik/gist?sort=semver&label=version&color=00ADD8)](CHANGELOG.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/ArturGulik/gist)](https://goreportcard.com/report/github.com/ArturGulik/gist)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

A drop-in `git` superset. Better defaults, smarter wrappers, transparent passthrough — `alias git=gist` and forget about it.

```
  main
  feature/payments-checkout  #87 ↑2
  hotfix/login
  i18n/translations
  scratchpad                 ◦
* feature/auth-flow          ~92 ↑1↓1
  ⇣ remote-feature
  1 stash
───────────────────
?? newfile.txt
```

## Quick start

```sh
go install github.com/ArturGulik/gist@latest
# ensure $(go env GOPATH)/bin (usually ~/go/bin) is on your $PATH
gist completion install --alias=git
# source ~/.bashrc  # or ~/.zshrc
```

## Install

Linux & macOS, amd64/arm64.

**Prebuilt binary:** grab the archive for your platform from the [latest release](https://github.com/ArturGulik/gist/releases/latest), extract `gist`, drop it on your `PATH`.

**Go ≥ 1.23:**

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

### Shell completion

`make install` sets this up automatically. For other install methods, run:

```sh
gist completion install                  # detects $SHELL, drops the file
gist completion install --alias=git      # also wraps `git` so `git up<TAB>` finds `update`
gist completion uninstall                # removes the file and any rc block
```

The base completion lands at an auto-loaded path — no rcfile edit needed.
`--alias=git` additionally appends a delimited block to your
`.bashrc`/`.zshrc`; re-running the install command updates that block in
place, and `uninstall` strips it back out.

If you'd rather wire it up by hand, `gist completion bash` and `gist
completion zsh` print the script to stdout (also accept `--alias=git`).

## Usage

```text
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
                     `gist completion install [--alias=git]` drops
                     the file in the right place; `uninstall` undoes it.
gist legend          Explain the symbols used by status
gist version         Print version
gist help            Print this help
gist <cmd> …         Any other command is passed through to git
```

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
