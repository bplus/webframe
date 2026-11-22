package demo

import (
	"log"
	"net/http"

	"github.com/bplus/webframe/demo/pages"
)

// HandleLongText renders the long text page (GET /longtext), which no menu
// links to: it exists for `make ui-check`.
func HandleLongText(w http.ResponseWriter, r *http.Request) {
	if err := App.Render(w, r, "Long text", pages.LongText()); err != nil {
		log.Printf("demo: render long text: %v", err)
	}
}
