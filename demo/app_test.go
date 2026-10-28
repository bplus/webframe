package demo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bplus/webframe/assets"
)

func TestShowcaseAndTrafficRenderThroughShell(t *testing.T) {
	mux := http.NewServeMux()
	assets.Mount(mux)
	mux.HandleFunc("GET /{$}", HandleShowcase)
	mux.HandleFunc("GET /traffic", HandleTraffic)
	mux.HandleFunc("GET /sidebar", HandleSidebar)
	srv := httptest.NewServer(App.Middleware(mux))
	defer srv.Close()

	get := func(path string) (int, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return resp.StatusCode, string(b)
	}

	for _, path := range []string{"/", "/traffic"} {
		code, body := get(path)
		if code != 200 {
			t.Errorf("GET %s = %d, want 200", path, code)
		}
		for _, want := range []string{"wf-topbar", " — Webframe</title>"} {
			if !strings.Contains(body, want) {
				t.Errorf("GET %s: body missing %q", path, want)
			}
		}
	}
	code, body := get("/sidebar")
	if code != 200 {
		t.Errorf("GET /sidebar = %d, want 200", code)
	}
	for _, want := range []string{"wf-shell--sidebar", `<aside class="wf-sidebar`, "Mine", "Discover", "Trending", "Sidebar — Webframe</title>", `aria-current="page"`} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /sidebar: body missing %q", want)
		}
	}
	if code, _ := get("/static/css/output.css"); code != 200 {
		t.Errorf("GET /static/css/output.css = %d, want 200", code)
	}
}

// TestInteractionFragments: the htmx demo endpoints answer with rendered
// components (no full page), carrying the hx wiring each demo relies on.
func TestInteractionFragments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/click", HandleClick)
	mux.HandleFunc("GET /api/search", HandleSearch)
	mux.HandleFunc("POST /api/validate", HandleValidate)
	mux.HandleFunc("POST /api/toast", HandleToast)
	mux.HandleFunc("POST /api/picked", HandlePicked)
	h := App.Middleware(mux)

	for _, c := range []struct {
		method, target string
		want, notWant  []string
	}{
		{"POST", "/api/click?count=1", []string{"Clicked 2 times", `hx-post="/api/click"`, `{&#34;count&#34;:&#34;2&#34;}`}, nil},
		{"GET", "/api/search?q=re", []string{"React", "Remix", "Express", `role="option"`, `data-wf-ac-item-value="react"`}, []string{"Vue", "No matches"}},
		{"GET", "/api/search?q=next", []string{`data-wf-ac-item-label="Next.js"`, `data-wf-ac-item-value="next-js"`}, nil},
		{"GET", "/api/search?q=zzz", []string{"No matches"}, []string{"React"}},
		{"POST", "/api/validate?field=email&value=nope", []string{`id="field-email-feedback"`, "Enter a valid email address"}, []string{"Looks good"}},
		{"POST", "/api/validate?field=email&value=a@b.c", []string{`id="field-email-feedback"`}, []string{"Looks good", "<p"}},
		{"POST", "/api/validate?field=name&value=Bo", []string{`id="field-name-feedback"`}, []string{"Looks good", "<p"}},
		{"POST", "/api/picked?framework=vue&q=Vue", []string{"framework=vue"}, []string{"nothing picked"}},
		{"POST", "/api/picked?q=Vu", []string{"framework=", "nothing picked"}, nil},
		{"POST", "/api/toast?variant=error", []string{`hx-swap-oob="beforeend:#toast-container"`, "Something went wrong", "x-init="}, nil},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.target, nil))
		body := rec.Body.String()
		if rec.Code != 200 {
			t.Errorf("%s %s = %d", c.method, c.target, rec.Code)
		}
		for _, w := range c.want {
			if !strings.Contains(body, w) {
				t.Errorf("%s %s: missing %q in:\n%s", c.method, c.target, w, body)
			}
		}
		for _, w := range append(c.notWant, "<html", "wf-topbar") {
			if strings.Contains(body, w) {
				t.Errorf("%s %s: unexpected %q in:\n%s", c.method, c.target, w, body)
			}
		}
	}
}
