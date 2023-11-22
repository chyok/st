package transfer

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chyok/st/config"
	"github.com/chyok/st/internal/pkg/discovery"
	"github.com/schollz/progressbar/v3"
)

func Send(filePath string, url string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("File [%s] not exist\n", filePath)
			return err
		} else {
			fmt.Printf("File [%s] error: %s\n", filePath, err)
			return err
		}
	}
	if fileInfo.IsDir() {
		sendDir(filePath, url)
	} else {
		if !postFileByHttp(filePath, path.Base(filePath), url) {
			return fmt.Errorf("Send failed")
		}
	}
	return nil
}

func SendToAll(filePath string) error {
	count := 0
LOOP:
	for {
		select {
		case i := <-discovery.DiscoveredIPChan:
			fmt.Println("Send to device [" + i[0] + "]")
			err := Send(filePath, "http://"+i[1]+":"+config.G.Port)
			if err != nil {
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

func sendDir(dirPath string, url string) error {
	var files []string
	var confirm string
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		fmt.Printf("Folder %s is empty.\n", dirPath)
		return nil
	}
	fmt.Println("\nAll file in folder:")
	for _, file := range files {
		fmt.Println(file)
	}
	fmt.Println("\nTransfer all files? [Y/N]")
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) == "y" {
		for _, file := range files {
			fileName, _ := filepath.Rel(dirPath, file)
			fileName = filepath.Join(filepath.Base(dirPath), fileName)
			postFileByHttp(file, fileName, url)
		}
	}
	return nil
}

func postFileByHttp(filePath string, filename string, url string) bool {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	file, open_err := os.Open(filePath)

	if open_err != nil {
		return false
	}

	defer file.Close()

	part, _ := writer.CreateFormFile("file", filepath.ToSlash(filename))
	fileInfo, _ := file.Stat()

	bar := progressbar.DefaultBytes(
		fileInfo.Size(),
		fmt.Sprintf("Uploading [%s]", filename),
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
