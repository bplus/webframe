// Command demo runs the webframe development/showcase server. It exists only
// for developing this library — webframe itself is a component module imported
// by consuming applications and ships no server.
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bplus/webframe/demo"
)

// main wires all demo routes — /static/ file serving, the showcase and traffic
// pages, and the HTMX/SSE API endpoints — onto a ServeMux and serves them on
// the hard-coded address :3333. It blocks forever; log.Fatal exits the process
// if the listener cannot bind (e.g. port already in use).
//
// @boundary HTTP - listens on :3333
// @side-effects binds a TCP port, writes startup line to stdout
func main() {
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Pages
	mux.HandleFunc("GET /{$}", demo.HandleShowcase)
	mux.HandleFunc("GET /traffic", demo.HandleTraffic)

	// HTMX demo endpoints
	mux.HandleFunc("GET /api/traffic/stream", demo.HandleTrafficStream)
	mux.HandleFunc("POST /api/click", demo.HandleClick)
	mux.HandleFunc("GET /api/search", demo.HandleSearch)
	mux.HandleFunc("POST /api/validate", demo.HandleValidate)
	mux.HandleFunc("POST /api/toast", demo.HandleToast)

	addr := ":3333"
	fmt.Printf("Webframe running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
