package receipt

import (
	"embed"
	"io/fs"
)

//go:embed static/index.html
var indexHTML []byte

//go:embed static/app.css
var appCSS []byte

//go:embed static/app.js
var appJS []byte

//go:embed static/controllers/*.js
var controllersFS embed.FS

// getControllersFS returns the embedded controllers filesystem
func getControllersFS() fs.FS {
	fsys, err := fs.Sub(controllersFS, "static/controllers")
	if err != nil {
		panic(err)
	}
	return fsys
}
