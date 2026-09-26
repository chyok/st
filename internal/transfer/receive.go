package transfer

import (
	"encoding/json"
	"fmt"

	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/chyok/st/internal/util"
)

// Client transfers files to and from other st instances found by discovery.
type Client struct {
	// Dir is where received files are saved.
	Dir string
	// Confirm asks the user whether to go ahead; nil asks on stdin.
	Confirm func(details, question string) bool
}

func (c *Client) confirm(details, question string) bool {
	if c.Confirm != nil {
		return c.Confirm(details, question)
	}
	return confirm(details, question)
}

// ReceiveFiles fetches everything the st sender at remoteAddr is sharing.
func (c *Client) ReceiveFiles(remoteAddr string) error {

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
		var details strings.Builder
		fmt.Fprintf(&details, "\nFound sender at %s offering a directory with %d files:\n", remoteAddr, len(pathInfo.Paths))
		for _, path := range pathInfo.Paths {
			fmt.Fprintf(&details, "- %s\n", path)
		}
		if !c.confirm(details.String(), "Do you want to download the entire directory? [y/n] ") {
			return nil
		}
		for _, path := range pathInfo.Paths {
			if err := c.downloadFile(remoteAddr, path); err != nil {
				return err
			}
		}
	} else {
		if len(pathInfo.Paths) != 1 {
			return fmt.Errorf("unexpected number of paths: %d", len(pathInfo.Paths))
		}
		details := fmt.Sprintf("\nFound sender at %s offering file: %s\n", remoteAddr, pathInfo.Paths[0])
		if !c.confirm(details, "Download it? [y/n] ") {
			return nil
		}
		if err := c.downloadFile(remoteAddr, pathInfo.Paths[0]); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) downloadFile(remoteAddr, path string) error {
	path = filepath.ToSlash(path)
	localPath, err := util.SafeJoin(c.Dir, path)
	if err != nil {
		return err
	}

	// Escape each segment so names containing "#", "?", "%" or spaces survive.
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	resp, err := http.Get(fmt.Sprintf("http://%s/download/%s", remoteAddr, strings.Join(segments, "/")))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	fmt.Printf("Downloading [%s]...\n", path)
	saved, err := util.SaveFile(localPath, resp.Body)
	if err != nil {
		return err
	}
	fmt.Printf("[✅] Download [%s] Success.\n", saved)
	return nil
}
