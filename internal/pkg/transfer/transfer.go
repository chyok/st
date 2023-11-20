package transfer

import (
	"bytes"
	"fmt"
	tmp "html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/pkg/discovery"
	"github.com/chyok/st/template"
	"github.com/schollz/progressbar/v3"
)

func ReceiveFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl, err := tmp.New("index").Parse(template.UploadPage)
		if err != nil {
			w.Write(([]byte(fmt.Sprintf("create template failed, err: %s", err))))
		}
		tmpl.Execute(w, config.G.DeviceName)
	} else {
		file, header, err := r.FormFile("file")
		fmt.Printf("Downloading [%s]...\r", header.Filename)
		if err != nil {
			http.Error(w, "Could not get file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", header.Filename)
			return
		}
		defer file.Close()

		out, err := os.Create(header.Filename)

		if err != nil {
			http.Error(w, "could not create file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", header.Filename)
			return
		}
		defer out.Close()
		_, err = io.Copy(out, file)
		if err != nil {
			http.Error(w, "could not write file", http.StatusInternalServerError)
			fmt.Printf("[X] Download [%s] Failed. \n", header.Filename)
			return
		}
		fmt.Printf("[√] Download [%s] Success.  \n", header.Filename)
	}
}

func SendFile(filePath string) error {
	count := 0
LOOP:
	for {
		select {
		case i := <-discovery.DiscoveredIPChan:
			fmt.Println("Send to device [" + i[0] + "]")
			if !postFileByHttp(filePath, "http://"+i[1]+":"+config.G.Port) {
				fmt.Println("Send failed")
			} else {
				count += 1
			}

		case <-time.After(3 * time.Second):
			if count > 0 {
				fmt.Println("Send to " + strconv.Itoa(count) + " device success")
			} else {
				fmt.Println("No device found")
			}
			break LOOP
		}

	}

	return nil
}

func postFileByHttp(filePath string, url string) bool {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, open_err := os.Open(filePath)

	if open_err != nil {
		return false
	}

	defer file.Close()

	part, _ := writer.CreateFormFile("file", filepath.Base(filePath))
	fileInfo, _ := file.Stat()

	bar := progressbar.DefaultBytes(
		fileInfo.Size(),
		"Uploading",
	)
	io.Copy(io.MultiWriter(part, bar), file)
	_, errFile1 := io.Copy(part, file)

	if errFile1 != nil {
		return false
	}
	err := writer.Close()

	if err != nil {
		return false
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)

	if err != nil {
		return false
	}

	defer res.Body.Close()

	return res.StatusCode == 200
}
