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
)

func TestFileServerHandlerDir(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "share")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := (&Sender{Path: root}).Routes()

	cases := map[string]int{
		"/download/share/sub/a.txt": http.StatusOK,
		"/download/x":               http.StatusNotFound, // shorter than prefix: used to panic
		"/download/":                http.StatusNotFound,
		"/download/shareXsub/a.txt": http.StatusNotFound, // same length, wrong prefix
		"/download/share/missing":   http.StatusNotFound,
	}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}
}

func TestSendHandlerMissingPathIs404(t *testing.T) {
	t.Parallel()
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
		h := (&Sender{Path: c.shared}).Routes()
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("sharing %s: GET %s = %d, want %d", filepath.Base(c.shared), c.path, rec.Code, c.want)
		}
	}
}

func multipartBody(t *testing.T, files map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("other", "ignored"); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, name))
		h.Set("Content-Type", "application/octet-stream")
		part, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, mw.FormDataContentType()
}

func TestReceiveHandlerUpload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := (&Receiver{Dir: dir}).Routes()
	if err := os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	body, ctype := multipartBody(t, map[string]string{
		"folder/sub/a.txt": "hello",
		"exists.txt":       "new",
	})
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}

	for path, want := range map[string]string{
		"folder/sub/a.txt": "hello", // folder structure kept
		"exists.txt":       "old",   // existing file untouched
		"exists (1).txt":   "new",
	} {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil || string(got) != want {
			t.Errorf("%s = %q, %v; want %q", path, got, err, want)
		}
	}
}

func TestReceiveHandlerRejectsBadUploads(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := (&Receiver{Dir: dir}).Routes()
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
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "escape.txt")); err == nil {
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
	t.Parallel()
	h := (&Sender{Path: shareWithEscapingLinks(t)}).Routes()

	get := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}

	if rec := get(http.MethodGet, "/download/share/inner-link"); rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Errorf("link inside share = %d %q, want 200 \"ok\"", rec.Code, rec.Body)
	}
	for _, p := range []string{"/download/share/file-link", "/download/share/dir-link/key", "/download/share/dir-link"} {
		if rec := get(http.MethodGet, p); rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "SECRET") || strings.Contains(rec.Body.String(), "key") {
			t.Errorf("GET %s = %d %q, want it refused", p, rec.Code, rec.Body)
		}
	}
	page := get(http.MethodGet, "/").Body.String()
	if !strings.Contains(page, "inner-link") || strings.Contains(page, "file-link") || strings.Contains(page, "dir-link") {
		t.Errorf("listing should show inner-link only among links:\n%s", page)
	}
	if rec := get(http.MethodGet, "/dir-link"); rec.Code == http.StatusOK {
		t.Errorf("listing a directory outside the share = %d, want refused", rec.Code)
	}

	var info struct{ Paths []string }
	rec := get(http.MethodPost, "/")
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	want := []string{"share/inner-link", "share/ok.txt"}
	if !slices.Equal(info.Paths, want) {
		t.Errorf("advertised paths = %v, want %v", info.Paths, want)
	}
}

func TestReceiveHandlerSaveFailureIs500(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	h := (&Receiver{Dir: dir}).Routes()
	// A plain file where the upload needs a directory makes saving fail.
	if err := os.WriteFile(filepath.Join(dir, "blocked"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	body, ctype := multipartBody(t, map[string]string{"blocked/a.txt": "x"})
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", ctype)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", rec.Code)
	}
}

func TestRoutesRejectOtherMethods(t *testing.T) {
	t.Parallel()
	for name, h := range map[string]http.Handler{
		"receiver": (&Receiver{Dir: t.TempDir()}).Routes(),
		"sender":   (&Sender{Path: t.TempDir()}).Routes(),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/", nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: PUT / = %d, want 405", name, rec.Code)
		}
	}
}

func TestReceiverServesPageAndStatic(t *testing.T) {
	t.Parallel()
	h := (&Receiver{DeviceName: "<my-box>", Dir: t.TempDir()}).Routes()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "&lt;my-box&gt;") {
		t.Errorf("upload page = %d, want 200 with the escaped device name", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/css/upload.css", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("static css = %d, want 200", rec.Code)
	}
}

func TestDownloadPageEscapesLinks(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "share")
	if err := os.MkdirAll(filepath.Join(root, "d?x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a#1.txt"), []byte("hash"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := (&Sender{Path: root}).Routes()

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	page := rec.Body.String()
	for _, link := range []string{`href="/d%3Fx"`, `href="/download/share/a%231.txt"`} {
		if !strings.Contains(page, link) {
			t.Errorf("page lacks %s:\n%s", link, page)
		}
	}

	// The escaped links must lead to the right places.
	for path, want := range map[string]int{"/d%3Fx": http.StatusOK, "/download/share/a%231.txt": http.StatusOK} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}
}
