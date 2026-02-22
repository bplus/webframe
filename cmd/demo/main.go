package main

import (
	"fmt"
	"log"
	"net/http"

	"webframe/demo"
)

func main() {
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Pages
	mux.HandleFunc("GET /{$}", demo.HandleShowcase)

	// HTMX demo endpoints
	mux.HandleFunc("POST /api/click", demo.HandleClick)
	mux.HandleFunc("GET /api/search", demo.HandleSearch)
	mux.HandleFunc("POST /api/validate", demo.HandleValidate)
	mux.HandleFunc("POST /api/toast", demo.HandleToast)

	addr := ":3333"
	fmt.Printf("Webframe running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
