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
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Render(w, r, "Movies", pageBody("hello page")); err != nil {
			t.Errorf("render: %v", err)
		}
	}))
	req := httptest.NewRequest("GET", "/movies", nil)
	if hx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
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
