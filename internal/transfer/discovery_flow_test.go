package transfer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chyok/st/internal/handler"
	"github.com/chyok/st/internal/protocol"
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

// fakeSender serves a file list and, for each path, the result of serve.
func fakeSender(t *testing.T, paths []string, serve func(w http.ResponseWriter, path string)) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(protocol.FileList{Type: protocol.Dir, Paths: paths}); err != nil {
			t.Error(err)
		}
	})
	mux.HandleFunc("GET /download/", func(w http.ResponseWriter, r *http.Request) {
		serve(w, strings.TrimPrefix(r.URL.Path, "/download/"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func TestPromptListsAtMostMaxListedFiles(t *testing.T) {
	t.Parallel()
	var paths []string
	for i := range maxListed + 5 {
		paths = append(paths, fmt.Sprintf("share/f%02d.txt", i))
	}
	addr := fakeSender(t, paths, func(w http.ResponseWriter, _ string) {})

	var shown string
	client := &Client{Dir: t.TempDir(), Confirm: func(details, _ string) bool {
		shown = details
		return false
	}}
	if err := client.ReceiveFiles(addr); err != nil {
		t.Fatal(err)
	}
	last := fmt.Sprintf("share/f%02d.txt", maxListed-1)
	hidden := fmt.Sprintf("share/f%02d.txt", maxListed)
	if !strings.Contains(shown, last) || strings.Contains(shown, hidden) || !strings.Contains(shown, "... and 5 more") {
		t.Errorf("prompt should list %d files then \"... and 5 more\":\n%s", maxListed, shown)
	}
}
