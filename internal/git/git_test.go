package git

import "testing"

func TestWebURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"git@github.com:owner/repo.git", "https://github.com/owner/repo"},
		{"git@gitlab.com:group/sub/repo.git", "https://gitlab.com/group/sub/repo"},
		{"git@github.com:owner/repo", "https://github.com/owner/repo"},
		{"ssh://git@github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"ssh://git@github.com:22/owner/repo.git", "https://github.com/owner/repo"},
		{"https://github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"https://github.com/owner/repo", "https://github.com/owner/repo"},
		{"https://user:pass@github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"http://github.com/owner/repo.git", "https://github.com/owner/repo"},
		{"  git@github.com:owner/repo.git  ", "https://github.com/owner/repo"},
		{"", ""},
		{"file:///srv/git/repo.git", ""},
		{"git@", ""},
	}
	for _, c := range cases {
		if got := WebURL(c.in); got != c.want {
			t.Errorf("WebURL(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}
