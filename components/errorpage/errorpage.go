// Package errorpage renders 404, 403 and 500 pages as one component and
// answers a request with the right status code, with or without an
// appshell.App in the request context.
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

// Internal answers r with the default 500 page showing requestID (when
// non-empty) so the user can quote it.
func Internal(w http.ResponseWriter, r *http.Request, requestID string) error {
	return Respond(w, r, Props{Code: http.StatusInternalServerError, RequestID: requestID})
}
