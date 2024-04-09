package util

import (
	"os"
	"path/filepath"
)

func GetDirFilePaths(dirPath string, relativeOnly bool) ([]string, error) {
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
