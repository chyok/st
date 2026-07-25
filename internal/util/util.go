package util

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
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

// GetDirFilePaths lists the regular files under dirPath. Symbolic links are
// kept only when they resolve to a regular file inside dirPath, because links
// leading elsewhere cannot be served.
func GetDirFilePaths(dirPath string, relativeOnly bool) ([]string, error) {
	root, err := os.OpenRoot(dirPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	fsys := root.FS()

	var filePaths []string
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			if d.Type()&fs.ModeSymlink == 0 {
				return nil // directory or special file
			}
			if info, err := fs.Stat(fsys, p); err != nil || !info.Mode().IsRegular() {
				return nil
			}
		}
		if relativeOnly {
			filePaths = append(filePaths, path.Join(filepath.Base(dirPath), p))
		} else {
			filePaths = append(filePaths, filepath.ToSlash(filepath.Join(dirPath, filepath.FromSlash(p))))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filePaths, nil
}

// SaveFile writes the contents of r to path and returns the path it actually
// used. Data goes to a hidden ".part" file in the same directory first and is
// moved into place only once fully written, so a failed transfer never leaves
// a truncated file behind. An existing file is never overwritten: if path is
// taken, a numbered name such as "a (1).txt" is used instead.
func SaveFile(path string, r io.Reader) (string, error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	tmp, err := os.CreateTemp(dir, "."+base+".*.part")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	_, err = io.Copy(tmp, r)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmpName)
		return "", err
	}

	final, err := reserveName(path)
	if err != nil {
		os.Remove(tmpName)
		return "", err
	}
	// The rename replaces the empty placeholder reserveName created.
	if err := os.Rename(tmpName, final); err != nil {
		os.Remove(tmpName)
		os.Remove(final)
		return "", err
	}
	return final, nil
}

// reserveName atomically creates an empty file at path, or at the first free
// "name (n).ext" variant of it, and returns the name it claimed.
func reserveName(path string) (string, error) {
	ext := filepath.Ext(path)
	if ext == filepath.Base(path) {
		ext = "" // dotfile such as ".bashrc": number it as ".bashrc (1)"
	}
	stem := strings.TrimSuffix(path, ext)
	for i := 0; i < 10000; i++ {
		candidate := path
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		f, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			f.Close()
			return candidate, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("no free file name for %q", path)
}
