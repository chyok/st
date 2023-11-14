package template

import "embed"

//go:embed css
var CssFs embed.FS

//go:embed upload.tmpl
var UploadPage string
