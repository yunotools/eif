package web

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var staticFiles embed.FS

func StaticFS() fs.FS {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return staticFS
}
