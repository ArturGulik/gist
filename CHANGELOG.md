# Changelog

All notable changes to **gist** are documented here. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.2] - 2026-05-02

### Fixed
- `gist version` now prints `gist <version>` cleanly when commit/date metadata is missing — e.g. `go install …@v1.0.2` builds, where the proxy-served source zip has no `.git` for the Go toolchain to read VCS metadata from. Goreleaser archives and `make install` builds keep the full `gist <version> (<sha>, <date>)` format.

## [1.0.1] - 2026-05-02

First effective public release.

### Fixed
- `gist version` printed `dev (none, unknown)` for binaries built via `go install …@version` because the `-ldflags` injection only happens during `make install` / goreleaser builds. `main.go` now falls back to `runtime/debug.ReadBuildInfo()` so `go install …@v1.0.1` reports the correct module version + VCS revision + commit time, with a `+dirty` suffix when the working tree was modified.
- README example was sanitized — internal branch names removed.

### Removed
- v1.0.0 retracted (`retract` directive in `go.mod`). Use v1.0.1 instead.

## [1.0.0] - 2026-05-02 [RETRACTED]

> **Retracted on 2026-05-02.** The published README contained internal branch names. Use **v1.0.1** instead. The cached source zip on `proxy.golang.org` is immutable and remains accessible only via an explicit `@v1.0.0` install.

First public release. `gist` is a non-interactive multi-branch git status — `git status`, but for every branch at once, sub-100ms, with PR/MR state inline.

### Added
- **Multi-branch status view** (`gist`, `gist status`) — one screen per repo: ahead/behind, stashes, working tree, and any in-progress merge/rebase/cherry-pick/revert/bisect.
- **Inline PR/MR indicators** — `#N` open / `~N` draft / `✓N` merged / `×N` closed, sourced from a local cache populated by `gist update` (GitHub via `gh`, GitLab via `glab`). OSC-8 hyperlinks on supporting terminals.
- **Per-branch detail view** (`gist branch [<br>]`) — upstream sync, MR/PR state, commits ahead of default, files changed.
- **Switch + status combo** (`gist switch …`) — wraps `git switch` and replaces "Switched to branch …" chatter with a single status row.
- **Remote view** (`gist remote`) — fetch/push URLs plus a clickable web URL.
- **Fetch passthrough** (`gist fetch …`) — `git fetch` then refresh PR/MR cache.
- **Configuration system** — global (`$XDG_CONFIG_HOME/gist/config`) + per-repo (`<git-dir>/gist/config`) git-config-style INI; auto-bootstrapped on first run with a fully-commented defaults file. Configurable symbols (incl. nerd-font alternatives), colors, optional columns, section visibility, and PR hyperlinks.
- **Pass-through to git** — any unknown subcommand forwards to `git` so `gist` can replace the `git` alias.
- **Color** — `NO_COLOR` and `GIST_COLOR=always|never` honored; auto-detect TTY otherwise.

### Project
- GPL-3.0 license.
- Build, lint, security, and release CI workflows (Linux + macOS).
- `Makefile` with `build`, `install`, `test`, `lint`, `vuln`, `release-snapshot`.
- Version metadata (`version`, `commit`, `date`) injected via `-ldflags` and surfaced by `gist version`.
- Layered package structure (`main` → `internal/<pkg>` → `test/integration`).
