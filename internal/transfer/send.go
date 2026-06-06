package transfer

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chyok/st/internal/util"
	"github.com/schollz/progressbar/v3"
)

func SendFiles(filePath string, url string) error {
	filePath = filepath.ToSlash(filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file [%s] not exist", filePath)
		}
		return fmt.Errorf("file [%s] error: %w", filePath, err)
	}

	if fileInfo.IsDir() {
		return postDirectory(filePath, url)
	}

	return postFile(filePath, path.Base(filePath), url)
}

func postDirectory(dirPath string, url string) error {
	files, err := util.GetDirFilePaths(dirPath, false)
	if err != nil {
		return err
	}

	var details strings.Builder
	details.WriteString("\nAll files in folder:\n")
	for _, file := range files {
		details.WriteString(file + "\n")
	}
	if !confirm(details.String(), "\nTransfer all files? [Y/N] ") {
		fmt.Println("\nCancel send all files")
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
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	bar := progressbar.DefaultBytes(
		fileInfo.Size(),
		fmt.Sprintf("Uploading [%s]", filename),
	)

	// Stream the multipart body through a pipe instead of buffering the whole
	// file in memory; the progress bar now tracks bytes actually sent.
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		part, err := writer.CreateFormFile("file", filepath.ToSlash(filename))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(io.MultiWriter(part, bar), file); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(writer.Close())
	}()

	req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		pr.Close()
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
