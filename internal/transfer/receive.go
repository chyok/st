package transfer

import (
	"encoding/json"
	"fmt"

	"io"
	"net/http"
	"os"
	"path/filepath"
)

func ReceiveFiles(remoteAddr string) error {

	resp, err := http.Post(fmt.Sprintf("http://%s/", remoteAddr), "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get file info: %s", resp.Status)
	}

	var pathInfo struct {
		Type  string   `json:"type"`
		Paths []string `json:"paths"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pathInfo); err != nil {
		return err
	}

	if pathInfo.Type == "dir" {
		fmt.Printf("Found directory with %d paths:\n", len(pathInfo.Paths))
		for _, path := range pathInfo.Paths {
			fmt.Printf("- %s\n", path)
		}
		fmt.Print("Do you want to download the entire directory? [y/n] ")
		var confirm string
		if _, err := fmt.Scanln(&confirm); err != nil || (confirm != "y" && confirm != "Y") {
			return nil
		}
		for _, path := range pathInfo.Paths {
			if err := downloadFile(remoteAddr, path); err != nil {
				return err
			}
		}
	} else {
		if len(pathInfo.Paths) != 1 {
			return fmt.Errorf("unexpected number of paths: %d", len(pathInfo.Paths))
		}
		if err := downloadFile(remoteAddr, pathInfo.Paths[0]); err != nil {
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
