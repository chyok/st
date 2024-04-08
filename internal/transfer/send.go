package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chyok/st/config"
	"github.com/chyok/st/web"
	"github.com/schollz/progressbar/v3"
)

func SendHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serveDownloadPage(w, r)
	case http.MethodPost:
		handleFilePaths(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFilePaths(w http.ResponseWriter, _ *http.Request) {
	currentPath := config.G.FilePath
	fileInfo, err := os.Stat(currentPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if fileInfo.IsDir() {
		files, err := getDirFilePaths(currentPath, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		PathInfo.Paths = files
		PathInfo.Type = "dir"
	} else {
		PathInfo.Paths = []string{filepath.Base(currentPath)}
		PathInfo.Type = "file"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PathInfo)
}

func serveDownloadPage(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	currentPath := config.G.FilePath

	basePath := filepath.Base(currentPath)

	if strings.HasSuffix(currentPath, "/") {
		basePath = filepath.Base(strings.TrimSuffix(currentPath, "/"))
	}

	if urlPath != "/" {
		currentPath = filepath.Join(currentPath, urlPath[1:])
		basePath = filepath.Join(currentPath, urlPath[1:])
	}
	fileInfo, err := os.Stat(currentPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		DeviceName  string
		IsDir       bool
		FileName    string
		CurrentPath string
		BasePath    string
		UrlPath     string
		Files       []os.DirEntry
	}{
		DeviceName:  config.G.DeviceName,
		BasePath:    basePath,
		CurrentPath: currentPath,
		UrlPath:     urlPath,
	}

	if fileInfo.IsDir() {
		data.IsDir = true

		files, err := os.ReadDir(currentPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Files = files
	} else {
		data.FileName = filepath.Base(currentPath)
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
}

func SendFile(filePath string, url string) error {
	filePath = filepath.ToSlash(filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file [%s] not exist", filePath)
		}
		return fmt.Errorf("file [%s] error: %w", filePath, err)
	}

	if fileInfo.IsDir() {
		return SendDirectory(filePath, url)
	}

	return postFile(filePath, path.Base(filePath), url)
}

func getDirFilePaths(dirPath string, relativeOnly bool) ([]string, error) {
	var filePaths []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if relativeOnly {
				fileName := filepath.Base(path)
				relativePath := filepath.ToSlash(filepath.Join(filepath.Base(dirPath), fileName))
				filePaths = append(filePaths, relativePath)
			} else {
				filePaths = append(filePaths, filepath.ToSlash(path))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filePaths, nil
}

func SendDirectory(dirPath string, url string) error {
	files, err := getDirFilePaths(dirPath, false)
	if err != nil {
		return err
	}

	fmt.Println("\nAll files in folder:")
	for _, file := range files {
		fmt.Println(file)
	}

	var confirm string
	fmt.Print("\nTransfer all files? [Y/N] ")
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Print("\nCancel send all files ")
		return nil
	}

	for _, file := range files {
		fileName, _ := filepath.Rel(dirPath, file)
		fileName = filepath.Join(filepath.Base(dirPath), fileName)
		err := postFile(file, fileName, url)
		if err != nil {
			return err
		}
	}
	fmt.Printf("Send folder %s success.\n", dirPath)
	return nil
}

func postFile(filePath string, filename string, url string) error {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.ToSlash(filename))
	if err != nil {
		return err
	}

	fileInfo, _ := file.Stat()
	bar := progressbar.DefaultBytes(
		fileInfo.Size(),
		fmt.Sprintf("Uploading [%s]", filename),
	)

	_, err = io.Copy(io.MultiWriter(part, bar), file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status code: %d", resp.StatusCode)
	}

	return nil
}
