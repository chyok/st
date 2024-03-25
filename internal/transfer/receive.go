package transfer

import (
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

// ReceiveHandler 处理文件上传请求
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

func ReceiveFile(remoteAddr, savePath string) error {
	resp, err := http.Get(fmt.Sprintf("http://%s/upload", remoteAddr))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	filename := filepath.Base(remoteAddr)
	if savePath != "" {
		filename = filepath.Join(savePath, filename)
	}

	fmt.Printf("Downloading [%s]...\n", filename)
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("[✅] Download [%s] Success.\n", filename)
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
