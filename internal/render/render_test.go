package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ArturGulik/gist/internal/ansi"
	"github.com/ArturGulik/gist/internal/app"
	"github.com/ArturGulik/gist/internal/config"
	"github.com/ArturGulik/gist/internal/model"
)

var updateGolden = flag.Bool("update", false, "rewrite render-test golden files instead of comparing")

// renderCase is a single golden-file scenario for the renderer.
type renderCase struct {
	name  string
	state model.RepoState
	color bool
	tweak func(*config.Config)
}

func TestRender(t *testing.T) {
	cases := []renderCase{
		{
			name: "single_branch",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "main",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsCurrent: true, IsDefault: true,
						Upstream: "origin/main", Subject: "Initial commit"},
				},
			},
		},
		{
			name: "multi_branch_sync_states",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "feature",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsDefault: true, Upstream: "origin/main"},
					{Name: "ahead-only", Hash: "13a578d", Upstream: "origin/ahead-only", Ahead: 2},
					{Name: "behind-only", Hash: "7150cc9", Upstream: "origin/behind-only", Behind: 3},
					{Name: "diverged", Hash: "560787b", Upstream: "origin/diverged", Ahead: 1, Behind: 1},
					{Name: "no-upstream", Hash: "e11597a"},
					{Name: "feature", Hash: "594d6e9", IsCurrent: true,
						Upstream: "origin/feature", Ahead: 1},
				},
			},
		},
		{
			name: "gone_upstream",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "main",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsCurrent: true, IsDefault: true, Upstream: "origin/main"},
					{Name: "old-feature", Hash: "139e828", Upstream: "origin/old-feature", Gone: true},
				},
			},
		},
		{
			name: "remote_only",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "main",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsCurrent: true, IsDefault: true, Upstream: "origin/main"},
					{Name: "remote-feature", Hash: "a3b2c1d", IsRemoteOnly: true},
				},
			},
		},
		{
			name: "with_prs",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "open-pr",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsDefault: true, Upstream: "origin/main"},
					{Name: "open-pr", Hash: "13a578d", IsCurrent: true, Upstream: "origin/open-pr",
						Ahead: 2, PRNumber: 87, PRState: "open"},
					{Name: "draft-pr", Hash: "7150cc9", Upstream: "origin/draft-pr",
						PRNumber: 92, PRState: "open", PRIsDraft: true},
					{Name: "merged-pr", Hash: "560787b", Upstream: "origin/merged-pr",
						PRNumber: 81, PRState: "merged"},
					{Name: "closed-pr", Hash: "139e828", Upstream: "origin/closed-pr",
						PRNumber: 73, PRState: "closed"},
				},
			},
		},
		{
			name: "in_progress_with_stash_and_status",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "feature",
				InProgress:    "rebase",
				StashCount:    2,
				StatusRaw:     []byte(" M render.go\n?? newfile.txt\n"),
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsDefault: true, Upstream: "origin/main"},
					{Name: "feature", Hash: "13a578d", IsCurrent: true, Upstream: "origin/feature", Ahead: 1},
				},
			},
		},
		{
			name: "detached_head",
			state: model.RepoState{
				DefaultBranch: "main",
				DetachedHead:  true,
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsDefault: true, Upstream: "origin/main"},
				},
			},
		},
		{
			name: "optional_columns_all_on",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "feature",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsDefault: true, Upstream: "origin/main",
						LastActivity: "2 days ago", Subject: "Initial commit"},
					{Name: "feature", Hash: "13a578d1", IsCurrent: true, Upstream: "origin/feature",
						Ahead: 2, LastActivity: "3 hours ago", Subject: "Add a thing"},
				},
			},
			tweak: func(c *config.Config) {
				c.Status.ShowHash = true
				c.Status.ShowDate = true
				c.Status.ShowSubject = true
			},
		},
		{
			name: "single_branch_colored",
			state: model.RepoState{
				DefaultBranch: "main",
				CurrentBranch: "main",
				Branches: []model.Branch{
					{Name: "main", Hash: "e956f1b", IsCurrent: true, IsDefault: true,
						Upstream: "origin/main", Ahead: 1},
				},
			},
			color: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Status.HyperlinkPRs = false // avoid shelling to git for OSC-8 URLs
			if tc.tweak != nil {
				tc.tweak(&cfg)
			}

			var buf bytes.Buffer
			a := &app.App{
				Cfg:   &cfg,
				Color: tc.color,
				Pen:   ansi.Pen{Color: tc.color},
				Out:   &buf,
				Err:   &buf,
			}
			r := New(a)
			if err := r.Status(&tc.state); err != nil {
				t.Fatalf("Status: %v", err)
			}
			checkGolden(t, filepath.Join("testdata", "render", tc.name+".golden"), buf.Bytes())
		})
	}
}

func checkGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v\nrun `go test -run %s -update` to create it", path, err, t.Name())
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output mismatch (run `go test -run %s -update` to refresh)\n--- got ---\n%s\n--- want ---\n%s", t.Name(), got, want)
	}
}
