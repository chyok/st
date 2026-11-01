package protocol

import "testing"

func TestEscapePath(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"share/a.txt":         "share/a.txt",
		"share/a b#1?.txt":    "share/a%20b%231%3F.txt",
		"share/100% done.txt": "share/100%25%20done.txt",
		"d?x/sub":             "d%3Fx/sub",
		"/":                   "/",
	}
	for in, want := range cases {
		if got := EscapePath(in); got != want {
			t.Errorf("EscapePath(%q) = %q, want %q", in, got, want)
		}
	}
}
