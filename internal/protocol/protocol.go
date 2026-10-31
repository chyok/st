// Package protocol defines what st instances exchange over HTTP.
package protocol

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
