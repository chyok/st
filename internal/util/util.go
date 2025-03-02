package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SafeJoin joins the untrusted relative path name onto base and returns an
// error if the result would escape base (absolute paths, ".." segments, etc.).
func SafeJoin(base, name string) (string, error) {
	name = filepath.FromSlash(name)
	if name == "" || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", fmt.Errorf("invalid path %q", name)
	}
	joined := filepath.Join(base, name)
	rel, err := filepath.Rel(base, joined)
	if err != nil || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid path %q", name)
	}
	return joined, nil
}

func GetDirFilePaths(dirPath string, relativeOnly bool) ([]string, error) {
	var filePaths []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if relativeOnly {
				rel, err := filepath.Rel(dirPath, path)
				if err != nil {
					return err
				}
				relativePath := filepath.ToSlash(filepath.Join(filepath.Base(dirPath), rel))
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
