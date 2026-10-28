// Package assets serves webframe's embedded static/ directory.
package assets

import (
	"io/fs"
	"net/http"

	"github.com/bplus/webframe"
	"github.com/bplus/webframe/utils"
)

// FS returns the embedded static/ tree rooted at static/.
func FS() fs.FS {
	sub, err := fs.Sub(webframe.Static, "static")
	if err != nil {
		panic(err) // impossible: literal valid path
	}
	return sub
}

// Handler serves the embedded assets under the /static/ URL prefix.
func Handler() http.Handler {
	return http.StripPrefix("/static/", http.FileServerFS(FS()))
}

// Mount registers Handler on mux at GET /static/.
func Mount(mux *http.ServeMux) {
	mux.Handle("GET /static/", Handler())
}

// URL returns a cache-busted URL for an asset path.
func URL(path string) string {
	return utils.ScriptURL(path)
}
