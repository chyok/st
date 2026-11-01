package transfer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/chyok/st/internal/protocol"
	"github.com/chyok/st/internal/util"
	"github.com/schollz/progressbar/v3"
)

// Client fetches files from other st instances found by discovery.
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

	var list protocol.FileList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}

	if list.Type == protocol.Dir {
		var details strings.Builder
		fmt.Fprintf(&details, "\nFound sender at %s offering a directory with %d files:\n", remoteAddr, len(list.Paths))
		for _, path := range list.Paths {
			fmt.Fprintf(&details, "- %s\n", path)
		}
		if !c.confirm(details.String(), "Do you want to download the entire directory? [y/n] ") {
			return nil
		}
		for _, path := range list.Paths {
			if err := c.downloadFile(remoteAddr, path); err != nil {
				return err
			}
		}
	} else {
		if len(list.Paths) != 1 {
			return fmt.Errorf("unexpected number of paths: %d", len(list.Paths))
		}
		details := fmt.Sprintf("\nFound sender at %s offering file: %s\n", remoteAddr, list.Paths[0])
		if !c.confirm(details, "Download it? [y/n] ") {
			return nil
		}
		if err := c.downloadFile(remoteAddr, list.Paths[0]); err != nil {
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

	resp, err := http.Get(fmt.Sprintf("http://%s/download/%s", remoteAddr, protocol.EscapePath(path)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	bar := progressbar.DefaultBytes(resp.ContentLength, fmt.Sprintf("Downloading [%s]", path))
	saved, err := util.SaveFile(localPath, io.TeeReader(resp.Body, bar))
	if err != nil {
		return err
	}
	fmt.Printf("[✅] Download [%s] Success.\n", saved)
	return nil
}
