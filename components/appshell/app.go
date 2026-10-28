package appshell

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

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
	// Right, when non-nil, renders in the topbar's right group (before the
	// user menu); ShellProps copies it into Props.Right.
	Right templ.Component
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

// Must returns the App stored by Middleware, or an error naming Middleware
// when the handler chain has none; page types call it from their Respond.
func Must(ctx context.Context) (*App, error) {
	a, ok := FromContext(ctx)
	if !ok {
		return nil, errors.New("appshell: no App in request context; wrap the handler in (*App).Middleware")
	}
	return a, nil
}

// PathFromContext returns the request path stored by Middleware, or "".
func PathFromContext(ctx context.Context) string {
	p, _ := ctx.Value(pathKey).(string)
	return p
}

// IsHX reports whether r was issued by htmx (any kind of htmx request,
// including boosted navigation and history restores).
func IsHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// WantsFragment reports whether r wants only the page body. htmx 2 sends
// HX-Request for boosted links (HX-Boosted) and history restores
// (HX-History-Restore-Request) too, and swaps those responses into <body>,
// so they need the full document; only a plain htmx request gets a fragment.
func WantsFragment(r *http.Request) bool {
	return IsHX(r) &&
		r.Header.Get("HX-Boosted") != "true" &&
		r.Header.Get("HX-History-Restore-Request") != "true"
}

// varyHeaders lists the request headers Render and RenderParts output
// depends on, so a cache never serves a fragment for a full-page request or
// the reverse. HX-Target is included because RenderParts writes a different
// region depending on it: a plain htmx request targeting the results id
// gets only that part, any other target gets the full body.
const varyHeaders = "HX-Request, HX-Boosted, HX-History-Restore-Request, HX-Target"

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
		Right:       a.Right,
		Width:       a.Width,
	}
	if a.User != nil {
		p.User = a.User(r)
	}
	return p
}

// Render writes a page: the bare body when WantsFragment(r), otherwise the
// full Document wrapping Shell wrapping body. It always sets Vary on the
// htmx request headers. It returns the render error
// rather than writing a 500 itself.
func (a *App) Render(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	return a.RenderParts(w, r, title, body, nil)
}

// RenderParts is Render with named swap regions. parts maps a swap target's
// element id to the component that renders just that region. The response
// depends on the request shape:
//
//   - WantsFragment(r) and the HX-Target header (with one leading "#"
//     stripped) names a key in parts: only that part is written.
//   - WantsFragment(r) otherwise (no HX-Target, an unknown target, or a
//     target whose part is nil): full is written, the body with no shell.
//   - any other request (not htmx, boosted, or a history restore): the whole
//     Document wrapping Shell wrapping full, as Render does.
//
// Content-Type and Vary are set exactly as in Render. An empty HX-Target
// never matches, even when parts has an "" key.
func (a *App) RenderParts(w http.ResponseWriter, r *http.Request, title string, full templ.Component, parts map[string]templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", varyHeaders)
	ctx := r.Context()
	if WantsFragment(r) {
		if target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#"); target != "" {
			if part := parts[target]; part != nil {
				return part.Render(ctx, w)
			}
		}
		return full.Render(ctx, w)
	}
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Shell(a.ShellProps(r)).Render(templ.WithChildren(ctx, full), w)
	})
	doc := Document(DocProps{Title: title, AppName: a.Name, Head: a.Head})
	return doc.Render(templ.WithChildren(ctx, page), w)
}
