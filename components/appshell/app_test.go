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
	if v := rec.Header().Get("Vary"); v != "HX-Request, HX-Boosted, HX-History-Restore-Request" {
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
	if v := rec.Header().Get("Vary"); v != "HX-Request, HX-Boosted, HX-History-Restore-Request" {
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
