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

func ReceiveFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := tmp.New("index").Parse(web.UploadPage)
		if err != nil {
			w.Write(([]byte(fmt.Sprintf("create template failed, err: %s", err))))
		}
		tmpl.Execute(w, config.G.DeviceName)
	} else {
		file, header, err := r.FormFile("file")
		_, params, _ := mime.ParseMediaType(header.Header.Get("Content-Disposition"))
		filename := params["filename"]
		fmt.Printf("Downloading [%s]...\r", filename)
		if err != nil {
			http.Error(w, "Could not get file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", filename)
			return
		}
		defer file.Close()

		filename = filepath.FromSlash(filename)
		dirPath := filepath.Dir(filename)
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			http.Error(w, "could not create file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", filename)
			return
		}

		out, err := os.Create(filename)

		if err != nil {
			http.Error(w, "could not create file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", filename)
			return
		}
		defer out.Close()
		_, err = io.Copy(out, file)
		if err != nil {
			http.Error(w, "could not write file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", filename)
			return
		}
		fmt.Printf("[√] Download [%s] Success.  \n", filename)
	}
}
