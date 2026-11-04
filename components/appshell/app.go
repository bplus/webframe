package appshell

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/commandpalette"
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
	// Right, when non-nil, renders in the topbar's right group, before the
	// user menu.
	Right templ.Component
	// Palette, when non-nil, gives every shell page a command palette and
	// its Search (⌘K) trigger. RenderBare pages have neither.
	Palette *commandpalette.Props
	// PaletteFor, when non-nil, builds the palette per request and takes
	// precedence over Palette. It is called once per shell render; a nil
	// result means no palette. Fragment renders never call it.
	PaletteFor func(*http.Request) *commandpalette.Props
	// User returns the signed-in user for r; nil means no user menu.
	User func(*http.Request) *User
	// Flash, when non-nil, supplies the one-shot message for the #wf-flash
	// slot (typically flash.Take). Render and RenderParts call it once per
	// response, before anything is written, so it may set cookies on w. A
	// fragment response sends it as an out-of-band swap of #wf-flash.
	// RenderBare never calls it. Handlers needing a non-200 status should
	// use RenderStatus rather than calling WriteHeader themselves.
	Flash func(w http.ResponseWriter, r *http.Request) templ.Component
	// OnPanic writes the whole 500 response (status, headers, body) for a
	// panic Middleware recovered before anything was written. nil means a
	// bare Document showing the request id; assign errorpage.PanicHandler
	// for the styled page. If OnPanic itself panics before writing, the
	// default answers instead.
	OnPanic func(w http.ResponseWriter, r *http.Request, requestID string, err any)
}

type ctxKey int

const (
	appKey ctxKey = iota
	pathKey
	requestIDKey
)

// Middleware stores the App, the request path and a request id in the
// context, and recovers panics in next.
//
// The request id is reused from an enclosing Middleware, else taken from
// the X-Request-Id header if it is at most 64 bytes of [A-Za-z0-9._-],
// else generated. The id used is echoed in the response's X-Request-Id
// header (set before next runs), so a browser's network panel or a proxy
// log can be matched to the server log and to the id an error page shows.
//
// A recovered panic is logged with the request id and stack. If nothing
// has been written yet, any redirect or htmx redirect/retarget headers are
// dropped and App.OnPanic answers with a 500; otherwise the response ends
// as it stands. http.ErrAbortHandler is re-panicked. The wrapped
// ResponseWriter still supports http.Flusher, http.Hijacker and
// io.ReaderFrom.
func (a *App) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestID(r.Context())
		if id == "" {
			if h := r.Header.Get("X-Request-Id"); validRequestID(h) {
				id = h
			}
		}
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), appKey, a)
		ctx = context.WithValue(ctx, pathKey, r.URL.Path)
		ctx = context.WithValue(ctx, requestIDKey, id)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-Id", id)
		rw := &recordingWriter{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			log.Printf("appshell: panic serving %s %s (request %s): %v\n%s", r.Method, r.URL.Path, id, v, debug.Stack())
			if rw.wrote {
				return
			}
			stripRedirectHeaders(rw.Header())
			a.answerPanic(rw, r, id, v)
		}()
		next.ServeHTTP(rw, r)
	})
}

// redirectHeaders are deleted before a panic's 500 is written, so the
// error page is never redirected, retargeted or pushed to the URL bar.
var redirectHeaders = []string{
	"Location",
	"HX-Redirect", "HX-Location", "HX-Refresh",
	"HX-Reswap", "HX-Retarget",
	"HX-Push-Url", "HX-Replace-Url",
}

// stripRedirectHeaders deletes redirectHeaders from h.
func stripRedirectHeaders(h http.Header) {
	for _, k := range redirectHeaders {
		h.Del(k)
	}
}

// answerPanic writes the 500 through App.OnPanic, or the default. A panic
// in OnPanic is logged and, if nothing was written yet, the default
// answers instead, so a broken error page never drops the connection.
func (a *App) answerPanic(rw *recordingWriter, r *http.Request, id string, v any) {
	if a.OnPanic == nil {
		a.defaultOnPanic(rw, r, id, v)
		return
	}
	defer func() {
		v2 := recover()
		if v2 == nil {
			return
		}
		if v2 == http.ErrAbortHandler {
			panic(v2)
		}
		log.Printf("appshell: OnPanic panicked for %s %s (request %s): %v\n%s", r.Method, r.URL.Path, id, v2, debug.Stack())
		if rw.wrote {
			return
		}
		stripRedirectHeaders(rw.Header())
		a.defaultOnPanic(rw, r, id, v)
	}()
	a.OnPanic(rw, r, id, v)
}

// RequestID returns the request id Middleware stored, or "" without
// Middleware. Error pages show it so a user's report can be matched to
// the log.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// maxRequestIDLen bounds an accepted X-Request-Id header (a UUID is 36).
const maxRequestIDLen = 64

// validRequestID reports whether a client-sent X-Request-Id is safe to
// write into log lines and pages.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}

// newRequestID returns 8 random bytes as 16 hex digits. crypto/rand.Read
// never returns an error on supported platforms.
func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// defaultOnPanic answers a panic with a 500 and a bare PanicPage that
// links home to App.HomeHref.
func (a *App) defaultOnPanic(w http.ResponseWriter, r *http.Request, requestID string, _ any) {
	SetHeaders(w)
	w.WriteHeader(http.StatusInternalServerError)
	if err := renderDocument(w, r, DocProps{Title: "Something went wrong", AppName: a.Name, Head: a.Head}, PanicPage(requestID, a.HomeHref)); err != nil {
		log.Printf("appshell: render default 500 for %s %s (request %s): %v", r.Method, r.URL.Path, requestID, err)
	}
}

// recordingWriter records whether anything reached the client, so the
// recover path knows whether a 500 can still be written.
type recordingWriter struct {
	http.ResponseWriter
	wrote bool
}

func (w *recordingWriter) WriteHeader(code int) {
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *recordingWriter) Write(b []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher. http.Flusher cannot report an error, so
// one from the underlying writer is dropped.
func (w *recordingWriter) Flush() {
	w.wrote = true
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Hijack implements http.Hijacker. A hijacked connection counts as
// written.
func (w *recordingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err == nil {
		w.wrote = true
	}
	return conn, rw, err
}

// ReadFrom implements io.ReaderFrom, delegating to the underlying writer
// so net/http keeps its sendfile path. It counts as written even when src
// is empty, since net/http sends the headers first.
func (w *recordingWriter) ReadFrom(src io.Reader) (int64, error) {
	w.wrote = true
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}
	return io.Copy(writerOnly{w.ResponseWriter}, src)
}

// writerOnly hides ReadFrom from io.Copy, which would otherwise recurse.
type writerOnly struct{ io.Writer }

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *recordingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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

// Redirect sends the browser to url, typically after a successful POST.
// A plain request gets 303 See Other with a Location header, so a reload
// on the new page does not submit the form again. An htmx request gets an
// empty 200 with an HX-Redirect header instead: htmx follows a 3xx itself
// and would swap the whole new page into the form, while HX-Redirect makes
// it load url as a full page. Set any flash cookie before calling it.
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	if IsHX(r) {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
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

// varyHeaders lists the request headers the rendered output depends on,
// so a cache never serves a fragment for a full-page request or the
// reverse.
const varyHeaders = "HX-Request, HX-Boosted, HX-History-Restore-Request, HX-Target"

// palette returns the command palette for r: PaletteFor's result when it
// is set, else Palette.
func (a *App) palette(r *http.Request) *commandpalette.Props {
	if a.PaletteFor != nil {
		return a.PaletteFor(r)
	}
	return a.Palette
}

// ShellProps builds the Shell Props for r from the App's fields, calling
// PaletteFor and User.
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
		Palette:     a.palette(r),
		Width:       a.Width,
	}
	if a.User != nil {
		p.User = a.User(r)
	}
	return p
}

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

// Render writes a page: the bare body when WantsFragment(r), otherwise the
// full Document wrapping Shell wrapping body. The title is
// "{title} — {App.Name}", or the app name alone when title is empty. It
// sets Content-Type and Vary, and returns render errors rather than
// writing a 500. Use RenderStatus for a non-200 status.
func (a *App) Render(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	return a.RenderParts(w, r, title, body, nil)
}

// RenderBare is Render without the Shell: no nav, no flash slot, and
// App.Flash is not called. Use it for sign-in and similar pages. The body
// still sits in a <main id="wf-main"> (MainID, with no classes, so it adds
// no layout), so an error page that errorpage sends to the content region
// lands on a bare page too.
func (a *App) RenderBare(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
	return renderDocument(w, r, DocProps{Title: title, AppName: a.Name, Head: a.Head}, body)
}

// RenderBare is (*App).RenderBare for handlers that run outside
// App.Middleware; name is the app name used in the title.
func RenderBare(w http.ResponseWriter, r *http.Request, name, title string, body templ.Component) error {
	return renderDocument(w, r, DocProps{Title: title, AppName: name}, body)
}

// SetHeaders sets Content-Type and Vary as every render does. A handler
// that writes its own status must call it before WriteHeader, since
// net/http freezes headers at that point.
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
// Document described by p holding body inside <main id="wf-main">.
func renderDocument(w http.ResponseWriter, r *http.Request, p DocProps, body templ.Component) error {
	SetHeaders(w)
	ctx := r.Context()
	if WantsFragment(r) {
		return body.Render(ctx, w)
	}
	return Document(p).Render(templ.WithChildren(ctx, bareMain(body)), w)
}

// bareMain wraps body in the content region a bare page has: the <main>
// with MainID, unstyled.
func bareMain(body templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := io.WriteString(w, `<main id="`+MainID+`">`); err != nil {
			return err
		}
		if err := body.Render(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, `</main>`)
		return err
	})
}

// RenderFragment writes frag alone as an htmx fragment response, the way
// Render answers a plain htmx request: it sets Content-Type and Vary and,
// when an App with Flash is in the request context, consumes a pending
// flash and sends it after frag as an out-of-band swap of #wf-flash.
// Without an App it writes frag alone. Set any other headers first.
func RenderFragment(w http.ResponseWriter, r *http.Request, frag templ.Component) error {
	SetHeaders(w)
	a, ok := FromContext(r.Context())
	if !ok {
		return frag.Render(r.Context(), w)
	}
	return a.writeFragment(w, r, 0, frag)
}

// writeFragment renders frag with status (0 means 200), followed by the
// pending flash out of band. A fragment render still consumes the flash,
// so a flash set by an htmx action shows now rather than on the next full
// render. Flash runs before the status is written: it may set a cookie.
func (a *App) writeFragment(w http.ResponseWriter, r *http.Request, status int, frag templ.Component) error {
	var oob templ.Component
	if a.Flash != nil {
		if f := a.Flash(w, r); f != nil {
			oob = flashOOB(f)
		}
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	ctx := r.Context()
	if err := frag.Render(ctx, w); err != nil || oob == nil {
		return err
	}
	return oob.Render(ctx, w)
}

// RenderParts is Render with named swap regions: parts maps an element id
// to the component rendering just that region. A fragment request whose
// HX-Target names a key in parts gets only that part; any other fragment
// request gets full without the shell; a full-page request renders as
// Render does.
func (a *App) RenderParts(w http.ResponseWriter, r *http.Request, title string, full templ.Component, parts map[string]templ.Component) error {
	return a.RenderPartsStatus(w, r, 0, title, full, parts)
}

// RenderStatus is Render with a status code (0 means 200), written after
// App.Flash has had a chance to set cookies.
func (a *App) RenderStatus(w http.ResponseWriter, r *http.Request, status int, title string, body templ.Component) error {
	return a.RenderPartsStatus(w, r, status, title, body, nil)
}

// ContentReswap is the HX-Reswap value RenderContentStatus sends: replace
// the inside of the content region and scroll the window to the top.
const ContentReswap = "innerHTML show:window:top"

// RenderContentStatus is RenderStatus for an answer that replaces the
// whole page content, whatever element sent the request. A full-page
// request renders exactly as RenderStatus does. A plain htmx request gets
// HX-Retarget "#wf-main" (the shell's <main>, MainID) and HX-Reswap
// ContentReswap, and its body is what <main> holds in a full render: the
// #wf-flash slot, with a pending flash inside it, then body. No URL is
// pushed. errorpage uses it so an error answered to a button that targets
// itself replaces the page instead of landing inside the button.
func (a *App) RenderContentStatus(w http.ResponseWriter, r *http.Request, status int, title string, body templ.Component) error {
	if !WantsFragment(r) {
		return a.RenderStatus(w, r, status, title, body)
	}
	SetHeaders(w)
	w.Header().Set("HX-Retarget", "#"+MainID)
	w.Header().Set("HX-Reswap", ContentReswap)
	var f templ.Component
	if a.Flash != nil {
		f = a.Flash(w, r) // before the status: Flash may set or clear a cookie
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	ctx := r.Context()
	if _, err := io.WriteString(w, `<div id="wf-flash">`); err != nil {
		return err
	}
	if f != nil {
		if err := f.Render(ctx, w); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, `</div>`); err != nil {
		return err
	}
	return body.Render(ctx, w)
}

// RenderPartsStatus is RenderParts with a status code, as in RenderStatus.
func (a *App) RenderPartsStatus(w http.ResponseWriter, r *http.Request, status int, title string, full templ.Component, parts map[string]templ.Component) error {
	SetHeaders(w)
	ctx := r.Context()
	writeStatus := func() {
		if status != 0 {
			w.WriteHeader(status)
		}
	}
	if WantsFragment(r) {
		frag := full
		if target := strings.TrimPrefix(r.Header.Get("HX-Target"), "#"); target != "" {
			if part := parts[target]; part != nil {
				frag = part
			}
		}
		return a.writeFragment(w, r, status, frag)
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
	doc := Document(DocProps{Title: title, AppName: a.Name, Head: a.Head, Palette: props.Palette})
	return doc.Render(templ.WithChildren(ctx, page), w)
}

// panicHome is the panic page's home link: href, or "/" when empty.
func panicHome(href string) string {
	if href == "" {
		return "/"
	}
	return href
}
