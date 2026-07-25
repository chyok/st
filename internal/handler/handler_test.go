package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func multipartBody(t *testing.T, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("other", "ignored")
	for name, content := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, name))
		h.Set("Content-Type", "application/octet-stream")
		part, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(part, content)
	}
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestReceiveHandlerUpload(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("exists.txt", []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	body, ctype := multipartBody(t, map[string]string{
		"folder/sub/a.txt": "hello",
		"exists.txt":       "new",
	})
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	ReceiveHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}

	for path, want := range map[string]string{
		"folder/sub/a.txt": "hello", // folder structure kept
		"exists.txt":       "old",   // existing file untouched
		"exists (1).txt":   "new",
	} {
		got, err := os.ReadFile(filepath.FromSlash(path))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
}

func TestReceiveHandlerRejectsBadUploads(t *testing.T) {
	t.Chdir(t.TempDir())
	cases := map[string]map[string]string{
		"path traversal": {"../escape.txt": "x"},
		"absolute path":  {"/tmp/escape.txt": "x"},
		"no file part":   {},
	}
	for name, files := range cases {
		body, ctype := multipartBody(t, files)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		req.Header.Set("Content-Type", ctype)
		rec := httptest.NewRecorder()
		ReceiveHandler(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "escape.txt")); err == nil {
		t.Error("upload escaped the receive directory")
	}
}

// shareWithEscapingLinks builds share/ containing an ordinary file, a link to
// that file (allowed), and links to a file and a directory outside share/.
func shareWithEscapingLinks(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "share")
	outside := filepath.Join(base, "secret")
	for _, d := range []string{root, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(root, "ok.txt"), []byte("ok"), 0o644))
	must(os.WriteFile(filepath.Join(outside, "key"), []byte("SECRET"), 0o644))
	if err := os.Symlink("ok.txt", filepath.Join(root, "inner-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	must(os.Symlink(filepath.Join(outside, "key"), filepath.Join(root, "file-link")))
	must(os.Symlink(outside, filepath.Join(root, "dir-link")))
	return root
}

func TestSymlinksCannotEscapeShare(t *testing.T) {
	config.G.FilePath = shareWithEscapingLinks(t)

	get := func(h http.HandlerFunc, method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(method, path, nil))
		return rec
	}

	if rec := get(FileServerHandler, http.MethodGet, "/download/share/inner-link"); rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Errorf("link inside share = %d %q, want 200 \"ok\"", rec.Code, rec.Body)
	}
	for _, p := range []string{"/download/share/file-link", "/download/share/dir-link/key", "/download/share/dir-link"} {
		if rec := get(FileServerHandler, http.MethodGet, p); rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "SECRET") || strings.Contains(rec.Body.String(), "key") {
			t.Errorf("GET %s = %d %q, want it refused", p, rec.Code, rec.Body)
		}
	}
	page := get(SendHandler, http.MethodGet, "/").Body.String()
	if !strings.Contains(page, "inner-link") || strings.Contains(page, "file-link") || strings.Contains(page, "dir-link") {
		t.Errorf("listing should show inner-link only among links:\n%s", page)
	}
	if rec := get(SendHandler, http.MethodGet, "/dir-link"); rec.Code == http.StatusOK {
		t.Errorf("listing a directory outside the share = %d, want refused", rec.Code)
	}

	var info struct{ Paths []string }
	rec := get(SendHandler, http.MethodPost, "/")
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	want := []string{"share/inner-link", "share/ok.txt"}
	if !slices.Equal(info.Paths, want) {
		t.Errorf("advertised paths = %v, want %v", info.Paths, want)
	}
}
