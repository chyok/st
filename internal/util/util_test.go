package util

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestGetDirFilePathsRelativeKeepsSubdirs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "share")
	for _, f := range []string{"top.txt", "sub/deep.txt", "sub/more/deeper.txt"} {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := GetDirFilePaths(root, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"share/sub/deep.txt", "share/sub/more/deeper.txt", "share/top.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetDirFilePaths = %v, want %v", got, want)
	}
}
