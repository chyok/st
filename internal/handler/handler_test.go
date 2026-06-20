package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chyok/st/config"
)

func TestFileServerHandlerDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "share")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	config.G.FilePath = root

	cases := map[string]int{
		"/download/share/sub/a.txt": http.StatusOK,
		"/download/x":               http.StatusNotFound, // shorter than prefix: used to panic
		"/download/":                http.StatusNotFound,
		"/download/shareXsub/a.txt": http.StatusNotFound, // same length, wrong prefix
		"/download/share/missing":   http.StatusNotFound,
	}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		FileServerHandler(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}
}

func TestSendHandlerMissingPathIs404(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "share")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		shared, path string
		want         int
	}{
		{dir, "/", http.StatusOK},
		{dir, "/missing", http.StatusNotFound},
		{dir, "/favicon.ico", http.StatusNotFound},
		{file, "/", http.StatusOK},
		{file, "/favicon.ico", http.StatusNotFound},
	}
	for _, c := range cases {
		config.G.FilePath = c.shared
		rec := httptest.NewRecorder()
		SendHandler(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("sharing %s: GET %s = %d, want %d", filepath.Base(c.shared), c.path, rec.Code, c.want)
		}
	}
}
