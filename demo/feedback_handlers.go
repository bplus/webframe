package demo

import (
	"log"
	"net/http"
	"strconv"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/authpage"
	"github.com/bplus/webframe/components/errorpage"
	"github.com/bplus/webframe/components/formfield"
)

// The handlers below are the demo's pages outside its own content: the
// three auth pages (no shell) and the three error pages (in the shell),
// plus the mux's catch-all. None of them keeps state; the sign-in form
// refuses every submission, since the demo has no accounts.

// signInFields are the sign-in form's two fields. Email is required and
// typed so the browser checks it; the password field never echoes a value.
var signInFields = []formfield.Field{
	{Name: "email", Label: "Email", Type: formfield.TypeEmail, Required: true, Placeholder: "you@example.com"},
	{Name: "password", Label: "Password", Type: formfield.TypePassword, Required: true},
}

// signInProps declares the sign-in page. AppName comes from the App in the
// request context (authpage.Respond fills it), Action posts back to
// /signin, and the footer links to the signed-out page so the three modes
// are reachable from one another.
func signInProps() authpage.Props {
	return authpage.Props{
		ID:      "signin",
		Action:  "/signin",
		Fields:  signInFields,
		Message: "Any email and password are refused: the demo has no accounts.",
		Submit:  action.Action{Label: "Sign in"},
	}
}

// HandleSignIn serves the sign-in page (GET /signin) and refuses every
// submission (POST /signin): the form re-renders with the typed email, an
// empty password and the error "Demo: sign-in is not wired", status 200 so
// an htmx submit swaps it. A real app would check the credentials, set its
// session and redirect (303) on success.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleSignIn(w http.ResponseWriter, r *http.Request) {
	p := signInProps()
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		p.Values = map[string]string{"email": r.FormValue("email")}
		p.Error = "Demo: sign-in is not wired"
	}
	if err := authpage.Respond(w, r, p); err != nil {
		log.Printf("demo: render sign in: %v", err)
	}
}

// HandleSignedOut serves the signed-out page (GET /signed-out): the
// message and a full-width "Sign in again" link to /signin.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleSignedOut(w http.ResponseWriter, r *http.Request) {
	p := authpage.Props{ID: "signed-out", Mode: authpage.ModeSignedOut, Action: "/signin"}
	if err := authpage.Respond(w, r, p); err != nil {
		log.Printf("demo: render signed out: %v", err)
	}
}

// HandleForbidden serves the forbidden page (GET /forbidden): the message
// and a full-width "Go home" link. It answers 200: the page is the demo of
// the component, not a refused request (the 403 error page is at
// /errors/403).
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleForbidden(w http.ResponseWriter, r *http.Request) {
	p := authpage.Props{ID: "forbidden", Mode: authpage.ModeForbidden}
	if err := authpage.Respond(w, r, p); err != nil {
		log.Printf("demo: render forbidden: %v", err)
	}
}

// demoRequestID is the request id the 500 page shows. A real app would
// take it from its logging middleware; the demo has none, so the id is
// fixed and the page is deterministic.
const demoRequestID = "req_7f3a9c2e41"

// HandleErrorPage serves the three error pages on demand (GET
// /errors/{code}) so the showcase can link to them: 404 and 403 with their
// default wording, 500 with the demo request id. Each answers with the
// status it shows; any other code is itself a 404.
//
// @boundary HTTP
// @side-effects writes HTML response with a non-200 status
func HandleErrorPage(w http.ResponseWriter, r *http.Request) {
	var err error
	switch code, _ := strconv.Atoi(r.PathValue("code")); code {
	case http.StatusForbidden:
		err = errorpage.Forbidden(w, r)
	case http.StatusInternalServerError:
		err = errorpage.Internal(w, r, demoRequestID)
	default:
		err = errorpage.NotFound(w, r)
	}
	if err != nil {
		log.Printf("demo: render error page %s: %v", r.PathValue("code"), err)
	}
}

// HandleNotFound is the mux's catch-all: every path no other pattern
// claims answers 404 with the error page, in the shell, with the nav still
// working.
//
// @boundary HTTP
// @side-effects writes HTML response with status 404
func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	if err := errorpage.NotFound(w, r); err != nil {
		log.Printf("demo: render not found %s: %v", r.URL.Path, err)
	}
}
