package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime"
	"strings"

	"net/http"
	"os"
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
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		_, params, err := mime.ParseMediaType(header.Header.Get("Content-Disposition"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filename := filepath.FromSlash(params["filename"])

		fmt.Printf("Downloading [%s]...\n", filename)
		dirPath := filepath.Dir(filename)
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		out, err := os.Create(filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("[✅] Download [%s] Success.\n", filename)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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
		path := r.URL.Path[len("/download/"+basePath):]
		fullPath := filepath.Join(currentPath, path)
		http.ServeFile(w, r, fullPath)
	} else {
		http.ServeFile(w, r, currentPath)
	}
}

func SendHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// serve download page for send
		realFilePath := filepath.Join(config.G.FilePath, r.URL.Path[1:])
		downloadPath := filepath.Join(filepath.Base(config.G.FilePath), r.URL.Path[1:])
		fileInfo, err := os.Stat(realFilePath)
		if err != nil {
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
