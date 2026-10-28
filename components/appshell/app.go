package appshell

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
)

// App is the app-wide configuration shared by every page: name, navigation,
// and chrome options. Install Middleware once, then call Render from each
// handler.
type App struct {
	Name, HomeHref string
	Layout         Layout
	Nav            []NavItem
	Groups         []NavGroup
	Width          Width
	MaxVisible     int
	ThemeToggle    bool
	LiveDot        *LiveDot
	// Head, when non-nil, renders extra content at the end of <head>.
	Head templ.Component
	// User returns the signed-in user for r; nil means no user menu.
	User func(*http.Request) *User
}

type ctxKey int

const (
	appKey ctxKey = iota
	pathKey
)

// Middleware stores the App and the request path in the request context so
// Shell can mark the active nav item without being handed the path.
func (a *App) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), appKey, a)
		ctx = context.WithValue(ctx, pathKey, r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext returns the App stored by Middleware.
func FromContext(ctx context.Context) (*App, bool) {
	a, ok := ctx.Value(appKey).(*App)
	return a, ok && a != nil
}

// PathFromContext returns the request path stored by Middleware, or "".
func PathFromContext(ctx context.Context) string {
	p, _ := ctx.Value(pathKey).(string)
	return p
}

// IsHX reports whether r was issued by htmx.
func IsHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// ShellProps builds the Shell Props for r from the App's fields.
func (a *App) ShellProps(r *http.Request) Props {
	p := Props{
		AppName:     a.Name,
		HomeHref:    a.HomeHref,
		Layout:      a.Layout,
		Nav:         a.Nav,
		Groups:      a.Groups,
		CurrentPath: r.URL.Path,
		MaxVisible:  a.MaxVisible,
		ThemeToggle: a.ThemeToggle,
		LiveDot:     a.LiveDot,
		Width:       a.Width,
	}
	if a.User != nil {
		p.User = a.User(r)
	}
	return p
}

// Render writes a page: the bare body for an htmx request, otherwise the
// full Document wrapping Shell wrapping body. It returns the render error
// rather than writing a 500 itself.
func (a *App) Render(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx := r.Context()
	if IsHX(r) {
		return body.Render(ctx, w)
	}
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Shell(a.ShellProps(r)).Render(templ.WithChildren(ctx, body), w)
	})
	doc := Document(DocProps{Title: title, AppName: a.Name, Head: a.Head})
	return doc.Render(templ.WithChildren(ctx, page), w)
}
