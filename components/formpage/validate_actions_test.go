package formpage

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
)

// TestRespond_RejectsInvalidAction: an action that contradicts itself
// (here a Form on a plain link) makes Respond return action.ErrInvalid
// and write nothing.
func TestRespond_RejectsInvalidAction(t *testing.T) {
	bad := action.Action{Label: "Open", Href: "/x", Form: "f"}
	var err error
	app := &appshell.App{Name: "Demo"}
	h := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = Respond(w, r, "T", Props{Cancel: &bad})
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !errors.Is(err, action.ErrInvalid) {
		t.Fatalf("Respond err = %v, want action.ErrInvalid", err)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("Respond wrote %d bytes for an invalid declaration", rec.Body.Len())
	}
}
