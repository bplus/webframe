// Package errorpage renders 404, 403, 405 and 500 pages as one component
// and answers a request with the right status code, with or without an
// appshell.App in the request context.
//
// # 404, 405 and Go's ServeMux
//
// http.ServeMux answers a method mismatch and a missing route itself, in
// plain text, before any handler runs, and has no hook for either. Wrap the
// mux in Handler and both become the styled pages:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("GET /items", listItems)
//	mux.HandleFunc("POST /items", createItem)
//	http.ListenAndServe(addr, app.Middleware(errorpage.Handler(mux)))
//
// Handler sits inside App.Middleware so the pages render in the shell. Do
// not also register a "/" catch-all for 404s: with one, the mux never
// reports a missing route, and a wrong method on a real route renders as a
// 404 instead of a 405.
//
// # Panics
//
// appshell.Middleware recovers handler panics and calls App.OnPanic, whose
// default is an unstyled 500. Assign PanicHandler to answer with this
// package's 500 page instead, showing the logged request id:
//
//	app.OnPanic = errorpage.PanicHandler
package errorpage

import (
	"log"
	"net/http"

	"github.com/bplus/webframe/components/appshell"
)

// Respond writes the error page for r with p.Code as the status (500 when
// 0). With an appshell.App in the request context it renders inside the
// shell; without one (say, a failure in middleware that runs before
// App.Middleware) it renders a bare document. A plain htmx request gets
// the body only. A render error is returned, not written, since the status
// line has already gone out.
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

// MethodNotAllowed answers r with the default 405 page. It does not set the
// Allow header; the caller must (Handler does).
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) error {
	return Respond(w, r, Props{Code: http.StatusMethodNotAllowed})
}

// Internal answers r with the default 500 page showing requestID (when
// non-empty) so the user can quote it.
func Internal(w http.ResponseWriter, r *http.Request, requestID string) error {
	return Respond(w, r, Props{Code: http.StatusInternalServerError, RequestID: requestID})
}

// PanicHandler is an appshell.App.OnPanic that answers a recovered panic
// with the 500 page showing requestID. Middleware has already logged the
// panic; a render error is logged here, since it cannot be returned.
//
//	app.OnPanic = errorpage.PanicHandler
func PanicHandler(w http.ResponseWriter, r *http.Request, requestID string, _ any) {
	if err := Internal(w, r, requestID); err != nil {
		log.Printf("errorpage: render 500 for %s %s (request %s): %v", r.Method, r.URL.Path, requestID, err)
	}
}

// Handler wraps mux so its plain-text 404 and 405 answers become the
// styled pages. Requests with a matching pattern go to the mux untouched.
//
// For the rest, Handler runs the mux's own fallback handler into a
// header-only recorder and reads its status: 405 becomes MethodNotAllowed
// (with the mux's Allow header copied over), 404 becomes NotFound, anything
// else (a path-cleaning redirect) goes to the mux. Asking the mux rather
// than probing methods by hand keeps Handler in step with ServeMux's rules
// across Go versions. No app handler is ever run on the mux's behalf.
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

// verdict records the status and headers of the mux's fallback handler and
// drops the body.
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
