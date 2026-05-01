package collect

import "testing"

func TestParseTrack(t *testing.T) {
	cases := []struct {
		in     string
		ahead  int
		behind int
		gone   bool
	}{
		{"", 0, 0, false},
		{"[gone]", 0, 0, true},
		{"[ahead 2]", 2, 0, false},
		{"[behind 1]", 0, 1, false},
		{"[ahead 2, behind 1]", 2, 1, false},
		{"[behind 4, ahead 7]", 7, 4, false},
		{"  [ahead 3]  ", 3, 0, false},
		{"[ahead]", 0, 0, false},
		{"[ahead foo]", 0, 0, false},
		{"[wat]", 0, 0, false},
	}
	for _, c := range cases {
		a, b, g := ParseTrack(c.in)
		if a != c.ahead || b != c.behind || g != c.gone {
			t.Errorf("ParseTrack(%q) = (%d, %d, %v); want (%d, %d, %v)",
				c.in, a, b, g, c.ahead, c.behind, c.gone)
		}
	}
}
