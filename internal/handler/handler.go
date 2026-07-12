package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"strings"

	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/util"
	"github.com/chyok/st/web"
)

func ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// serve upload page for receive
		tmpl, err := template.New("index").Parse(web.UploadPage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, config.G.DeviceName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		// receive file and save
		if err := receiveUpload(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// receiveUpload streams every "file" part of a multipart request straight to
// disk. Unlike r.FormFile it does not spool large uploads to a temporary file
// first, so each upload is written to disk only once.
func receiveUpload(r *http.Request) error {
	mr, err := r.MultipartReader()
	if err != nil {
		return err
	}
	received := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		// part.FileName() drops directories; parse the header ourselves to
		// keep the folder structure of browser folder uploads.
		_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			return err
		}
		filename, err := util.SafeJoin(".", params["filename"])
		if err != nil {
			return err
		}

		fmt.Printf("Downloading [%s]...\n", filename)
		saved, err := util.SaveFile(filename, part)
		part.Close()
		if err != nil {
			return err
		}
		fmt.Printf("[✅] Download [%s] Success.\n", saved)
		received++
	}
	if received == 0 {
		return errors.New("no file in request")
	}
	return nil
}

func FileServerHandler(w http.ResponseWriter, r *http.Request) {
	currentPath := config.G.FilePath

	fileInfo, err := os.Stat(currentPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	basePath := filepath.Base(currentPath)
	if fileInfo.IsDir() {
		prefix := "/download/" + basePath
		if r.URL.Path != prefix && !strings.HasPrefix(r.URL.Path, prefix+"/") {
			http.NotFound(w, r)
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, prefix)
		fullPath := filepath.Join(currentPath, filepath.FromSlash(rel))
		http.ServeFile(w, r, fullPath)
	} else {
		http.ServeFile(w, r, currentPath)
	}
}

func SendHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// serve download page for send
		if r.URL.Path != "/" {
			// A single shared file has no sub-paths.
			if info, err := os.Stat(config.G.FilePath); err == nil && !info.IsDir() {
				http.NotFound(w, r)
				return
			}
		}
		realFilePath := filepath.Join(config.G.FilePath, r.URL.Path[1:])
		// downloadPath ends up in URLs, so always use forward slashes.
		downloadPath := path.Join(filepath.Base(config.G.FilePath), r.URL.Path[1:])
		fileInfo, err := os.Stat(realFilePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			DeviceName   string
			IsDir        bool
			FileName     string
			DownloadPath string
			UrlPath      string
			Files        []os.DirEntry
		}{
			DeviceName:   config.G.DeviceName,
			DownloadPath: downloadPath,
			UrlPath:      strings.TrimSuffix(r.URL.Path, "/"),
		}

		if fileInfo.IsDir() {
			data.IsDir = true

			files, err := os.ReadDir(realFilePath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			data.Files = files
		} else {
			data.FileName = filepath.Base(realFilePath)
		}

		tmpl, err := template.New("download").Parse(web.DownloadPage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		// return file or folder information in JSON format for convenient send to the recipient
		currentPath := config.G.FilePath
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
			files, err := util.GetDirFilePaths(currentPath, true)
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
		json.NewEncoder(w).Encode(pathInfo)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
