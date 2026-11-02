// Package errorpage renders 404, 403, 405 and 500 pages as one component
// and answers a request with the right status code, with or without an
// appshell.App in the request context.
//
// # 404, 405 and Go's ServeMux
//
// Go's http.ServeMux (1.22 and later, method patterns such as "GET /items")
// answers a method mismatch itself: a plain-text "Method Not Allowed" with
// an Allow header, written before any handler runs, so a handler can never
// turn it into this page. ServeMux has no hook for it, and its not-found
// answer is plain text too. Wrap the mux in Handler and both become the
// styled pages:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("GET /items", listItems)
//	mux.HandleFunc("POST /items", createItem)
//	http.ListenAndServe(addr, app.Middleware(errorpage.Handler(mux)))
//
// Handler sits inside App.Middleware so the pages render in the shell. It
// replaces any "/" catch-all route the app registered for its 404 page:
// with a catch-all in the table every path has a pattern, the mux never
// reports "no route", and the 405 branch never fires (a wrong method on a
// real route would fall into the catch-all and render as a 404). Register
// the real routes only and let Handler answer the rest. An app that is
// content with the mux's plain answers (an API, a form that is only ever
// posted by its own page) needs none of this.
//
// # Panics
//
// appshell.Middleware recovers a panic in a handler and calls App.OnPanic;
// its default is a bare, unstyled 500 document. Assign PanicHandler so the
// panic answers with this package's 500 page (in the shell, showing the
// request id the log line carries):
//
//	app.OnPanic = errorpage.PanicHandler
package errorpage

import (
	"log"
	"net/http"

	"github.com/bplus/webframe/components/appshell"
)

// Respond writes the error page for r with p.Code as the status (500 when
// 0). With an appshell.App in the request context (the handler sits under
// App.Middleware) it renders inside the shell through App.Render, so the
// nav still works; without one (a 500 from a middleware that runs before
// App.Middleware) it renders a bare document through appshell.RenderBare.
// Either way a plain htmx request gets the body only, per Render's rule.
//
// Headers go out in the only order net/http allows: appshell.SetHeaders
// (Content-Type, Vary), then WriteHeader(code), then the render, which
// finds the status already written and leaves it. A render error is
// returned, not written: the status line has already gone out.
func Respond(w http.ResponseWriter, r *http.Request, p Props) error {
	status := p.Code
	if status == 0 {
		status = http.StatusInternalServerError
	}
	title, _ := defaults(p)
	appshell.SetHeaders(w)
	if app, ok := appshell.FromContext(r.Context()); ok {
		// RenderStatus runs App.Flash and sets the headers before writing
		// the status, so a pending flash is still cleared.
		return app.RenderStatus(w, r, status, title, ErrorPage(p))
	}
	// No App (and no flash): no app name, so the document title is the
	// page title alone.
	appshell.SetHeaders(w)
	w.WriteHeader(status)
	return appshell.RenderBare(w, r, "", title, ErrorPage(p))
}

// NotFound answers r with the default 404 page.
func NotFound(w http.ResponseWriter, r *http.Request) error {
	return Respond(w, r, Props{Code: http.StatusNotFound})
}

// Forbidden answers r with the default 403 page.
func Forbidden(w http.ResponseWriter, r *http.Request) error {
	return Respond(w, r, Props{Code: http.StatusForbidden})
}

// MethodNotAllowed answers r with the default 405 page ("Not allowed
// here"). It does not set the Allow header; the caller knows the methods.
// Handler, the ServeMux wrapper, collects them and sets it before calling
// this.
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) error {
	return Respond(w, r, Props{Code: http.StatusMethodNotAllowed})
}

// Internal answers r with the default 500 page showing requestID (when
// non-empty) so the user can quote it.
func Internal(w http.ResponseWriter, r *http.Request, requestID string) error {
	return Respond(w, r, Props{Code: http.StatusInternalServerError, RequestID: requestID})
}

// PanicHandler is an appshell.App.OnPanic that answers a recovered panic
// with Internal: the 500 page showing requestID, in the shell when the App
// is in r's context. appshell.Middleware calls it only when the handler had
// not yet written a status or any body, so the status line is still free.
// The panic value is already logged by Middleware; a render error is
// logged here because nothing else can be done with it at this point.
//
//	app.OnPanic = errorpage.PanicHandler
func PanicHandler(w http.ResponseWriter, r *http.Request, requestID string, _ any) {
	if err := Internal(w, r, requestID); err != nil {
		log.Printf("errorpage: render 500 for %s %s (request %s): %v", r.Method, r.URL.Path, requestID, err)
	}
}

// Handler wraps mux so its plain-text 404 and 405 answers become the
// styled pages. A request the mux has a pattern for (a real route, or the
// mux's own trailing-slash redirect) goes to the mux untouched.
//
// Otherwise ServeMux.Handler has returned an empty pattern, and with it
// one of the mux's own three answers: its 405 handler (the path matches
// some pattern, but not for this method; it carries the Allow header the
// mux computed), its 404 handler, or a redirect to the cleaned path. None
// of these is an app handler, so Handler runs it into a header-only
// recorder to read the verdict: a 405 becomes MethodNotAllowed with the
// mux's own Allow header copied over, a 404 becomes NotFound, and anything
// else (the redirect) goes to the mux. Asking the mux instead of probing
// the methods by hand keeps Handler in step with ServeMux's rules in
// whatever Go version the app is built with (go1.27, for one, counts a
// path whose trailing-slash form has a route as a 405 for other methods,
// not a 404; a hand-rolled probe would have to know that).
//
// Handler never runs an app handler on the mux's behalf. Both pages
// render through Respond: in the shell when the handler chain has an
// appshell.App (wrap Handler in App.Middleware), bare otherwise.
func Handler(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, pattern := mux.Handler(r)
		if pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		var v verdict
		h.ServeHTTP(&v, r)
		switch v.code {
		case http.StatusMethodNotAllowed:
			if allow := v.header.Get("Allow"); allow != "" {
				w.Header().Set("Allow", allow)
			}
			if err := MethodNotAllowed(w, r); err != nil {
				log.Printf("errorpage: render 405 for %s %s: %v", r.Method, r.URL.Path, err)
			}
		case http.StatusNotFound:
			if err := NotFound(w, r); err != nil {
				log.Printf("errorpage: render 404 for %s %s: %v", r.Method, r.URL.Path, err)
			}
		default:
			mux.ServeHTTP(w, r)
		}
	})
}

// verdict is the ResponseWriter Handler gives the mux's own 404/405/
// redirect handler: it keeps the status and headers and drops the body.
type verdict struct {
	header http.Header
	code   int
}

func (v *verdict) Header() http.Header {
	if v.header == nil {
		v.header = http.Header{}
	}
	return v.header
}

func (v *verdict) WriteHeader(code int) {
	if v.code == 0 {
		v.code = code
	}
}

func (v *verdict) Write(b []byte) (int, error) {
	if v.code == 0 {
		v.code = http.StatusOK
	}
	return len(b), nil
}
