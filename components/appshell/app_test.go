package appshell

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func pageBody(text string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("<p>" + text + "</p>"))
		return err
	})
}

func testApp() *App {
	return &App{
		Name: "Demo",
		Nav:  []NavItem{{Label: "Inbox", Href: "/inbox"}, {Label: "Movies", Href: "/movies"}},
	}
}

func serve(t *testing.T, a *App, hx bool) string {
	t.Helper()
	var h map[string]string
	if hx {
		h = map[string]string{"HX-Request": "true"}
	}
	return serveHeaders(t, a, h)
}

func serveHeaders(t *testing.T, a *App, headers map[string]string) string {
	t.Helper()
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Render(w, r, "Movies", pageBody("hello page")); err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	req := httptest.NewRequest("GET", "/movies", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if v := rec.Header().Get("Vary"); v != "HX-Request, HX-Boosted, HX-History-Restore-Request, HX-Target" {
		t.Errorf("Vary = %q", v)
	}
	return rec.Body.String()
}

func TestMiddleware_PutsAppAndPathInContext(t *testing.T) {
	a := testApp()
	called := false
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got, ok := FromContext(r.Context())
		if !ok || got != a {
			t.Errorf("FromContext = %v, %v", got, ok)
		}
		if p := PathFromContext(r.Context()); p != "/movies" {
			t.Errorf("PathFromContext = %q", p)
		}
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/movies", nil))
	if !called {
		t.Fatal("handler not called")
	}
	if _, ok := FromContext(context.Background()); ok {
		t.Error("FromContext on empty ctx should be false")
	}
	if p := PathFromContext(context.Background()); p != "" {
		t.Errorf("PathFromContext on empty ctx = %q", p)
	}
}

func TestRender_FullDocument(t *testing.T) {
	got := serve(t, testApp(), false)
	for _, want := range []string{"<title>Movies — Demo</title>", "wf-topbar", "hello page"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(strings.ToLower(got), "<!doctype html>") {
		t.Errorf("missing doctype in:\n%s", got)
	}
}

func TestRender_HXRequestBodyOnly(t *testing.T) {
	got := serve(t, testApp(), true)
	if !strings.Contains(got, "hello page") {
		t.Errorf("missing body in:\n%s", got)
	}
	for _, bad := range []string{"<html", "wf-topbar", "<title>"} {
		if strings.Contains(got, bad) {
			t.Errorf("unexpected %q in:\n%s", bad, got)
		}
	}
}

func TestRender_BoostedAndHistoryRestoreGetFullDocument(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"boosted":         {"HX-Request": "true", "HX-Boosted": "true"},
		"history-restore": {"HX-Request": "true", "HX-History-Restore-Request": "true"},
	} {
		got := serveHeaders(t, testApp(), headers)
		for _, want := range []string{"<title>Movies — Demo</title>", "wf-topbar", "hello page"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q in:\n%s", name, want, got)
			}
		}
	}
}

func TestShell_PathFromContext(t *testing.T) {
	a := testApp()
	var got string
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if err := Shell(Props{AppName: "Demo", Nav: a.Nav}).Render(r.Context(), &buf); err != nil {
			t.Errorf("render: %v", err)
		}
		got = buf.String()
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/inbox", nil))
	if !strings.Contains(got, `aria-current="page"`) || !strings.Contains(got, "wf-nav-link--active") {
		t.Errorf("no active item in:\n%s", got)
	}
}

func TestShellProps_CallsUser(t *testing.T) {
	a := testApp()
	a.User = func(*http.Request) *User { return &User{Name: "B"} }
	p := a.ShellProps(httptest.NewRequest("GET", "/movies", nil))
	if p.User == nil || p.User.Name != "B" {
		t.Errorf("User = %+v", p.User)
	}
	if p.CurrentPath != "/movies" || p.AppName != "Demo" {
		t.Errorf("props = %+v", p)
	}
}

// serveParts runs RenderParts through Middleware for one request shape and
// returns the body. The full body is a page header plus the results part;
// the parts map names the results region.
func serveParts(t *testing.T, a *App, headers map[string]string) string {
	t.Helper()
	results := pageBody("results rows")
	full := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := w.Write([]byte("<h1>Page Header</h1>")); err != nil {
			return err
		}
		return results.Render(ctx, w)
	})
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := a.RenderParts(w, r, "Movies", full, map[string]templ.Component{"list-results": results})
		if err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	req := httptest.NewRequest("GET", "/movies", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if v := rec.Header().Get("Vary"); v != "HX-Request, HX-Boosted, HX-History-Restore-Request, HX-Target" {
		t.Errorf("Vary = %q", v)
	}
	return rec.Body.String()
}

func TestRenderParts_TargetedPart(t *testing.T) {
	for name, target := range map[string]string{"bare": "list-results", "hashed": "#list-results"} {
		got := serveParts(t, testApp(), map[string]string{"HX-Request": "true", "HX-Target": target})
		if got != "<p>results rows</p>" {
			t.Errorf("%s: body = %q, want the part only", name, got)
		}
		for _, bad := range []string{"Page Header", "<html", "wf-topbar"} {
			if strings.Contains(got, bad) {
				t.Errorf("%s: unexpected %q in:\n%s", name, bad, got)
			}
		}
	}
}

func TestRenderParts_OtherTargetGetsBody(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"other-target": {"HX-Request": "true", "HX-Target": "something-else"},
		"no-target":    {"HX-Request": "true"},
		"empty-target": {"HX-Request": "true", "HX-Target": ""},
	} {
		got := serveParts(t, testApp(), headers)
		for _, want := range []string{"Page Header", "results rows"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q in:\n%s", name, want, got)
			}
		}
		for _, bad := range []string{"<html", "wf-topbar", "<title>"} {
			if strings.Contains(got, bad) {
				t.Errorf("%s: unexpected %q in:\n%s", name, bad, got)
			}
		}
	}
}

func TestRenderParts_BoostedGetsDocument(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"boosted":         {"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "list-results"},
		"history-restore": {"HX-Request": "true", "HX-History-Restore-Request": "true", "HX-Target": "list-results"},
		"plain-get":       {"HX-Target": "list-results"},
	} {
		got := serveParts(t, testApp(), headers)
		if !strings.Contains(strings.ToLower(got), "<!doctype html>") {
			t.Errorf("%s: missing doctype in:\n%s", name, got)
		}
		for _, want := range []string{"<title>Movies — Demo</title>", "wf-topbar", "Page Header", "results rows"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q in:\n%s", name, want, got)
			}
		}
	}
}

func TestMust_NoApp(t *testing.T) {
	a, err := Must(context.Background())
	if err == nil || a != nil {
		t.Fatalf("Must on empty ctx = %v, %v; want nil, error", a, err)
	}
	if !strings.Contains(err.Error(), "Middleware") {
		t.Errorf("error %q does not name Middleware", err)
	}
}

func TestMust_WithApp(t *testing.T) {
	a := testApp()
	var got *App
	var gotErr error
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, gotErr = Must(r.Context())
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/movies", nil))
	if gotErr != nil || got != a {
		t.Errorf("Must = %v, %v; want the app, nil", got, gotErr)
	}
}

func TestShellProps_Right(t *testing.T) {
	a := testApp()
	a.Right = pageBody("right slot")
	req := httptest.NewRequest("GET", "/movies", nil)
	p := a.ShellProps(req)
	if p.Right == nil {
		t.Fatal("ShellProps did not copy Right")
	}
	var buf bytes.Buffer
	if err := Shell(p).Render(req.Context(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	i := strings.Index(got, "wf-topbar-right")
	j := strings.Index(got, "right slot")
	if i < 0 || j < 0 || j < i {
		t.Errorf("Right not rendered inside the topbar right group:\n%s", got)
	}
}

// serveBare runs (*App).RenderBare through Middleware for one request shape
// and returns the recorder.
func serveBare(t *testing.T, a *App, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.RenderBare(w, r, "Sign in", pageBody("bare page")); err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	req := httptest.NewRequest("GET", "/signin", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func checkRenderHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if v := rec.Header().Get("Vary"); v != varyHeaders {
		t.Errorf("Vary = %q", v)
	}
}

func TestRenderBare_FullDocumentWithoutShell(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"plain":           nil,
		"boosted":         {"HX-Request": "true", "HX-Boosted": "true"},
		"history-restore": {"HX-Request": "true", "HX-History-Restore-Request": "true"},
	} {
		rec := serveBare(t, testApp(), headers)
		checkRenderHeaders(t, rec)
		got := rec.Body.String()
		if !strings.Contains(strings.ToLower(got), "<!doctype html>") {
			t.Errorf("%s: missing doctype in:\n%s", name, got)
		}
		for _, want := range []string{"<title>Sign in — Demo</title>", "bare page", "output.css", "confirm.js", `id="wf-confirm"`} {
			if !strings.Contains(got, want) {
				t.Errorf("%s: missing %q in:\n%s", name, want, got)
			}
		}
		for _, bad := range []string{"wf-topbar", "wf-shell", `id="wf-flash"`, "<nav", "<main"} {
			if strings.Contains(got, bad) {
				t.Errorf("%s: unexpected %q (shell leaked into bare render) in:\n%s", name, bad, got)
			}
		}
	}
}

func TestRenderBare_HXRequestBodyOnly(t *testing.T) {
	rec := serveBare(t, testApp(), map[string]string{"HX-Request": "true"})
	checkRenderHeaders(t, rec)
	if got := rec.Body.String(); got != "<p>bare page</p>" {
		t.Errorf("body = %q, want the body only", got)
	}
}

func TestRenderBare_AppHead(t *testing.T) {
	a := testApp()
	a.Head = pageBody("extra head")
	got := serveBare(t, a, nil).Body.String()
	i := strings.Index(got, "extra head")
	j := strings.Index(got, "</head>")
	if i < 0 || j < 0 || i > j {
		t.Errorf("App.Head not rendered inside <head>:\n%s", got)
	}
}

func TestRenderBare_PackageFuncNoApp(t *testing.T) {
	// No Middleware: the handler has no App in its context.
	serveNoApp := func(headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/boom", nil)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		if _, ok := FromContext(req.Context()); ok {
			t.Fatal("test precondition: no App expected in context")
		}
		if err := RenderBare(rec, req, "Acme", "Something broke", pageBody("error body")); err != nil {
			t.Fatalf("render: %v", err)
		}
		return rec
	}
	rec := serveNoApp(nil)
	checkRenderHeaders(t, rec)
	got := rec.Body.String()
	if !strings.Contains(strings.ToLower(got), "<!doctype html>") {
		t.Errorf("missing doctype in:\n%s", got)
	}
	for _, want := range []string{"<title>Something broke — Acme</title>", "error body", "wf-theme", "data-theme", "output.css"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, bad := range []string{"wf-topbar", "wf-shell", `id="wf-flash"`} {
		if strings.Contains(got, bad) {
			t.Errorf("unexpected %q in:\n%s", bad, got)
		}
	}

	rec = serveNoApp(map[string]string{"HX-Request": "true"})
	checkRenderHeaders(t, rec)
	if got := rec.Body.String(); got != "<p>error body</p>" {
		t.Errorf("hx body = %q, want the body only", got)
	}

	// An empty title is the name alone.
	rec = httptest.NewRecorder()
	if err := RenderBare(rec, httptest.NewRequest("GET", "/", nil), "Acme", "", pageBody("x")); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := rec.Body.String(); !strings.Contains(got, "<title>Acme</title>") {
		t.Errorf("missing name-only title in:\n%s", got)
	}
}

func TestRender_EmptyTitleIsAppNameAlone(t *testing.T) {
	a := testApp()
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Render(w, r, "", pageBody("home")); err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	got := rec.Body.String()
	if !strings.Contains(got, "<title>Demo</title>") {
		t.Errorf("missing app-only title in:\n%s", got)
	}
	if strings.Contains(got, "<title> — Demo</title>") || strings.Contains(got, "<title>— Demo</title>") {
		t.Errorf("dangling separator in title:\n%s", got)
	}
}

func TestRender_StatusWrittenByHandlerWins(t *testing.T) {
	a := testApp()
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if err := a.Render(w, r, "", pageBody("broken")); err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 written by the handler to stand", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "broken") {
		t.Errorf("missing body in:\n%s", rec.Body.String())
	}
}

// flashCounter is an App.Flash that counts its calls and renders a marker.
func flashCounter(calls *int) func(http.ResponseWriter, *http.Request) templ.Component {
	return func(w http.ResponseWriter, r *http.Request) templ.Component {
		*calls++
		http.SetCookie(w, &http.Cookie{Name: "wf_flash", Value: "", MaxAge: -1, Path: "/"})
		return pageBody("flash-banner")
	}
}

// flashInsideSlot reports whether the flash marker sits inside #wf-flash
// and before the page body.
func flashInsideSlot(got string) bool {
	slot := strings.Index(got, `<div id="wf-flash">`)
	marker := strings.Index(got, "flash-banner")
	body := strings.Index(got, "hello page")
	if slot < 0 || marker < 0 || body < 0 {
		return false
	}
	close := slot + strings.Index(got[slot:], "</div>")
	return slot < marker && marker < close && close < body
}

func TestFlash_CalledOncePerFullRender(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"plain":           nil,
		"boosted":         {"HX-Request": "true", "HX-Boosted": "true"},
		"history-restore": {"HX-Request": "true", "HX-History-Restore-Request": "true"},
	} {
		calls := 0
		a := testApp()
		a.Flash = flashCounter(&calls)
		got := serveHeaders(t, a, headers)
		if calls != 1 {
			t.Errorf("%s: Flash called %d times, want 1", name, calls)
		}
		if !flashInsideSlot(got) {
			t.Errorf("%s: flash not rendered inside #wf-flash above the body:\n%s", name, got)
		}
	}
}

func TestFlash_SetsCookieBeforeBody(t *testing.T) {
	calls := 0
	a := testApp()
	a.Flash = flashCounter(&calls)
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Render(w, r, "Movies", pageBody("hello page")); err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/movies", nil))
	// Result() uses the header snapshot taken at the first write; a cookie
	// set after that point would be missing here.
	if sc := rec.Result().Header.Get("Set-Cookie"); !strings.Contains(sc, "wf_flash=") {
		t.Errorf("Set-Cookie written by Flash did not make it into the response headers: %q", sc)
	}
}

// TestFlash_BodyOnlyRenderSendsOOB: a plain htmx body render consumes the
// flash too and appends it out of band into #wf-flash, after the body, so
// a flash set by an htmx action shows on the current page.
func TestFlash_BodyOnlyRenderSendsOOB(t *testing.T) {
	calls := 0
	a := testApp()
	a.Flash = flashCounter(&calls)
	got := serve(t, a, true)
	if calls != 1 {
		t.Errorf("Flash called %d times for a plain htmx body render, want 1", calls)
	}
	oob := strings.Index(got, `<div id="wf-flash" hx-swap-oob="true">`)
	body := strings.Index(got, "hello page")
	marker := strings.Index(got, "flash-banner")
	if oob < 0 || marker < oob || body < 0 || oob < body {
		t.Errorf("want body, then the oob #wf-flash holding the flash:\n%s", got)
	}
	if strings.Contains(got, "wf-shell") {
		t.Errorf("shell leaked into body-only render:\n%s", got)
	}
	// No flash: no oob slot at all.
	a = testApp()
	a.Flash = func(http.ResponseWriter, *http.Request) templ.Component { return nil }
	if got := serve(t, a, true); strings.Contains(got, "wf-flash") {
		t.Errorf("empty flash added an oob slot:\n%s", got)
	}
}

func TestFlash_PartRenderSendsOOB(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"targeted-part": {"HX-Request": "true", "HX-Target": "list-results"},
		"other-target":  {"HX-Request": "true", "HX-Target": "something-else"},
	} {
		calls := 0
		a := testApp()
		a.Flash = flashCounter(&calls)
		got := serveParts(t, a, headers)
		if calls != 1 {
			t.Errorf("%s: Flash called %d times, want 1", name, calls)
		}
		if !strings.Contains(got, `<div id="wf-flash" hx-swap-oob="true">`) || !strings.Contains(got, "flash-banner") {
			t.Errorf("%s: flash oob missing from part render:\n%s", name, got)
		}
	}
	calls := 0
	a := testApp()
	a.Flash = flashCounter(&calls)
	got := serveParts(t, a, map[string]string{"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "list-results"})
	if calls != 1 {
		t.Errorf("boosted RenderParts: Flash called %d times, want 1", calls)
	}
	if !strings.Contains(got, "flash-banner") {
		t.Errorf("boosted RenderParts: flash missing from the document:\n%s", got)
	}
}

func TestFlash_NotCalledByRenderBare(t *testing.T) {
	calls := 0
	a := testApp()
	a.Flash = flashCounter(&calls)
	got := serveBare(t, a, nil).Body.String()
	if calls != 0 {
		t.Errorf("Flash called %d times by RenderBare, want 0", calls)
	}
	if strings.Contains(got, "flash-banner") {
		t.Errorf("flash leaked into bare render:\n%s", got)
	}
}

func TestFlash_NilRendersEmptySlot(t *testing.T) {
	a := testApp()
	if a.Flash != nil {
		t.Fatal("test precondition: Flash nil")
	}
	got := serve(t, a, false)
	if !strings.Contains(got, `<div id="wf-flash"></div>`) {
		t.Errorf("missing empty #wf-flash slot in:\n%s", got)
	}
	if strings.Count(got, `id="wf-flash"`) != 1 {
		t.Errorf("want exactly one #wf-flash, got %d", strings.Count(got, `id="wf-flash"`))
	}
}

func TestFlash_NilResultRendersEmptySlot(t *testing.T) {
	a := testApp()
	a.Flash = func(http.ResponseWriter, *http.Request) templ.Component { return nil }
	got := serve(t, a, false)
	if !strings.Contains(got, `<div id="wf-flash"></div>`) {
		t.Errorf("missing empty #wf-flash slot in:\n%s", got)
	}
}

// TestRenderContentStatus: a full-page request renders as RenderStatus does
// (the shell's <main> carries MainID); a plain htmx request is retargeted
// at that main and gets its inner content, flash slot first.
func TestRenderContentStatus(t *testing.T) {
	a := &App{Name: "Demo", Flash: func(http.ResponseWriter, *http.Request) templ.Component { return templ.Raw("<p>FLASH</p>") }}
	body := templ.Raw("<p>BODY</p>")
	run := func(hx bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("DELETE", "/x", nil)
		if hx {
			req.Header.Set("HX-Request", "true")
		}
		rec := httptest.NewRecorder()
		if err := a.RenderContentStatus(rec, req, 404, "Gone", body); err != nil {
			t.Fatal(err)
		}
		return rec
	}
	full := run(false)
	if full.Code != 404 || full.Header().Get("HX-Retarget") != "" || !strings.Contains(full.Body.String(), `<main id="wf-main"`) {
		t.Errorf("full page: %d %q\n%s", full.Code, full.Header().Get("HX-Retarget"), full.Body.String())
	}
	frag := run(true)
	h := frag.Result().Header
	if frag.Code != 404 || h.Get("HX-Retarget") != "#wf-main" || h.Get("HX-Reswap") != ContentReswap {
		t.Errorf("fragment headers: %d %v", frag.Code, h)
	}
	if got := frag.Body.String(); got != `<div id="wf-flash"><p>FLASH</p></div><p>BODY</p>` {
		t.Errorf("fragment body = %q", got)
	}
}
