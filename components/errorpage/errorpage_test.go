package errorpage

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
)

func render(t *testing.T, ctx context.Context, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := ErrorPage(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestDefaultsPerCode(t *testing.T) {
	cases := []struct {
		code                   int
		digits, title, message string
	}{
		{404, ">404<", "Page not found", "does not exist or has moved"},
		{403, ">403<", "Not allowed", "You do not have access to this page."},
		{500, ">500<", "Something went wrong", "tell us the request id below"},
		{0, ">500<", "Something went wrong", "The server hit an error."},
		{418, ">418<", "Something went wrong", "The server hit an error."},
	}
	for _, c := range cases {
		out := render(t, context.Background(), Props{Code: c.code})
		for _, want := range []string{c.digits, c.title, c.message} {
			if !strings.Contains(out, want) {
				t.Errorf("code %d: missing %q in %s", c.code, want, out)
			}
		}
		if !strings.Contains(out, "<h1") {
			t.Errorf("code %d: title should be the page h1", c.code)
		}
	}
}

func TestCustomTitleAndMessageWin(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 404, Title: "Lost", Message: "Nothing here."})
	for _, want := range []string{"Lost", "Nothing here."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, unwanted := range []string{"Page not found", "does not exist"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("default %q should be overridden", unwanted)
		}
	}
}

func TestRequestIDOnlyWhenSet(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 500})
	if strings.Contains(out, "Request id") || strings.Contains(out, "font-mono") {
		t.Errorf("no request id should render when unset: %s", out)
	}
	out = render(t, context.Background(), Props{Code: 500, RequestID: "req-abc-123"})
	if !strings.Contains(out, "req-abc-123") || !strings.Contains(out, "font-mono") {
		t.Errorf("request id should render in mono: %s", out)
	}
}

func TestCodeUsesTabularNums(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 404})
	for _, want := range []string{"tabular-nums", "text-6xl", "max-w-md", "text-center"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing class %q", want)
		}
	}
}

func TestHomeDefaultsToSlashWithoutApp(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 404})
	if !strings.Contains(out, `href="/"`) || !strings.Contains(out, "Go home") {
		t.Errorf("want default Go home link to /: %s", out)
	}
}

func TestHomeDefaultsToAppHomeHref(t *testing.T) {
	app := &appshell.App{Name: "Demo", HomeHref: "/dash"}
	var got string
	h := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = render(t, r.Context(), Props{Code: 404})
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	if !strings.Contains(got, `href="/dash"`) {
		t.Errorf("want App.HomeHref: %s", got)
	}
}

func TestCustomHomeWins(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 404, Home: &action.Action{Label: "Back to items", Href: "/items"}})
	if !strings.Contains(out, `href="/items"`) || !strings.Contains(out, "Back to items") {
		t.Errorf("custom Home should render: %s", out)
	}
	if strings.Contains(out, "Go home") {
		t.Errorf("default label should not render with a custom Home")
	}
}

func TestHomeHrefSanitized(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 404, Home: &action.Action{Label: "x", Href: "javascript:alert(1)"}})
	if strings.Contains(out, "javascript:") {
		t.Errorf("javascript: href must be sanitized: %s", out)
	}
	if !strings.Contains(out, "about:invalid") {
		t.Errorf("want about:invalid: %s", out)
	}
}

func TestTextEscaped(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 500, Title: "<b>t</b>", Message: "<i>m</i>", RequestID: "<x>"})
	for _, raw := range []string{"<b>", "<i>", "<x>"} {
		if strings.Contains(out, raw) {
			t.Errorf("unescaped %q in %s", raw, out)
		}
	}
}

func TestDetailsRendered(t *testing.T) {
	out := render(t, context.Background(), Props{Code: 500, Details: templ.Raw("<pre>stack</pre>")})
	if !strings.Contains(out, "<pre>stack</pre>") {
		t.Errorf("Details should render: %s", out)
	}
}

// serve runs Respond under app's Middleware when app is non-nil.
func serve(t *testing.T, app *appshell.App, p Props, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Respond(w, r, p); err != nil {
			t.Errorf("Respond: %v", err)
		}
	})
	if app != nil {
		h = app.Middleware(h)
	}
	req := httptest.NewRequest("GET", "/missing", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func assertHeaders(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("status = %d, want %d", rec.Code, status)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Request") {
		t.Errorf("Vary = %q, want the htmx headers", v)
	}
}

func TestRespondWithApp(t *testing.T) {
	app := &appshell.App{Name: "Demo", HomeHref: "/dash", Nav: []appshell.NavItem{{Label: "Items", Href: "/items"}}}
	rec := serve(t, app, Props{Code: 404}, nil)
	assertHeaders(t, rec, http.StatusNotFound)
	out := rec.Body.String()
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(out)), "<!doctype") {
		t.Errorf("want full document")
	}
	for _, want := range []string{"<title>Page not found — Demo</title>", "Items", `href="/dash"`, "Page not found"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestRespondWithoutApp(t *testing.T) {
	rec := serve(t, nil, Props{Code: 500, RequestID: "r1"}, nil)
	assertHeaders(t, rec, http.StatusInternalServerError)
	out := rec.Body.String()
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(out)), "<!doctype") {
		t.Errorf("want full bare document")
	}
	if strings.Contains(out, "wf-shell") {
		t.Errorf("no shell without an App")
	}
	for _, want := range []string{"<title>Something went wrong</title>", "r1", `href="/"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestRespondFragmentForHX(t *testing.T) {
	rec := serve(t, nil, Props{Code: 403}, map[string]string{"HX-Request": "true"})
	assertHeaders(t, rec, http.StatusForbidden)
	if strings.Contains(strings.ToLower(rec.Body.String()), "<!doctype") {
		t.Errorf("plain htmx request should get the body only")
	}
}

func TestRespondDefaultStatus500(t *testing.T) {
	rec := serve(t, nil, Props{}, nil)
	assertHeaders(t, rec, http.StatusInternalServerError)
}

func TestShorthands(t *testing.T) {
	cases := []struct {
		name string
		fn   func(http.ResponseWriter, *http.Request) error
		code int
		want string
	}{
		{"NotFound", NotFound, 404, "Page not found"},
		{"Forbidden", Forbidden, 403, "Not allowed"},
		{"Internal", func(w http.ResponseWriter, r *http.Request) error { return Internal(w, r, "req-9") }, 500, "req-9"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		if err := c.fn(rec, httptest.NewRequest("GET", "/", nil)); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		assertHeaders(t, rec, c.code)
		if !strings.Contains(rec.Body.String(), c.want) {
			t.Errorf("%s: missing %q", c.name, c.want)
		}
	}
}
