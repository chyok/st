package util

import (
	"path/filepath"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	base := filepath.FromSlash("/tmp/base")
	ok := map[string]string{
		"a.txt":         "/tmp/base/a.txt",
		"dir/a.txt":     "/tmp/base/dir/a.txt",
		"dir/../a.txt":  "/tmp/base/a.txt",
		"./dir/./a.txt": "/tmp/base/dir/a.txt",
	}
	for in, want := range ok {
		got, err := SafeJoin(base, in)
		if err != nil {
			t.Errorf("SafeJoin(%q) unexpected error: %v", in, err)
			continue
		}
		if got != filepath.FromSlash(want) {
			t.Errorf("SafeJoin(%q) = %q, want %q", in, got, want)
		}
	}

	bad := []string{"", ".", "..", "../a.txt", "dir/../../a.txt", "/etc/passwd"}
	for _, in := range bad {
		if got, err := SafeJoin(base, in); err == nil {
			t.Errorf("SafeJoin(%q) = %q, want error", in, got)
		}
	}
}
