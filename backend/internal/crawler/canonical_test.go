package crawler

import "testing"

func TestCanonicalize(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"HTTP://Example.com/Path", "http://example.com/Path"},
		{"https://example.com:443/path#frag", "https://example.com/path"},
		{"http://example.com:80/a/../b", "http://example.com/b"},
		{"example.com/test", "http://example.com/test"},
		{"https://example.com/search?q=beta&b=1&a=2", "https://example.com/search?a=2&b=1&q=beta"},
	}
	for _, c := range cases {
		got, _, err := Canonicalize(c.input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", c.input, err)
		}
		if got != c.want {
			t.Fatalf("canonicalize(%s)=%s want %s", c.input, got, c.want)
		}
	}
}