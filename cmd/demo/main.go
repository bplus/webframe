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

// main serves demo.Handler (every demo route, wrapped in the shell
// middleware; the table is in demo.Routes) on the hard-coded address :3333.
// It blocks forever; log.Fatal exits the process if the listener cannot
// bind (e.g. port already in use).
func main() {
	addr := ":3333"
	fmt.Printf("Webframe running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, demo.Handler()))
}
