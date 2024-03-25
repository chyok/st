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
	"strings"

	"github.com/schollz/progressbar/v3"
)

// SendFile 发送单个文件
func SendFile(filePath string, url string) error {
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

// SendDirectory 发送目录
func SendDirectory(dirPath string, url string) error {
	var files []string
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

	fmt.Println("\nAll files in folder:")
	for _, file := range files {
		fmt.Println(file)
	}

	var confirm string
	fmt.Print("\nTransfer all files? [Y/N] ")
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
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
