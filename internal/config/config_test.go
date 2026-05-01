package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ArturGulik/gist/internal/ansi"
)

// TestDefaultConfigTextRoundTrip pins DefaultText against Default. If a new
// knob is added to one but not the other, this test fails — preventing the
// embedded user-facing config from drifting away from the code defaults.
func TestDefaultConfigTextRoundTrip(t *testing.T) {
	cfg := Default()
	if err := applyConfigFile(&cfg, strings.NewReader(DefaultText), "default"); err != nil {
		t.Fatalf("default config text failed to parse: %v", err)
	}
	want := Default()
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("default config text does not round-trip to Default().\n got: %+v\nwant: %+v", cfg, want)
	}
}

func TestStripComment(t *testing.T) {
	cases := map[string]string{
		`foo`:                 `foo`,
		`foo # comment`:       `foo `,
		`foo ; comment`:       `foo `,
		`"a # b" rest`:        `"a # b" rest`,
		`"a # b" # tail`:      `"a # b" `,
		`# whole line`:        ``,
		`key = "v;v" ; trail`: `key = "v;v" `,
	}
	for in, want := range cases {
		if got := stripComment(in); got != want {
			t.Errorf("stripComment(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestUnquote(t *testing.T) {
	cases := map[string]string{
		`"foo"`: `foo`,
		`foo`:   `foo`,
		`""`:    ``,
		`"a`:    `"a`,
		`a"`:    `a"`,
	}
	for in, want := range cases {
		if got := unquote(in); got != want {
			t.Errorf("unquote(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestParseColor(t *testing.T) {
	t.Run("empty produces empty style", func(t *testing.T) {
		s, err := parseColor("")
		if err != nil {
			t.Fatal(err)
		}
		if len(s) != 0 {
			t.Errorf("got %d codes; want 0", len(s))
		}
	})
	t.Run("multiple tokens", func(t *testing.T) {
		s, err := parseColor("bold green")
		if err != nil {
			t.Fatal(err)
		}
		if len(s) != 2 {
			t.Fatalf("got %d codes; want 2", len(s))
		}
		if s[0] != ansi.SgrBold || s[1] != ansi.FgGreen {
			t.Errorf("wrong codes: %v", s)
		}
	})
	t.Run("case-insensitive", func(t *testing.T) {
		_, err := parseColor("BOLD Green")
		if err != nil {
			t.Errorf("BOLD Green should parse: %v", err)
		}
	})
	t.Run("unknown token errors", func(t *testing.T) {
		_, err := parseColor("octarine")
		if err == nil {
			t.Errorf("octarine should be an error")
		}
	})
}

func TestApplyConfigFile(t *testing.T) {
	t.Run("basic overlay", func(t *testing.T) {
		cfg := Default()
		input := `
[status]
    show-subject = true
    show-date = yes

[symbol]
    ahead = "▲"
    pr-open = ">"

[color]
    pr-open = "bold magenta"

[sections]
    stash = false
`
		if err := applyConfigFile(&cfg, strings.NewReader(input), "test"); err != nil {
			t.Fatal(err)
		}
		if !cfg.Status.ShowSubject {
			t.Error("ShowSubject should be true")
		}
		if !cfg.Status.ShowDate {
			t.Error("ShowDate should be true")
		}
		if cfg.Symbols.Ahead != "▲" {
			t.Errorf("ahead = %q; want ▲", cfg.Symbols.Ahead)
		}
		if cfg.Symbols.PROpen != ">" {
			t.Errorf("pr-open = %q; want >", cfg.Symbols.PROpen)
		}
		if cfg.Sections.Stash {
			t.Error("stash should be false")
		}
		if len(cfg.Colors.PROpen) != 2 {
			t.Errorf("pr-open color got %d codes; want 2", len(cfg.Colors.PROpen))
		}
	})

	t.Run("comments and blank lines", func(t *testing.T) {
		cfg := Default()
		input := `
# top comment
[symbol]
    # inline section comment
    ahead = "X"  ; trailing semi-comment

`
		if err := applyConfigFile(&cfg, strings.NewReader(input), "test"); err != nil {
			t.Fatal(err)
		}
		if cfg.Symbols.Ahead != "X" {
			t.Errorf("ahead = %q; want X", cfg.Symbols.Ahead)
		}
	})

	t.Run("unknown section errors", func(t *testing.T) {
		cfg := Default()
		err := applyConfigFile(&cfg, strings.NewReader("[bogus]\nfoo = 1\n"), "test")
		if err == nil {
			t.Fatal("want error for unknown section")
		}
	})

	t.Run("unknown key errors", func(t *testing.T) {
		cfg := Default()
		err := applyConfigFile(&cfg, strings.NewReader("[status]\nbogus = 1\n"), "test")
		if err == nil {
			t.Fatal("want error for unknown key")
		}
	})

	t.Run("key outside section errors", func(t *testing.T) {
		cfg := Default()
		err := applyConfigFile(&cfg, strings.NewReader("foo = bar\n"), "test")
		if err == nil {
			t.Fatal("want error for orphan key")
		}
	})

	t.Run("invalid bool errors", func(t *testing.T) {
		cfg := Default()
		err := applyConfigFile(&cfg, strings.NewReader("[status]\nshow-subject = maybe\n"), "test")
		if err == nil {
			t.Fatal("want error for non-bool value")
		}
	})
}
