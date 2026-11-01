// Package protocol defines what st instances exchange over HTTP.
package protocol

import (
	"net/url"
	"strings"
)

// Kinds of share in a FileList.
const (
	File = "file"
	Dir  = "dir"
)

// FileList is a sender's reply to "POST /": what it shares. Each path can be
// fetched from "/download/<path>".
type FileList struct {
	Type  string   `json:"type"` // File or Dir
	Paths []string `json:"paths"`
}

// EscapePath percent-encodes each segment of a slash-separated path, so names
// containing "#", "?", "%" or spaces stay part of the path in a URL.
func EscapePath(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}
