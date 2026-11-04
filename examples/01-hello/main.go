// Command hello is tutorial step 1: one page in the webframe shell.
package main

import (
	"log"
	"net/http"

	"github.com/bplus/webframe/assets"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/prose"
	"github.com/bplus/webframe/compose"
)

var app = &appshell.App{Name: "Hello"}

func home(w http.ResponseWriter, r *http.Request) {
	page := compose.Join(
		pageheader.PageHeader(pageheader.Props{Title: "Hello, webframe"}),
		prose.Prose(prose.Props{Text: "This page has no HTML in it. Every piece is a component."}),
	)
	if err := app.Render(w, r, "Home", page); err != nil {
		log.Print(err)
	}
}

func routes() http.Handler {
	mux := http.NewServeMux()
	assets.Mount(mux)
	mux.HandleFunc("GET /{$}", home)
	return app.Middleware(mux)
}

func main() {
	log.Print("listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", routes()))
}
