package web

import "embed"

//go:embed all:static
//go:embed index.html login.html
var FS embed.FS
