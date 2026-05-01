package render

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/collect"
	"github.com/ArturGulik/gist/internal/config"
	"github.com/ArturGulik/gist/internal/git"
	"github.com/ArturGulik/gist/internal/model"
	"github.com/ArturGulik/gist/internal/update"
)

// Renderer holds the ambient state every status/branch render needs:
// the config, the pen (color flag), and the output writer. Construct one
// per render via New(a).
type Renderer struct {
	cfg *config.Config
	pen ansi.Pen
	out io.Writer
}

// New constructs a Renderer from an App.
func New(a *app.App) *Renderer {
	return &Renderer{cfg: a.Cfg, pen: a.Pen, out: a.Out}
}

// RunStatus is the entry point for the `status` subcommand.
func RunStatus(a *app.App, _ []string) error {
	if !git.InWorkTree() {
		fmt.Fprintln(a.Out, "not a git repository")
		return nil
	}
	if !git.HasCommits() {
		fmt.Fprintln(a.Out, "(no commits yet)")
		return nil
	}

	state, err := collect.RepoState(a.Color)
	if err != nil {
		return err
	}
	return New(a).Status(state)
}

// Status renders the full multi-branch status view to r.out.
func (r *Renderer) Status(s *model.RepoState) error {
	cfg := r.cfg

	if s.InProgress != "" && cfg.Sections.InProgressBanner {
		fmt.Fprintln(r.out, r.pen.Format(cfg.Colors.InProgress, "%s %s in progress", cfg.Symbols.InProgress, s.InProgress))
	}
	if s.DetachedHead {
		fmt.Fprintln(r.out, r.pen.Style("(detached HEAD)", ansi.FgYellow))
	}

	sortBranches(s.Branches, s.DefaultBranch)

	// Pre-compute per-column widths so the optional columns line up.
	indW, hashW, dateW := 0, 0, 0
	for _, b := range s.Branches {
		if n := ansi.VisibleWidth(r.indicator(b)); n > indW {
			indW = n
		}
		if cfg.Status.ShowHash && len(b.Hash) > hashW {
			hashW = len(b.Hash)
		}
		if cfg.Status.ShowDate && len(b.LastActivity) > dateW {
			dateW = len(b.LastActivity)
		}
	}

	// Branch-name column width is needed only when something appears to its
	// right (hash / date / subject). When nothing follows, names render at
	// natural width — preserves the current concise look.
	hasTrailingCol := cfg.Status.ShowHash || cfg.Status.ShowDate || cfg.Status.ShowSubject
	nameW := 0
	if hasTrailingCol {
		for _, b := range s.Branches {
			if n := len(b.Name); n > nameW {
				nameW = n
			}
		}
	}

	for _, b := range s.Branches {
		r.branchRow(b, indW, nameW, hashW, dateW)
	}

	if s.StashCount > 0 && cfg.Sections.Stash {
		fmt.Fprintln(r.out, r.pen.Apply(cfg.Colors.StatusMeta, fmt.Sprintf("  %d stash", s.StashCount)))
	}

	if cfg.Sections.StatusFooter && len(s.StatusRaw) > 0 {
		fmt.Fprintln(r.out, r.pen.Apply(cfg.Colors.Divider, "───────────────────"))
		_, _ = r.out.Write(s.StatusRaw)
	}

	return nil
}

// Branch renders a single branch row at natural width — used by `gist switch`
// to print a status line for the resulting branch.
func (r *Renderer) Branch(b model.Branch) {
	r.branchRow(b, 0, 0, 0, 0)
}

// sortBranches puts the default branch first, then local branches
// alphabetically, then remote-only branches alphabetically.
func sortBranches(bs []model.Branch, defaultBranch string) {
	rank := func(b model.Branch) int {
		switch {
		case b.Name == defaultBranch:
			return 0
		case !b.IsRemoteOnly:
			return 1
		default:
			return 2
		}
	}
	sort.SliceStable(bs, func(i, j int) bool {
		ri, rj := rank(bs[i]), rank(bs[j])
		if ri != rj {
			return ri < rj
		}
		return bs[i].Name < bs[j].Name
	})
}

func (r *Renderer) branchRow(b model.Branch, indW, nameW, hashW, dateW int) {
	cfg := r.cfg
	nameStyled := b.Name
	switch {
	case b.Gone:
		// Upstream was configured but the remote ref is gone. Distinct from
		// "never pushed" (b.Upstream == ""), which git itself distinguishes
		// via %(upstream:track).
		nameStyled = r.pen.Apply(cfg.Colors.BranchGone, b.Name)
	case b.IsCurrent:
		nameStyled = r.pen.Apply(cfg.Colors.BranchCurrent, b.Name)
	case b.IsDefault:
		nameStyled = r.pen.Apply(cfg.Colors.BranchDefault, b.Name)
	case b.IsRemoteOnly:
		nameStyled = r.pen.Apply(cfg.Colors.BranchRemoteOnly, b.Name)
	case b.PRState == "merged":
		nameStyled = r.pen.Apply(cfg.Colors.BranchPRMerged, b.Name)
	}

	ind := r.indicator(b)
	if pad := indW - ansi.VisibleWidth(ind); pad > 0 {
		ind += strings.Repeat(" ", pad)
	}

	// Pad the branch name itself so optional columns to the right align.
	if nameW > 0 {
		if pad := nameW - len(b.Name); pad > 0 {
			nameStyled += strings.Repeat(" ", pad)
		}
	}

	parts := make([]string, 0, 5)
	if ind != "" {
		parts = append(parts, ind)
	}
	parts = append(parts, nameStyled)

	if cfg.Status.ShowHash {
		hash := b.Hash
		if pad := hashW - len(hash); pad > 0 {
			hash += strings.Repeat(" ", pad)
		}
		parts = append(parts, r.pen.Apply(cfg.Colors.StatusMeta, hash))
	}
	if cfg.Status.ShowDate {
		date := b.LastActivity
		if pad := dateW - len(date); pad > 0 {
			date += strings.Repeat(" ", pad)
		}
		parts = append(parts, r.pen.Apply(cfg.Colors.StatusMeta, date))
	}
	if cfg.Status.ShowSubject {
		parts = append(parts, b.Subject)
	}

	fmt.Fprintln(r.out, strings.Join(parts, " "))
}

// indicator composes the pre-name status column from independent segments.
// Each segment function returns "" when it has nothing to say.
func (r *Renderer) indicator(b model.Branch) string {
	cfg := r.cfg
	segs := []string{
		r.PRSegment(b),
		r.syncSegment(b),
	}
	if b.IsRemoteOnly {
		segs = []string{
			r.PRSegment(b),
			r.pen.Apply(cfg.Colors.BranchRemoteOnly, cfg.Symbols.RemoteOnly),
		}
	}
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, " ")
}

// PRSegment renders PR state (if any) as #N / ✓N / ×N. If
// status.hyperlink-prs is on and the PR's web URL can be derived, the
// rendered segment is wrapped in an OSC-8 hyperlink. Exported because
// `gist branch` reuses the formatter for the MR/PR row.
func (r *Renderer) PRSegment(b model.Branch) string {
	if b.PRNumber == 0 || b.IsDefault {
		return ""
	}
	cfg := r.cfg
	var glyph string
	var s ansi.Style
	switch b.PRState {
	case "merged":
		glyph, s = cfg.Symbols.PRMerged, cfg.Colors.PRMerged
	case "closed":
		glyph, s = cfg.Symbols.PRClosed, cfg.Colors.PRClosed
	case "open":
		fallthrough
	default:
		if b.PRIsDraft {
			glyph, s = cfg.Symbols.PRDraft, cfg.Colors.PRDraft
		} else {
			glyph, s = cfg.Symbols.PROpen, cfg.Colors.PROpen
		}
	}
	text := r.pen.Format(s, "%s%d", glyph, b.PRNumber)
	if cfg.Status.HyperlinkPRs {
		if url := update.PRWebURL(b.PRNumber); url != "" {
			return r.pen.Hyperlink(url, text)
		}
	}
	return text
}

// syncSegment renders local-vs-upstream state. Empty when in perfect sync.
func (r *Renderer) syncSegment(b model.Branch) string {
	cfg := r.cfg
	switch {
	case b.Upstream == "":
		return r.pen.Apply(cfg.Colors.SyncNoUpstream, cfg.Symbols.NoUpstream)
	case b.Gone:
		// Strikethrough on the branch name communicates this state;
		// no indicator glyph needed.
		return ""
	case b.Ahead > 0 && b.Behind > 0:
		return r.pen.Format(cfg.Colors.SyncAhead, "%s%d", cfg.Symbols.Ahead, b.Ahead) +
			r.pen.Format(cfg.Colors.SyncBehind, "%s%d", cfg.Symbols.Behind, b.Behind)
	case b.Ahead > 0:
		return r.pen.Format(cfg.Colors.SyncAhead, "%s%d", cfg.Symbols.Ahead, b.Ahead)
	case b.Behind > 0:
		return r.pen.Format(cfg.Colors.SyncBehind, "%s%d", cfg.Symbols.Behind, b.Behind)
	}
	return ""
}
