package transfer

import (
	"bytes"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPostFileStreamsContent(t *testing.T) {
	data := make([]byte, 3<<20)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "big.bin")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}

	var gotName string
	var gotData []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, h, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer f.Close()
		gotName = h.Filename
		gotData, _ = io.ReadAll(f)
	}))
	defer srv.Close()

	if err := postFile(src, "dir/big.bin", srv.URL); err != nil {
		t.Fatal(err)
	}
	if gotName != "big.bin" {
		t.Errorf("filename = %q, want %q", gotName, "big.bin")
	}
	if !bytes.Equal(gotData, data) {
		t.Errorf("received %d bytes, content mismatch", len(gotData))
	}
}

func TestPostFileReportsServerError(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.txt")
	os.WriteFile(src, []byte("x"), 0o644)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer srv.Close()
	if err := postFile(src, "a.txt", srv.URL); err == nil {
		t.Fatal("expected error for 400 response")
	}
}
