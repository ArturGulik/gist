package ansi

import "testing"

func TestVisibleWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"\x1b[31mabc\x1b[0m", 3},
		{"\x1b[1m\x1b[32mhi\x1b[0m", 2},
		{"a\x1b[31mb\x1b[0mc", 3},
		// CSI terminated by '~' (e.g. PageUp keycode) — pre-fix this leaked
		// the digits as visible characters.
		{"\x1b[5~x", 1},
		{"\x1b[200~paste\x1b[201~", 5},
		// Bare two-byte ESC sequence.
		{"\x1bcabc", 3},
		// OSC 8 hyperlink (ST = ESC \) — pre-fix the URL leaked as visible.
		{"\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\", 4},
		// OSC terminated by BEL.
		{"\x1b]0;title\x07x", 1},
		// Hyperlinked text plus following plain content.
		{"\x1b]8;;u\x1b\\ab\x1b]8;;\x1b\\ cd", 5},
	}
	for _, c := range cases {
		if got := VisibleWidth(c.in); got != c.want {
			t.Errorf("VisibleWidth(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}

func TestPenApply(t *testing.T) {
	off := Pen{Color: false}
	on := Pen{Color: true}
	if got := off.Apply(Style{SgrBold}, "hi"); got != "hi" {
		t.Errorf("color-off Apply = %q; want %q", got, "hi")
	}
	if got := on.Apply(Style{}, "hi"); got != "hi" {
		t.Errorf("empty-style Apply = %q; want %q", got, "hi")
	}
	if got := on.Apply(Style{SgrBold, FgGreen}, "hi"); got != SgrBold+FgGreen+"hi"+SgrReset {
		t.Errorf("Apply codes wrong: %q", got)
	}
}
