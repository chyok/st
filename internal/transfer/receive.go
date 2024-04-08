package transfer

import (
	"encoding/json"
	"fmt"
	tmp "html/template"

	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chyok/st/config"
	"github.com/chyok/st/web"
)

var PathInfo struct {
	Type  string   `json:"type"`
	Paths []string `json:"paths"`
}

func ReceiveHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serveUploadPage(w, r)
	case http.MethodPost:
		uploadFile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
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

func ReceiveFile(remoteAddr string) error {
	resp, err := http.Post(fmt.Sprintf("http://%s/", remoteAddr), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get file info: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&PathInfo); err != nil {
		return err
	}

	if PathInfo.Type == "dir" {
		fmt.Printf("Found directory with %d paths:\n", len(PathInfo.Paths))
		for _, path := range PathInfo.Paths {
			fmt.Printf("- %s\n", path)
		}
		fmt.Print("Do you want to download the entire directory? [y/n] ")
		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil || (confirm != "y" && confirm != "Y") {
			return nil
		}
		for _, path := range PathInfo.Paths {
			if err := downloadFile(remoteAddr, path); err != nil {
				return err
			}
		}
	} else {
		if len(PathInfo.Paths) != 1 {
			return fmt.Errorf("unexpected number of paths: %d", len(PathInfo.Paths))
		}
		if err := downloadFile(remoteAddr, PathInfo.Paths[0]); err != nil {
			return err
		}
	}
	return nil
}

func downloadFile(remoteAddr, path string) error {
	path = filepath.ToSlash(path)
	resp, err := http.Get(fmt.Sprintf("http://%s/download/%s", remoteAddr, path))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	fmt.Printf("Downloading [%s]...\n", path)
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("[✅] Download [%s] Success.\n", path)
	return nil
}

func serveUploadPage(w http.ResponseWriter, _ *http.Request) {
	tmpl, err := tmp.New("index").Parse(web.UploadPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, config.G.DeviceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
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
}
