package errorpage

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
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

// TestServeMuxWrapperRoutes405: Go's ServeMux answers a method mismatch
// with a plain-text 405 on its own; Handler turns it into the styled page
// (with Allow), leaves real routes alone and still gives an unknown path
// the 404 page.
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
	h := app.Middleware(Handler(mux))
	do := func(method, path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec
	}

	rec = do("POST", "/items")
	assertHeaders(t, rec, http.StatusMethodNotAllowed)
	// The mux's own Allow (sorted), copied over.
	if allow := rec.Result().Header.Get("Allow"); allow != "DELETE, GET, HEAD" {
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

// TestHandler_HeadAndOptions: a HEAD on a GET route is the route itself
// (the mux treats HEAD as GET), so it is never a 405; OPTIONS is probed
// and listed in Allow when registered.
func TestHandler_HeadAndOptions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ITEMS")) })
	mux.HandleFunc("OPTIONS /items", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	h := Handler(mux)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("HEAD", "/items", nil))
	if rec.Code != 200 {
		t.Errorf("HEAD on a GET route: %d, want 200 from the route", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("PUT", "/items", nil))
	if rec.Code != 405 {
		t.Fatalf("PUT: %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD, OPTIONS" {
		t.Errorf("Allow = %q, want GET, HEAD, OPTIONS", allow)
	}
}

// TestHandler_TrailingSlashEdgeMatchesMux: with only "GET /items/"
// registered, "POST /items" has no route of its own; whatever the bare
// mux answers (405 with Allow since the mux counts the trailing-slash
// form's methods, as Go 1.27 does) Handler must answer the same code with
// the same Allow, only styled. "GET /items" is the mux's own redirect to
// /items/ and passes through untouched. A wrong method under the subtree
// is a plain 405.
func TestHandler_TrailingSlashEdgeMatchesMux(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("TREE")) })

	bare := httptest.NewRecorder()
	mux.ServeHTTP(bare, httptest.NewRequest("POST", "/items", nil))
	if bare.Code != 404 && bare.Code != 405 {
		t.Fatalf("bare mux POST /items: %d", bare.Code)
	}

	h := Handler(mux)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/items", nil))
	if rec.Code != bare.Code || rec.Header().Get("Allow") != bare.Header().Get("Allow") {
		t.Errorf("Handler POST /items: %d Allow=%q; bare mux: %d Allow=%q", rec.Code, rec.Header().Get("Allow"), bare.Code, bare.Header().Get("Allow"))
	}
	want := map[int]string{404: "Page not found", 405: "Not allowed here"}[bare.Code]
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("want the styled %d page (%q), got:\n%s", bare.Code, want, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/items", nil))
	if rec.Code/100 != 3 || rec.Header().Get("Location") != "/items/" {
		t.Errorf("GET /items: %d Location=%q, want the mux's redirect to /items/", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/items/7", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("POST /items/7: %d Allow=%q, want 405 GET, HEAD", rec.Code, rec.Header().Get("Allow"))
	}
}

// TestHandler_UncleanPathRedirectPassesThrough: "//items" with the wrong
// method is a redirect to the cleaned path (an empty pattern from the mux,
// but not a 404 or 405), so the mux answers it.
func TestHandler_UncleanPathRedirectPassesThrough(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, _ *http.Request) {})
	rec := httptest.NewRecorder()
	Handler(mux).ServeHTTP(rec, httptest.NewRequest("POST", "//items", nil))
	if rec.Code/100 != 3 || rec.Header().Get("Location") != "/items" {
		t.Errorf("POST //items: %d Location=%q, want the mux's redirect to /items", rec.Code, rec.Header().Get("Location"))
	}
}

// TestHandler_NeverRunsAppHandlers: deciding 404 vs 405 runs only the
// mux's own answer, never an app handler for another method.
func TestHandler_NeverRunsAppHandlers(t *testing.T) {
	ran := 0
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /items/{id}", func(w http.ResponseWriter, _ *http.Request) { ran++ })
	rec := httptest.NewRecorder()
	Handler(mux).ServeHTTP(rec, httptest.NewRequest("GET", "/items/3", nil))
	if rec.Code != 405 || ran != 0 {
		t.Errorf("code %d, DELETE handler ran %d times", rec.Code, ran)
	}
}

// TestHandler_WithoutApp: outside App.Middleware the pages render bare,
// still with the right status and Allow.
func TestHandler_WithoutApp(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /x", func(w http.ResponseWriter, _ *http.Request) {})
	rec := httptest.NewRecorder()
	Handler(mux).ServeHTTP(rec, httptest.NewRequest("POST", "/x", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("%d Allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
	body := rec.Body.String()
	if !strings.Contains(strings.ToLower(body), "<!doctype html>") || strings.Contains(body, "wf-shell") {
		t.Errorf("want a bare document without the shell, got:\n%s", body)
	}
}

// TestPanicHandler_RendersInternal: wired as App.OnPanic, a panicking
// handler answers with the styled 500 page in the shell, showing the id
// Middleware generated or the X-Request-Id the request carried.
func TestPanicHandler_RendersInternal(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)

	app := &appshell.App{Name: "Demo", HomeHref: "/", OnPanic: PanicHandler}
	h := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest("GET", "/items", nil)
	req.Header.Set("X-Request-Id", "req_abc123")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assertHeaders(t, rec, http.StatusInternalServerError)
	for _, want := range []string{">500<", "Something went wrong", "Request id: req_abc123", "wf-shell", "<title>Something went wrong — Demo</title>"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("missing %q in:\n%s", want, rec.Body.String())
		}
	}
	if !strings.Contains(logs.String(), "appshell: panic serving GET /items (request req_abc123): boom") {
		t.Errorf("log line missing, got:\n%s", logs.String())
	}
}

// serveMethod is serve with a request method and an App.Flash that always
// returns a banner.
func serveMethod(t *testing.T, withApp bool, method string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := NotFound(w, r); err != nil {
			t.Errorf("NotFound: %v", err)
		}
	})
	if withApp {
		app := &appshell.App{Name: "Demo", HomeHref: "/", Flash: func(http.ResponseWriter, *http.Request) templ.Component {
			return templ.Raw("<p>FLASHBANNER</p>")
		}}
		h = app.Middleware(h)
	}
	req := httptest.NewRequest(method, "/items/7", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestRespond_ActionOnItselfLandsInMain: an htmx action with no hx-target
// (a Delete button in a page header) targets itself. The error page must
// not be pasted inside the button: it replaces the shell's main content
// region, with a fresh #wf-flash slot holding the pending flash inline (an
// out-of-band copy would land in the slot the swap removes).
func TestRespond_ActionOnItselfLandsInMain(t *testing.T) {
	cases := map[string]struct {
		method  string
		headers map[string]string
	}{
		"delete, target has no id":      {"DELETE", map[string]string{"HX-Request": "true"}},
		"delete, trigger has an id":     {"DELETE", map[string]string{"HX-Request": "true", "HX-Trigger": "del"}},
		"post, form targets itself":     {"POST", map[string]string{"HX-Request": "true", "HX-Trigger": "edit", "HX-Target": "edit"}},
		"patch, row targets itself": {"PATCH", map[string]string{"HX-Request": "true", "HX-Trigger": "row-3", "HX-Target": "row-3"}},
	}
	for name, c := range cases {
		rec := serveMethod(t, true, c.method, c.headers)
		h := rec.Result().Header
		if rec.Code != 404 {
			t.Errorf("%s: status %d", name, rec.Code)
		}
		if got := h.Get("HX-Retarget"); got != "#"+appshell.MainID {
			t.Errorf("%s: HX-Retarget = %q, want #%s", name, got, appshell.MainID)
		}
		if got := h.Get("HX-Reswap"); !strings.HasPrefix(got, "innerHTML") {
			t.Errorf("%s: HX-Reswap = %q, want innerHTML", name, got)
		}
		if h.Get("HX-Push-Url") != "" {
			t.Errorf("%s: an error must push no URL", name)
		}
		body := rec.Body.String()
		slot := strings.Index(body, `<div id="wf-flash">`)
		banner := strings.Index(body, "FLASHBANNER")
		page := strings.Index(body, "Page not found")
		if slot != 0 || banner < slot || page < banner {
			t.Errorf("%s: want the #wf-flash slot holding the flash, then the page:\n%s", name, body)
		}
		if strings.Contains(body, "hx-swap-oob") || strings.Contains(body, "<main") {
			t.Errorf("%s: want main's inner content only, no oob flash:\n%s", name, body)
		}
	}
}

// TestRespond_RegionTargetsKeepTheirTarget: an htmx request aimed at a
// named region other than the trigger (a list's results, a split pane's
// detail), any GET (lazy tabs, polls, which data-wf-swallow-errors covers)
// and a full-page request are answered exactly as before: no retarget.
func TestRespond_RegionTargetsKeepTheirTarget(t *testing.T) {
	cases := map[string]struct {
		method  string
		headers map[string]string
	}{
		"delete into results region": {"DELETE", map[string]string{"HX-Request": "true", "HX-Trigger": "row-menu", "HX-Target": "items-results"}},
		"delete, region, no trigger": {"DELETE", map[string]string{"HX-Request": "true", "HX-Target": "items-results"}},
		"get on itself":              {"GET", map[string]string{"HX-Request": "true"}},
		"get into detail region":     {"GET", map[string]string{"HX-Request": "true", "HX-Target": "mail-detail"}},
		"plain delete":               {"DELETE", nil},
		"boosted post":               {"POST", map[string]string{"HX-Request": "true", "HX-Boosted": "true"}},
	}
	for name, c := range cases {
		rec := serveMethod(t, true, c.method, c.headers)
		h := rec.Result().Header
		if h.Get("HX-Retarget") != "" || h.Get("HX-Reswap") != "" {
			t.Errorf("%s: unexpected retarget %q / %q", name, h.Get("HX-Retarget"), h.Get("HX-Reswap"))
		}
		if !strings.Contains(rec.Body.String(), "Page not found") {
			t.Errorf("%s: page missing", name)
		}
	}
}

// TestRespond_ActionOnItselfWithoutApp: outside App.Middleware there is no
// shell and no main region, so the page replaces the body.
func TestRespond_ActionOnItselfWithoutApp(t *testing.T) {
	rec := serveMethod(t, false, "DELETE", map[string]string{"HX-Request": "true"})
	h := rec.Result().Header
	if h.Get("HX-Retarget") != "body" || !strings.HasPrefix(h.Get("HX-Reswap"), "innerHTML") {
		t.Errorf("want body/innerHTML, got %q / %q", h.Get("HX-Retarget"), h.Get("HX-Reswap"))
	}
	if !strings.Contains(rec.Body.String(), "Page not found") {
		t.Errorf("page missing: %s", rec.Body.String())
	}
}
