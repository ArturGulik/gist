package update

import "testing"

func TestNormalizeGitLabState(t *testing.T) {
	cases := map[string]string{
		"opened": "open",
		"merged": "merged",
		"closed": "closed",
		"":       "",
		"locked": "locked",
	}
	for in, want := range cases {
		if got := normalizeGitLabState(in); got != want {
			t.Errorf("normalizeGitLabState(%q) = %q; want %q", in, got, want)
		}
	}
}
