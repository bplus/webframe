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
	// Flash, when non-nil, supplies the one-shot message for the Shell's
	// #wf-flash slot. Render and RenderParts call it exactly once per
	// response: one that includes the Shell (a non-htmx request, a boosted
	// link, a history restore) renders the result inside the slot; a
	// body-only or part-only htmx response appends it after the fragment
	// as an out-of-band swap of #wf-flash (nothing when the result is nil),
	// so a flash set by an htmx action shows at once. RenderBare never
	// calls it. It runs before the status and the first byte of the body are
	// written (a handler that needs a non-200 status uses RenderStatus or
	// RenderPartsStatus instead of calling WriteHeader itself), so it may set
	// or clear cookies on w. A nil result renders the empty slot on
	// a Shell render. appshell holds no cookie logic itself: an app wires
	// App.Flash = flash.Take.
	Flash func(w http.ResponseWriter, r *http.Request) templ.Component
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
//
// An empty title makes the document title the app name alone (an error
// page uses this); otherwise it is "{title} — {App.Name}".
//
// Render never calls WriteHeader itself: the first body write sends 200
// unless the handler wrote a status first, in which case that status
// stands. Content-Type and Vary are set before anything is written, so a
// handler that writes a status before calling Render should expect them
// to be dropped by net/http (headers are frozen at WriteHeader).
// flashOOB wraps a flash component in the #wf-flash slot marked for an
// out-of-band swap, so a fragment response replaces the slot in place.
func flashOOB(f templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, `<div id="wf-flash" hx-swap-oob="true">`); err != nil {
			return err
		}
		if err := f.Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</div>`)
		return err
	})
}

func (a *App) Render(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	return a.RenderParts(w, r, title, body, nil)
}

// RenderBare writes a page with no Shell: the bare body when
// WantsFragment(r), otherwise the full Document (title suffix, stylesheet,
// theme init, script bundle, App.Head) holding body as its only child. No
// nav, no #wf-flash slot, and App.Flash is not called. Sign-in and signed-
// out pages use it. Content-Type and Vary are set exactly as in Render.
func (a *App) RenderBare(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	return renderDocument(w, r, DocProps{Title: title, AppName: a.Name, Head: a.Head}, body)
}

// RenderBare is (*App).RenderBare with no App: name is the app name for
// the title suffix and the Document uses its default theme init and no
// extra head content. A handler that runs before App.Middleware (a 500
// from an outer middleware, a sign-in page outside the shell) can answer
// with it. Content-Type and Vary are set exactly as in (*App).Render.
func RenderBare(w http.ResponseWriter, r *http.Request, name, title string, body templ.Component) error {
	return renderDocument(w, r, DocProps{Title: title, AppName: name}, body)
}

// SetHeaders sets the response headers every render shares: Content-Type
// (text/html) and Vary on the htmx request headers. Render, RenderParts
// and RenderBare call it themselves, so a handler only needs it when it
// writes a non-200 status before rendering: net/http freezes the header
// map at WriteHeader, so the order is SetHeaders, WriteHeader, Render.
// errorpage.Respond does exactly that.
func SetHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	for _, v := range h.Values("Vary") {
		if v == varyHeaders {
			return
		}
	}
	h.Add("Vary", varyHeaders)
}

// renderDocument writes body alone when WantsFragment(r), otherwise the
// Document described by p holding body as its only child.
func renderDocument(w http.ResponseWriter, r *http.Request, p DocProps, body templ.Component) error {
	SetHeaders(w)
	ctx := r.Context()
	if WantsFragment(r) {
		return body.Render(ctx, w)
	}
	return Document(p).Render(templ.WithChildren(ctx, body), w)
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
//
// Only the third shape includes the Shell, so only it calls App.Flash
// (once, before any output), and the result fills the #wf-flash slot.
func (a *App) RenderParts(w http.ResponseWriter, r *http.Request, title string, full templ.Component, parts map[string]templ.Component) error {
	return a.RenderPartsStatus(w, r, 0, title, full, parts)
}

// RenderStatus is Render with a status code: status is written after
// App.Flash has run and the headers are set (a status written by the
// handler first would freeze the header map and lose the flash's clearing
// cookie, Content-Type and Vary). 0 means 200. errorpage uses it.
func (a *App) RenderStatus(w http.ResponseWriter, r *http.Request, status int, title string, body templ.Component) error {
	return a.RenderPartsStatus(w, r, status, title, body, nil)
}

// RenderPartsStatus is RenderParts with a status code, written in the same
// order as RenderStatus. 0 means 200. formpage uses it for a 422.
func (a *App) RenderPartsStatus(w http.ResponseWriter, r *http.Request, status int, title string, full templ.Component, parts map[string]templ.Component) error {
	SetHeaders(w)
	ctx := r.Context()
	writeStatus := func() {
		if status != 0 {
			w.WriteHeader(status)
		}
	}
	if WantsFragment(r) {
		// A fragment render still consumes a pending flash: the banner is
		// sent out of band into the page's #wf-flash slot, so a flash set
		// by an htmx action (or by a redirect htmx followed as XHR) shows
		// on the page the user is looking at instead of waiting for the
		// next full render. Called before the first write, like below.
		var oob templ.Component
		if a.Flash != nil {
			if f := a.Flash(w, r); f != nil {
				oob = flashOOB(f)
			}
		}
		frag := full
		if target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#"); target != "" {
			if part := parts[target]; part != nil {
				frag = part
			}
		}
		writeStatus()
		if err := frag.Render(ctx, w); err != nil || oob == nil {
			return err
		}
		return oob.Render(ctx, w)
	}
	props := a.ShellProps(r)
	if a.Flash != nil {
		// Before the first write: Flash may set or clear a cookie.
		props.Flash = a.Flash(w, r)
	}
	writeStatus()
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Shell(props).Render(templ.WithChildren(ctx, full), w)
	})
	doc := Document(DocProps{Title: title, AppName: a.Name, Head: a.Head})
	return doc.Render(templ.WithChildren(ctx, page), w)
}
