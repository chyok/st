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
