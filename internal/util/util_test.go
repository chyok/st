package util

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestSaveFileNeverOverwrites(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "a.txt")
	want := []string{"a.txt", "a (1).txt", "a (2).txt"}
	for i, name := range want {
		got, err := SaveFile(target, strings.NewReader(name))
		if err != nil {
			t.Fatal(err)
		}
		if got != filepath.Join(dir, "sub", name) {
			t.Errorf("save #%d went to %q, want %q", i, got, name)
		}
	}
	for _, name := range want {
		b, err := os.ReadFile(filepath.Join(dir, "sub", name))
		if err != nil || string(b) != name {
			t.Errorf("%s = %q, %v; want its own content", name, b, err)
		}
	}
	assertNoPartFiles(t, filepath.Join(dir, "sub"))
}

func TestSaveFileDotfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, ".bashrc")
	if _, err := SaveFile(target, strings.NewReader("1")); err != nil {
		t.Fatal(err)
	}
	got, err := SaveFile(target, strings.NewReader("2"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, ".bashrc (1)"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type failingReader struct{ n int }

func (r *failingReader) Read(p []byte) (int, error) {
	if r.n == 0 {
		return 0, errors.New("connection lost")
	}
	r.n--
	return copy(p, "partial"), nil
}

func TestSaveFileFailureLeavesNothing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := SaveFile(filepath.Join(dir, "a.txt"), &failingReader{n: 2}); err == nil {
		t.Fatal("expected error")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("leftover files after failed save: %v", entries)
	}
}

func assertNoPartFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".part") {
			t.Errorf("leftover temp file %s", e.Name())
		}
	}
}

// Not parallel: RemovePartFiles would delete other tests' in-progress saves.
func TestRemovePartFilesCleansInterruptedSave(t *testing.T) {
	dir := t.TempDir()
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		_, err := SaveFile(filepath.Join(dir, "a.txt"), pr)
		done <- err
	}()
	// Write returns once SaveFile is copying into its .part file.
	if _, err := pw.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}

	RemovePartFiles()
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("files left after RemovePartFiles: %v", entries)
	}
	pw.CloseWithError(errors.New("interrupted"))
	if err := <-done; err == nil {
		t.Error("SaveFile succeeded after interruption")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("files left after failed save: %v", entries)
	}
}
