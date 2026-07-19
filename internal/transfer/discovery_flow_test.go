package transfer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/handler"
)

func setStdin(t *testing.T, input string) {
	t.Helper()
	stdin, reader = strings.NewReader(input), nil
	t.Cleanup(func() { stdin, reader = nil, nil })
}

// startSender serves shared through the real send handlers, the way a peer
// running "st <shared>" would.
func startSender(t *testing.T, shared string) string {
	t.Helper()
	config.G.FilePath = shared
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.SendHandler)
	mux.HandleFunc("/download/", handler.FileServerHandler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func TestReceiveFilesDirectoryWithSpecialNames(t *testing.T) {
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
	t.Chdir(t.TempDir())

	setStdin(t, "y\n")
	if err := ReceiveFiles(addr); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join("share", filepath.FromSlash(name)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
}
