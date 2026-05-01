package branch

import (
	"fmt"
	"strings"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/collect"
	"github.com/ArturGulik/gist/internal/git"
	"github.com/ArturGulik/gist/internal/model"
	"github.com/ArturGulik/gist/internal/render"
	"github.com/ArturGulik/gist/internal/update"
)

func RunBranch(a *app.App, args []string) error {
	if !git.InWorkTree() {
		fmt.Fprintln(a.Out, "not a git repository")
		return nil
	}
	if !git.HasCommits() {
		fmt.Fprintln(a.Out, "(no commits yet)")
		return nil
	}

	name := git.CurrentBranch()
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		fmt.Fprintln(a.Err, "gist branch: detached HEAD — specify a branch name")
		return nil
	}
	if _, err := git.Run("show-ref", "--verify", "--quiet", "refs/heads/"+name); err != nil {
		fmt.Fprintf(a.Err, "gist branch: %q not found\n", name)
		return nil
	}

	defaultBranch := collect.DetectDefaultBranch()

	// Collect upstream tracking info via for-each-ref (same source as status).
	raw, _ := git.Run("for-each-ref",
		"--format=%(upstream:short)%00%(upstream:track)",
		"refs/heads/"+name)
	refParts := strings.SplitN(raw, "\x00", 2)
	var upstream, trackRaw string
	if len(refParts) == 2 {
		upstream, trackRaw = refParts[0], refParts[1]
	}
	ahead, behind, gone := collect.ParseTrack(trackRaw)

	// PR/MR from cache.
	cache, _ := update.LoadCache()
	var pr update.CacheEntry
	if cache != nil {
		pr = cache[name]
	}

	cfg := a.Cfg
	pen := a.Pen
	const labelW = 10
	fmt.Fprintf(a.Out, "%s\n\n", pen.Style(name, ansi.SgrBold))

	// Upstream row.
	fmt.Fprintf(a.Out, "%-*s", labelW, "upstream")
	switch {
	case upstream == "":
		fmt.Fprintln(a.Out, pen.Apply(cfg.Colors.StatusMeta, "none (never pushed)"))
	case gone:
		fmt.Fprintf(a.Out, "%s  %s\n", pen.Apply(cfg.Colors.BranchGone, upstream), pen.Apply(cfg.Colors.StatusMeta, "gone"))
	default:
		var segs []string
		if ahead > 0 {
			segs = append(segs, pen.Format(cfg.Colors.SyncAhead, "%s%d", cfg.Symbols.Ahead, ahead))
		}
		if behind > 0 {
			segs = append(segs, pen.Format(cfg.Colors.SyncBehind, "%s%d", cfg.Symbols.Behind, behind))
		}
		if len(segs) == 0 {
			fmt.Fprintf(a.Out, "%s  %s\n", upstream, pen.Apply(cfg.Colors.StatusMeta, "in sync"))
		} else {
			fmt.Fprintf(a.Out, "%s  %s\n", upstream, strings.Join(segs, " "))
		}
	}

	// MR/PR row.
	r := render.New(a)
	fmt.Fprintf(a.Out, "%-*s", labelW, "mr/pr")
	if pr.Number == 0 {
		fmt.Fprintln(a.Out, pen.Apply(cfg.Colors.StatusMeta, "none"))
	} else {
		b := model.Branch{PRNumber: pr.Number, PRState: pr.State, PRIsDraft: pr.IsDraft}
		stateLabel := pr.State
		if pr.IsDraft {
			stateLabel = "draft"
		}
		fmt.Fprintf(a.Out, "%s  %s\n", r.PRSegment(b), stateLabel)
	}

	if defaultBranch == "" || defaultBranch == name {
		return nil
	}

	// Commits ahead of default branch.
	fmt.Fprintln(a.Out)
	commits, err := collect.CommitsAhead(defaultBranch, name)
	if err == nil {
		if len(commits) == 0 {
			fmt.Fprintln(a.Out, pen.Format(cfg.Colors.StatusMeta, "no commits ahead of %s", defaultBranch))
		} else {
			authorW := 0
			for _, c := range commits {
				if l := len(c.Author); l > authorW {
					authorW = l
				}
			}
			if authorW > 24 {
				authorW = 24
			}

			fmt.Fprintf(a.Out, "%d commit(s) ahead of %s:\n\n", len(commits), defaultBranch)
			for _, c := range commits {
				author := c.Author
				if len(author) > authorW {
					author = author[:authorW-1] + "…"
				}
				fmt.Fprintf(a.Out, "%s  %s  %-*s  %s\n",
					pen.Apply(cfg.Colors.StatusMeta, c.Hash),
					pen.Apply(cfg.Colors.StatusMeta, c.Date),
					authorW, author,
					c.Subject,
				)
			}
		}
	}

	// Files changed vs default branch (three-dot diff = since merge base).
	files, err := collect.FilesChanged(defaultBranch, name)
	if err == nil && len(files) > 0 {
		fmt.Fprintf(a.Out, "\n%d file(s) changed:\n\n", len(files))
		for _, f := range files {
			tab := strings.IndexByte(f, '\t')
			if tab < 0 {
				continue
			}
			code := f[:1]
			rest := f[tab+1:]
			var path string
			if code == "R" || code == "C" {
				// rename/copy: OLD\tNEW
				p2 := strings.SplitN(rest, "\t", 2)
				if len(p2) == 2 {
					path = p2[0] + " → " + p2[1]
				} else {
					path = rest
				}
			} else {
				path = rest
			}
			var codeStyled string
			switch code {
			case "A":
				codeStyled = pen.Style("A", ansi.FgGreen)
			case "D":
				codeStyled = pen.Style("D", ansi.FgRed)
			case "M":
				codeStyled = pen.Style("M", ansi.FgYellow)
			case "R":
				codeStyled = pen.Style("R", ansi.FgCyan)
			default:
				codeStyled = code
			}
			fmt.Fprintf(a.Out, "%s  %s\n", codeStyled, path)
		}
	}

	return nil
}
