package listpage

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/datatable"
)

// TestRespond_RejectsInvalidAction: an action that contradicts itself
// (here a Form on a plain link) makes Respond return action.ErrInvalid
// and write nothing.
func TestRespond_RejectsInvalidAction(t *testing.T) {
	bad := action.Action{Label: "Open", Href: "/x", Form: "f"}
	var err error
	app := &appshell.App{Name: "Demo"}
	h := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err = Respond(w, r, "T", Props{Toolbar: []action.Action{bad}})
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

// TestRespondResults_RejectsInvalidRowAction: a row action with Submit and
// HxDelete makes RespondResults return action.ErrInvalid, writing nothing.
func TestRespondResults_RejectsInvalidRowAction(t *testing.T) {
	rec := httptest.NewRecorder()
	p := Props{Rows: []datatable.Row{{ID: "1", Actions: []action.Action{{Label: "Del", Submit: true, HxDelete: "/1"}}}}}
	err := RespondResults(rec, httptest.NewRequest("DELETE", "/1", nil), p)
	if !errors.Is(err, action.ErrInvalid) || rec.Body.Len() != 0 || rec.Header().Get("HX-Replace-Url") != "" {
		t.Errorf("err = %v, body %d bytes, HX-Replace-Url %q", err, rec.Body.Len(), rec.Header().Get("HX-Replace-Url"))
	}
}
