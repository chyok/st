package web

import "embed"

//go:embed css
var CssFs embed.FS

//go:embed upload.tmpl
var UploadPage string

//go:embed download.tmpl
var DownloadPage string
