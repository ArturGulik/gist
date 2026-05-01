package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/git"
)

// DefaultText is the canonical, fully-documented default config — every
// field with its default value and a comment in non-obvious cases.
// Two consumers:
//  1. Bootstrap writes it to ~/.config/gist/config on first run so users have
//     a self-documenting starting point.
//  2. `gist config` prints it on demand — handy for `gist config > path`.
//
// Invariant: applying this text to Default() must yield Default() unchanged.
// TestDefaultConfigTextRoundTrip pins this so the two can't drift.
const DefaultText = `# gist configuration — git-config-style INI.
# This file was generated on first run. Edit freely; gist won't overwrite it.
# To reset to defaults, delete ~/.config/gist/ and re-run gist, or run:
#     gist config > ~/.config/gist/config

[status]
    # Append commit subject as a column on each branch row.
    show-subject = false

    # Append short commit hash column.
    show-hash = false

    # Append relative committer date (e.g. "2 days ago").
    show-date = false

    # OSC-8 hyperlink PR numbers to their forge URL. Requires a terminal
    # that supports OSC-8 (most modern ones do).
    hyperlink-prs = true

[sections]
    # "N stash" line below the branch list.
    stash = true

    # ` + "`git status -s`" + ` output (uncommitted changes) below the branch list.
    status-footer = true

    # "⚠ rebase in progress" banner above the branch list.
    in-progress-banner = true

[symbol]
    # Sync indicators (precede ahead/behind counts).
    ahead = "↑"
    behind = "↓"
    no-upstream = "◦"
    remote-only = "⇣"
    in-progress = "⚠"

    # PR/MR state glyphs (precede the PR number).
    pr-open = "#"
    pr-draft = "~"
    pr-merged = "✓"
    pr-closed = "×"

[color]
    # Color values: any combination of  bold dim italic strike  plus a
    # foreground name  red green yellow blue magenta cyan ,
    # space-separated. Empty string disables styling for that role.

    # Branch-name styling. Precedence (top wins): gone > current > default
    # > remote-only > pr-merged.
    branch-current = "bold green"
    branch-default = "bold"
    branch-gone = "strike dim"
    branch-remote-only = "dim"
    branch-pr-merged = "dim"

    # PR/MR number styling (paired with [symbol] glyphs above).
    pr-open = "cyan"
    pr-draft = "dim cyan"
    pr-merged = "dim green"
    pr-closed = "red"

    # Sync indicator styling.
    sync-ahead = "yellow"
    sync-behind = "red"
    sync-no-upstream = "dim"

    # In-progress banner color.
    in-progress = "bold yellow"

    # Divider between branch list and status footer.
    divider = "dim"

    # Generic dim-meta style (stash row, hash/date columns, "in sync"
    # labels in 'gist branch', etc.).
    status-meta = "dim"
`

// Config holds every knob users can set. Construct with Default() and then
// overlay file values via Load — fields not mentioned in the file keep their
// defaults. Color fields are ansi.Style values already resolved from their
// textual ("bold green") form, so the renderer never re-parses.
type Config struct {
	Status   StatusConfig
	Sections SectionsConfig
	Symbols  Symbols
	Colors   Colors
}

// StatusConfig controls what the default `gist status` view shows. The
// concise current look is preserved by defaulting all extra columns off;
// users opt in.
type StatusConfig struct {
	ShowSubject  bool
	ShowHash     bool
	ShowDate     bool // relative committer date
	HyperlinkPRs bool
}

type SectionsConfig struct {
	Stash            bool
	StatusFooter     bool
	InProgressBanner bool
}

type Symbols struct {
	Ahead, Behind, NoUpstream, RemoteOnly, InProgress string
	PROpen, PRDraft, PRMerged, PRClosed               string
}

type Colors struct {
	BranchCurrent, BranchDefault, BranchGone, BranchRemoteOnly, BranchPRMerged ansi.Style
	PROpen, PRDraft, PRMerged, PRClosed                                        ansi.Style
	SyncAhead, SyncBehind, SyncNoUpstream                                      ansi.Style
	InProgress, Divider, StatusMeta                                            ansi.Style
}

// Default matches the existing hardcoded behavior 1:1 — loading no config
// file results in identical output to the pre-config era.
func Default() Config {
	return Config{
		Status: StatusConfig{
			HyperlinkPRs: true,
		},
		Sections: SectionsConfig{
			Stash:            true,
			StatusFooter:     true,
			InProgressBanner: true,
		},
		Symbols: Symbols{
			Ahead:      "↑",
			Behind:     "↓",
			NoUpstream: "◦",
			RemoteOnly: "⇣",
			InProgress: "⚠",
			PROpen:     "#",
			PRDraft:    "~",
			PRMerged:   "✓",
			PRClosed:   "×",
		},
		// SGR-code order matches parseColor's textual token order
		// (modifier-first), so Default and the embedded DefaultText
		// round-trip to deep-equal Configs.
		Colors: Colors{
			BranchCurrent:    ansi.S(ansi.SgrBold, ansi.FgGreen),
			BranchDefault:    ansi.S(ansi.SgrBold),
			BranchGone:       ansi.S(ansi.SgrStrike, ansi.SgrDim),
			BranchRemoteOnly: ansi.S(ansi.SgrDim),
			BranchPRMerged:   ansi.S(ansi.SgrDim),
			PROpen:           ansi.S(ansi.FgCyan),
			PRDraft:          ansi.S(ansi.SgrDim, ansi.FgCyan),
			PRMerged:         ansi.S(ansi.SgrDim, ansi.FgGreen),
			PRClosed:         ansi.S(ansi.FgRed),
			SyncAhead:        ansi.S(ansi.FgYellow),
			SyncBehind:       ansi.S(ansi.FgRed),
			SyncNoUpstream:   ansi.S(ansi.SgrDim),
			InProgress:       ansi.S(ansi.SgrBold, ansi.FgYellow),
			Divider:          ansi.S(ansi.SgrDim),
			StatusMeta:       ansi.S(ansi.SgrDim),
		},
	}
}

// userConfigDir returns the gist global config directory ($XDG_CONFIG_HOME/gist
// or ~/.config/gist). Returns "" if neither is resolvable.
func userConfigDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "gist")
}

// Bootstrap writes DefaultText to the global config path on first run,
// identified by the gist config directory not existing. Once the directory
// exists, this function is a no-op even if the user later deletes the file —
// predictable behavior, and avoids fighting the user.
//
// Returns the bootstrap path on success (so main can announce it), "" if no
// write happened, and an error if writing failed.
func Bootstrap() (string, error) {
	dir := userConfigDir()
	if dir == "" {
		return "", nil
	}
	if _, err := os.Stat(dir); err == nil {
		return "", nil // already bootstrapped (or user-created)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(DefaultText), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Load builds a Config by overlaying global then per-repo config files
// onto the defaults. Missing files are not errors — gist must work in any
// environment without ceremony.
func Load() (Config, error) {
	cfg := Default()
	for _, path := range configPaths() {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cfg, fmt.Errorf("open %s: %w", path, err)
		}
		err = applyConfigFile(&cfg, f, path)
		f.Close()
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// configPaths returns the ordered list of files to read, lowest priority first.
// Per-repo overrides global. Missing repo (or git-dir lookup failure) just
// skips the per-repo file.
func configPaths() []string {
	var paths []string
	if home, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(home, "gist", "config"))
	}
	if d, err := git.Run("rev-parse", "--git-common-dir"); err == nil {
		if !filepath.IsAbs(d) {
			if top, err2 := git.Run("rev-parse", "--show-toplevel"); err2 == nil {
				d = filepath.Join(top, d)
			}
		}
		paths = append(paths, filepath.Join(d, "gist", "config"))
	}
	return paths
}

// applyConfigFile parses a git-config-style INI and mutates cfg. The format:
//
//	[section]
//	    key = value           # comment
//	    other = "quoted val"  ; or this style
//
// Keys outside any section, unknown sections, and unknown keys all produce
// an error so typos aren't silent. The path argument is used in error messages.
func applyConfigFile(cfg *Config, r io.Reader, path string) error {
	scanner := bufio.NewScanner(r)
	section := ""
	lineno := 0
	for scanner.Scan() {
		lineno++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return fmt.Errorf("%s:%d: expected key = value", path, lineno)
		}
		key := strings.TrimSpace(line[:eq])
		val := unquote(strings.TrimSpace(line[eq+1:]))
		if section == "" {
			return fmt.Errorf("%s:%d: key %q outside any [section]", path, lineno, key)
		}
		if err := setConfigValue(cfg, section, key, val); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineno, err)
		}
	}
	return scanner.Err()
}

// stripComment removes everything from the first unquoted # or ; onward.
func stripComment(s string) string {
	inQuote := false
	for i, r := range s {
		if r == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote && (r == '#' || r == ';') {
			return s[:i]
		}
	}
	return s
}

// unquote strips a single pair of surrounding double quotes. Doesn't handle
// escapes — values with embedded quotes aren't a real use case here.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// setConfigValue dispatches a single (section, key, value) tuple to the right
// Config field. Returns an error for unknown sections/keys (typo-catching) or
// invalid values (e.g. unparseable bool, unknown color token).
func setConfigValue(cfg *Config, section, key, val string) error {
	switch section {
	case "status":
		switch key {
		case "show-subject":
			return setBool(&cfg.Status.ShowSubject, val)
		case "show-hash":
			return setBool(&cfg.Status.ShowHash, val)
		case "show-date":
			return setBool(&cfg.Status.ShowDate, val)
		case "hyperlink-prs":
			return setBool(&cfg.Status.HyperlinkPRs, val)
		}
	case "sections":
		switch key {
		case "stash":
			return setBool(&cfg.Sections.Stash, val)
		case "status-footer":
			return setBool(&cfg.Sections.StatusFooter, val)
		case "in-progress-banner":
			return setBool(&cfg.Sections.InProgressBanner, val)
		}
	case "symbol":
		target := symbolField(&cfg.Symbols, key)
		if target == nil {
			return fmt.Errorf("unknown key symbol.%s", key)
		}
		*target = val
		return nil
	case "color":
		target := colorField(&cfg.Colors, key)
		if target == nil {
			return fmt.Errorf("unknown key color.%s", key)
		}
		s, err := parseColor(val)
		if err != nil {
			return fmt.Errorf("color.%s: %w", key, err)
		}
		*target = s
		return nil
	default:
		return fmt.Errorf("unknown section [%s]", section)
	}
	return fmt.Errorf("unknown key %s.%s", section, key)
}

func symbolField(s *Symbols, key string) *string {
	switch key {
	case "ahead":
		return &s.Ahead
	case "behind":
		return &s.Behind
	case "no-upstream":
		return &s.NoUpstream
	case "remote-only":
		return &s.RemoteOnly
	case "in-progress":
		return &s.InProgress
	case "pr-open":
		return &s.PROpen
	case "pr-draft":
		return &s.PRDraft
	case "pr-merged":
		return &s.PRMerged
	case "pr-closed":
		return &s.PRClosed
	}
	return nil
}

func colorField(c *Colors, key string) *ansi.Style {
	switch key {
	case "branch-current":
		return &c.BranchCurrent
	case "branch-default":
		return &c.BranchDefault
	case "branch-gone":
		return &c.BranchGone
	case "branch-remote-only":
		return &c.BranchRemoteOnly
	case "branch-pr-merged":
		return &c.BranchPRMerged
	case "pr-open":
		return &c.PROpen
	case "pr-draft":
		return &c.PRDraft
	case "pr-merged":
		return &c.PRMerged
	case "pr-closed":
		return &c.PRClosed
	case "sync-ahead":
		return &c.SyncAhead
	case "sync-behind":
		return &c.SyncBehind
	case "sync-no-upstream":
		return &c.SyncNoUpstream
	case "in-progress":
		return &c.InProgress
	case "divider":
		return &c.Divider
	case "status-meta":
		return &c.StatusMeta
	}
	return nil
}

func setBool(target *bool, val string) error {
	switch strings.ToLower(val) {
	case "true", "yes", "on", "1":
		*target = true
	case "false", "no", "off", "0", "":
		*target = false
	default:
		return fmt.Errorf("invalid bool %q", val)
	}
	return nil
}

// colorTokens maps every word the user can write in a color value to the
// SGR code it produces. Add more here (e.g. "underline", "white") as needed.
var colorTokens = map[string]string{
	"bold":    ansi.SgrBold,
	"dim":     ansi.SgrDim,
	"italic":  ansi.SgrItalic,
	"strike":  ansi.SgrStrike,
	"red":     ansi.FgRed,
	"green":   ansi.FgGreen,
	"yellow":  ansi.FgYellow,
	"blue":    ansi.FgBlue,
	"magenta": ansi.FgMagenta,
	"cyan":    ansi.FgCyan,
}

// parseColor turns "bold green" / "strike dim" / "" into a Style. Empty value
// produces an empty Style (no styling). Unknown tokens are an error so typos
// fail loudly.
func parseColor(val string) (ansi.Style, error) {
	if strings.TrimSpace(val) == "" {
		return ansi.Style{}, nil
	}
	var codes []string
	for _, tok := range strings.Fields(val) {
		code, ok := colorTokens[strings.ToLower(tok)]
		if !ok {
			return ansi.Style{}, fmt.Errorf("unknown color token %q", tok)
		}
		codes = append(codes, code)
	}
	return ansi.Style(codes), nil
}
