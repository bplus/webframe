// Package errorpage renders 404, 403, 405 and 500 pages as one component
// and answers a request with the right status code, with or without an
// appshell.App in the request context.
//
// # 405 and Go's ServeMux
//
// Go's http.ServeMux (1.22 and later, method patterns such as "GET /items")
// answers a method mismatch itself: a plain-text "Method Not Allowed" with
// an Allow header, written before any handler runs, so a handler can never
// turn it into this page. ServeMux has no hook for it. An app that wants
// the styled page wraps the mux and decides first: ServeMux.Handler
// returns an empty pattern both when nothing matches the path and when
// something matches the path but not the method, so the wrapper probes the
// other methods to tell the two apart (and collects the Allow header the
// mux would have sent):
//
//	func styledErrors(mux *http.ServeMux) http.Handler {
//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			if _, pattern := mux.Handler(r); pattern != "" {
//				mux.ServeHTTP(w, r) // a real route, or the mux's own redirect
//				return
//			}
//			var allowed []string
//			probe := r.Clone(r.Context())
//			for _, m := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
//				probe.Method = m
//				if _, pattern := mux.Handler(probe); pattern != "" {
//					allowed = append(allowed, m)
//				}
//			}
//			if len(allowed) == 0 {
//				_ = errorpage.NotFound(w, r)
//				return
//			}
//			w.Header().Set("Allow", strings.Join(allowed, ", "))
//			_ = errorpage.MethodNotAllowed(w, r)
//		})
//	}
//
//	http.ListenAndServe(addr, app.Middleware(styledErrors(mux)))
//
// The wrapper sits inside App.Middleware so the page renders in the shell.
// An app that is content with the mux's plain 405 (an API path, a form
// that is only ever posted by its own page) needs none of this.
package errorpage

import (
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
// here"). It does not set the Allow header; the caller knows the methods
// (see the package doc for the ServeMux wrapper that collects them).
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) error {
	return Respond(w, r, Props{Code: http.StatusMethodNotAllowed})
}

// Internal answers r with the default 500 page showing requestID (when
// non-empty) so the user can quote it.
func Internal(w http.ResponseWriter, r *http.Request, requestID string) error {
	return Respond(w, r, Props{Code: http.StatusInternalServerError, RequestID: requestID})
}
