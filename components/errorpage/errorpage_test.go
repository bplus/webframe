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
		{405, ">405<", "Not allowed here", "This address does not accept that request."},
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
	// rec.Result() holds the headers as they were sent (frozen at
	// WriteHeader); rec.Header() is the live map and hides a header set
	// too late.
	h := rec.Result().Header
	if ct := h.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if v := h.Get("Vary"); !strings.Contains(v, "HX-Request") {
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
		{"MethodNotAllowed", MethodNotAllowed, 405, "Not allowed here"},
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

// TestRespond_ClearsPendingFlash: an error page consumes a pending flash
// and its clearing cookie is sent even though the status is not 200 (the
// status is written after App.Flash ran).
func TestRespond_ClearsPendingFlash(t *testing.T) {
	cleared := false
	app := &appshell.App{Name: "Demo", HomeHref: "/", Flash: func(w http.ResponseWriter, r *http.Request) templ.Component {
		http.SetCookie(w, &http.Cookie{Name: "wf_flash", Value: "", MaxAge: -1, Path: "/"})
		cleared = true
		return templ.Raw("<p>FLASHBANNER</p>")
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/missing", nil)
	app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := NotFound(w, r); err != nil {
			t.Fatal(err)
		}
	})).ServeHTTP(rec, req)
	res := rec.Result()
	if res.StatusCode != 404 || !cleared {
		t.Fatalf("status %d, flash called %v", res.StatusCode, cleared)
	}
	var sawClear bool
	for _, c := range res.Cookies() {
		if c.Name == "wf_flash" && c.MaxAge < 0 {
			sawClear = true
		}
	}
	if !sawClear {
		t.Errorf("clearing Set-Cookie lost (headers frozen before Flash ran): %v", res.Header)
	}
	if !strings.Contains(rec.Body.String(), "FLASHBANNER") {
		t.Errorf("flash banner missing from the error page")
	}
}

// styledErrors is the package doc's ServeMux wrapper, verbatim, so the
// documented pattern is proven against the real mux.
func styledErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r) // a real route, or the mux's own redirect
			return
		}
		var allowed []string
		probe := r.Clone(r.Context())
		for _, m := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"} {
			probe.Method = m
			if _, pattern := mux.Handler(probe); pattern != "" {
				allowed = append(allowed, m)
			}
		}
		if len(allowed) == 0 {
			_ = NotFound(w, r)
			return
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		_ = MethodNotAllowed(w, r)
	})
}

// TestServeMuxWrapperRoutes405: Go's ServeMux answers a method mismatch
// with a plain-text 405 on its own; the documented wrapper turns it into
// the styled page (with Allow), leaves real routes alone and still gives
// an unknown path the 404 page.
func TestServeMuxWrapperRoutes405(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ITEMS")) })
	mux.HandleFunc("DELETE /items", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("GONE")) })

	// The bare mux: plain text, no page.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/items", nil))
	if rec.Code != 405 || strings.Contains(rec.Body.String(), "<") {
		t.Fatalf("bare mux: %d %q", rec.Code, rec.Body.String())
	}

	app := &appshell.App{Name: "Demo", HomeHref: "/"}
	h := app.Middleware(styledErrors(mux))
	do := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}

	rec = do("POST", "/items")
	assertHeaders(t, rec, http.StatusMethodNotAllowed)
	if allow := rec.Result().Header.Get("Allow"); allow != "GET, HEAD, DELETE" {
		t.Errorf("Allow = %q", allow)
	}
	for _, want := range []string{">405<", "Not allowed here", "This address does not accept that request.", "<title>Not allowed here — Demo</title>"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("missing %q in:\n%s", want, rec.Body.String())
		}
	}

	rec = do("GET", "/items")
	if rec.Code != 200 || rec.Body.String() != "ITEMS" {
		t.Errorf("real route: %d %q", rec.Code, rec.Body.String())
	}

	rec = do("GET", "/nowhere")
	assertHeaders(t, rec, http.StatusNotFound)
	if !strings.Contains(rec.Body.String(), "Page not found") || rec.Result().Header.Get("Allow") != "" {
		t.Errorf("unknown path: %d Allow=%q", rec.Code, rec.Result().Header.Get("Allow"))
	}
}
