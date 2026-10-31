package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/url"
	"strings"

	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/chyok/st/internal/util"
	"github.com/chyok/st/web"
)

var (
	uploadTmpl   = template.Must(template.New("upload").Parse(web.UploadPage))
	downloadTmpl = template.Must(template.New("download").
			Funcs(template.FuncMap{"pathEscape": pathEscape}).
			Parse(web.DownloadPage))
)

// pathEscape percent-encodes each segment of a slash-separated path, so names
// with "#" or "?" stay part of the path in links.
func pathEscape(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

// Receiver serves the upload page and saves uploaded files under Dir.
type Receiver struct {
	DeviceName string
	Dir        string
}

// Routes returns the HTTP handler for a receiving st instance.
func (h *Receiver) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.page)
	mux.HandleFunc("POST /", h.upload)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.CssFs))))
	return mux
}

func (h *Receiver) page(w http.ResponseWriter, r *http.Request) {
	if err := uploadTmpl.Execute(w, h.DeviceName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Receiver) upload(w http.ResponseWriter, r *http.Request) {
	if status, err := h.receiveUpload(r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

// receiveUpload streams every "file" part of a multipart request straight to
// disk. Unlike r.FormFile it does not spool large uploads to a temporary file
// first, so each upload is written to disk only once. On failure it returns
// the HTTP status to report: 400 for a bad request, 500 if saving failed.
func (h *Receiver) receiveUpload(r *http.Request) (int, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return http.StatusBadRequest, err
	}
	received := 0
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return http.StatusBadRequest, err
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		// part.FileName() drops directories; parse the header ourselves to
		// keep the folder structure of browser folder uploads.
		_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			return http.StatusBadRequest, err
		}
		filename, err := util.SafeJoin(h.Dir, params["filename"])
		if err != nil {
			return http.StatusBadRequest, err
		}

		fmt.Printf("Downloading [%s]...\n", filename)
		saved, err := util.SaveFile(filename, part)
		part.Close()
		if err != nil {
			return http.StatusInternalServerError, err
		}
		fmt.Printf("[✅] Download [%s] Success.\n", saved)
		received++
	}
	if received == 0 {
		return http.StatusBadRequest, errors.New("no file in request")
	}
	return http.StatusOK, nil
}

// Sender serves Path, a file or a folder, for download.
type Sender struct {
	DeviceName string
	Path       string
}

// Routes returns the HTTP handler for a sending st instance.
func (h *Sender) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.page)
	mux.HandleFunc("POST /", h.info)
	mux.HandleFunc("GET /download/", h.download)
	return mux
}

func (h *Sender) download(w http.ResponseWriter, r *http.Request) {
	fileInfo, err := os.Stat(h.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !fileInfo.IsDir() {
		http.ServeFile(w, r, h.Path)
		return
	}

	prefix := "/download/" + filepath.Base(h.Path)
	if r.URL.Path != prefix && !strings.HasPrefix(r.URL.Path, prefix+"/") {
		http.NotFound(w, r)
		return
	}
	// os.Root refuses any path, including via symlinks, that leaves the share.
	root, err := os.OpenRoot(h.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer root.Close()
	http.ServeFileFS(w, r, root.FS(), rootName(strings.TrimPrefix(r.URL.Path, prefix)))
}

// rootName turns a URL path into a name for an os.Root filesystem.
func rootName(urlPath string) string {
	if name := strings.Trim(urlPath, "/"); name != "" {
		return name
	}
	return "."
}

// listDir reads a directory of an os.Root filesystem. Symlinks are shown as
// what they point to, and dropped when they lead outside the root.
func listDir(fsys fs.FS, name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		return nil, err
	}
	kept := entries[:0]
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink != 0 {
			info, err := fs.Stat(fsys, path.Join(name, e.Name()))
			if err != nil {
				continue
			}
			e = fs.FileInfoToDirEntry(info)
		}
		kept = append(kept, e)
	}
	return kept, nil
}

// page renders the download page for the shared file or a folder in it.
func (h *Sender) page(w http.ResponseWriter, r *http.Request) {
	sharedInfo, err := os.Stat(h.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.URL.Path != "/" && !sharedInfo.IsDir() {
		// A single shared file has no sub-paths.
		http.NotFound(w, r)
		return
	}
	// downloadPath ends up in URLs, so always use forward slashes.
	downloadPath := path.Join(filepath.Base(h.Path), r.URL.Path[1:])

	fileInfo := sharedInfo
	var fsys fs.FS
	name := rootName(r.URL.Path)
	if sharedInfo.IsDir() {
		root, err := os.OpenRoot(h.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer root.Close()
		fsys = root.FS()
		fileInfo, err = fs.Stat(fsys, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		DeviceName   string
		IsDir        bool
		FileName     string
		DownloadPath string
		UrlPath      string
		Files        []os.DirEntry
	}{
		DeviceName:   h.DeviceName,
		DownloadPath: downloadPath,
		UrlPath:      strings.TrimSuffix(r.URL.Path, "/"),
	}

	if fileInfo.IsDir() {
		data.IsDir = true
		files, err := listDir(fsys, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Files = files
	} else {
		data.FileName = fileInfo.Name()
	}

	if err := downloadTmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// info lists the shared paths as JSON so a receiving st can fetch them all.
func (h *Sender) info(w http.ResponseWriter, r *http.Request) {
	currentPath := h.Path
	fileInfo, err := os.Stat(currentPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var pathInfo struct {
		Type  string   `json:"type"`
		Paths []string `json:"paths"`
	}

	if fileInfo.IsDir() {
		files, err := util.GetDirFilePaths(currentPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pathInfo.Paths = files
		pathInfo.Type = "dir"
	} else {
		pathInfo.Paths = []string{filepath.Base(currentPath)}
		pathInfo.Type = "file"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(pathInfo); err != nil {
		fmt.Printf("Failed to send file list: %v\n", err)
	}
}
