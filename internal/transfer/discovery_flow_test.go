package transfer

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chyok/st/internal/handler"
)

// answer returns a Confirm func that always gives the same answer.
func answer(yes bool) func(string, string) bool {
	return func(string, string) bool { return yes }
}

// startSender serves shared through the real send handlers, the way a peer
// running "st <shared>" would.
func startSender(t *testing.T, shared string) string {
	t.Helper()
	srv := httptest.NewServer((&handler.Sender{Path: shared}).Routes())
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func TestReceiveFilesSingleFile(t *testing.T) {
	t.Parallel()
	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	addr := startSender(t, src)
	dir := t.TempDir()

	if err := (&Client{Dir: dir, Confirm: answer(false)}).ReceiveFiles(addr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err == nil {
		t.Fatal("file downloaded although the prompt was declined")
	}

	if err := (&Client{Dir: dir, Confirm: answer(true)}).ReceiveFiles(addr); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "a.txt")); err != nil || string(got) != "hello" {
		t.Errorf("a.txt = %q, %v; want %q", got, err, "hello")
	}
}

func TestReceiveFilesDirectoryWithSpecialNames(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "share")
	files := map[string]string{
		"plain.txt":         "1",
		"sub/a b#1.txt":     "2",
		"sub/100% done.txt": "3",
	}
	for name, content := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	addr := startSender(t, root)
	dir := t.TempDir()

	if err := (&Client{Dir: dir, Confirm: answer(true)}).ReceiveFiles(addr); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, "share", filepath.FromSlash(name)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
}
