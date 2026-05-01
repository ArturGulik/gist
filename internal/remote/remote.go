package remote

import (
	"fmt"
	"strings"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/git"
)

func RunRemote(a *app.App, _ []string) error {
	if !git.InWorkTree() {
		fmt.Fprintln(a.Out, "not a git repository")
		return nil
	}

	out, err := git.Run("remote")
	if err != nil || strings.TrimSpace(out) == "" {
		fmt.Fprintln(a.Out, "no remotes configured")
		return nil
	}

	pen := a.Pen
	remotes := strings.Split(strings.TrimSpace(out), "\n")
	for i, remote := range remotes {
		if i > 0 {
			fmt.Fprintln(a.Out)
		}
		fmt.Fprintln(a.Out, pen.Style(remote, ansi.SgrBold))

		fetchURL, _ := git.Run("remote", "get-url", remote)
		pushURL, _ := git.Run("remote", "get-url", "--push", remote)

		const labelW = 7
		fmt.Fprintf(a.Out, "  %-*s %s\n", labelW, "fetch", fetchURL)
		if pushURL != fetchURL {
			fmt.Fprintf(a.Out, "  %-*s %s\n", labelW, "push", pushURL)
		}

		if webURL := git.WebURL(fetchURL); webURL != "" {
			fmt.Fprintf(a.Out, "  %-*s %s\n", labelW, "web", pen.Hyperlink(webURL, pen.Style(webURL, ansi.FgCyan)))
		}
	}
	return nil
}
