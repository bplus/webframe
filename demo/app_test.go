package demo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bplus/webframe/assets"
)

func TestShowcaseRendersThroughShell(t *testing.T) {
	mux := http.NewServeMux()
	assets.Mount(mux)
	mux.HandleFunc("GET /{$}", HandleShowcase)
	RegisterItemRoutes(mux)
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

	for _, path := range []string{"/"} {
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

	// The showcase sections for the page-layer components: each
	// new component's signature markup is on the page.
	_, body := get("/")
	for _, want := range []string{
		`role="switch"`,                                         // toggle field
		`hx-post="/api/validate"`,                               // Validate wiring
		`id="field-email-feedback"`,                             // its feedback slot
		`id="pick-form"`,                                        // form.Form
		`name="source" value="showcase"`,                        // its Hidden entry
		`data-wf-ac-required`,                                   // required autocomplete
		`<figure`,                                               // figure
		`src="/static/img/placeholder-wide.svg"`,                // image, local art
		`loading="lazy"`,                                        // image default
		`role="toolbar"`,                                        // toolbar
		`aria-label="More actions"`,                             // its phone menu
		`aria-label="Pagination"`,                               // pagination
		"41–50 of 200 items",                                    // page 5 of 20 at 10 a page
		"0 items",                                               // the zero-row summary
		`aria-sort="ascending"`,                                 // datatable sorted column
		`aria-label="Select all rows"`,                          // its selection header
		"No items match",                                        // the empty table's state
		`<dl`,                                                   // descriptionlist
		`role="alert"`,                                          // the error banner
		`href="/items"`, `href="/items/1"`, `href="/items/new"`, // pages panel
		"Weird",       // badge.StatusLabel on an unknown word
		"items-start", // figure AlignStart
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /: body missing %q", want)
		}
	}
	if code, _ := get("/static/img/placeholder-wide.svg"); code != 200 {
		t.Errorf("GET /static/img/placeholder-wide.svg = %d, want 200", code)
	}
	// The Items app renders through SidebarApp: the sidebar shell with its
	// MINE and DISCOVER groups, the Items entry active.
	code, body := get("/items")
	if code != 200 {
		t.Errorf("GET /items = %d, want 200", code)
	}
	for _, want := range []string{"wf-shell--sidebar", `<aside class="wf-sidebar`, "Mine", "Discover", "Trending", "Builds", "Items — Webframe</title>", `aria-current="page"`} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /items: body missing %q", want)
		}
	}
	// The topbar pages link across to it.
	if _, body := get("/"); !strings.Contains(body, `href="/items"`) {
		t.Error("GET /: topbar has no Items link")
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
		// Validate posts the control's own name=value, nothing else.
		{"POST", "/api/validate?email=nope", []string{`id="field-email-feedback"`, `role="alert"`, "Enter a valid email address"}, []string{"Looks good"}},
		{"POST", "/api/validate?email=", []string{`id="field-email-feedback"`, "Email is required"}, nil},
		{"POST", "/api/validate?email=a@b.c", []string{`id="field-email-feedback"`}, []string{"Looks good", "<p"}},
		{"POST", "/api/validate?name=B", []string{`id="field-name-feedback"`, "Name must be at least 2 characters"}, nil},
		{"POST", "/api/validate?name=Bo", []string{`id="field-name-feedback"`}, []string{"Looks good", "<p"}},
		{"POST", "/api/picked?framework=vue&q=Vue&source=showcase", []string{"framework=vue", "q=Vue"}, []string{"nothing picked"}},
		{"POST", "/api/picked?q=Vu", []string{"framework=", "nothing picked"}, nil},
		// /api/toast is htmx-only (toast.Send); feedback_test.go covers it.
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

	// A validate request naming neither field has no Feedback slot to
	// answer with, so it is refused rather than swapped.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/validate?other=x", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /api/validate?other=x = %d, want 400", rec.Code)
	}
}

// TestFilterDemo: the showcase's Filter bar section renders a bar with a
// 16-option single-choice group, and its endpoint answers the results
// region's content for a search and a school.
func TestFilterDemo(t *testing.T) {
	h := Handler()
	body := do(h, "GET", "/", nil, nil).Body.String()
	wantAll(t, body, `id="filter-demo"`, `data-tui-filterbar-hx-get="/api/filter-demo"`, `id="filter-demo-results"`,
		`<div id="filter-demo-control-school" role="group" aria-label="School" data-tui-filterbar-choice`, "46</strong> of 46 spells")
	if n := strings.Count(body, `name="school"`); n != 17 {
		t.Errorf("%d School buttons, want 17 (All and 16 schools)", n)
	}
	rec := do(h, "GET", "/api/filter-demo?fq=fire&school=evocation", nil, hx("filter-demo-results"))
	if rec.Code != 200 {
		t.Fatalf("GET /api/filter-demo = %d", rec.Code)
	}
	body = rec.Body.String()
	wantAll(t, body, "3</strong> of 46 spells", "Fire Bolt", "Fireball", "Wall of Fire",
		// The School group again, out of band, with the counts of the search
		// and the chosen school selected; the search box is not sent.
		`<div id="filter-demo-control-school" role="group" aria-label="School" data-tui-filterbar-choice hx-swap-oob="true"`,
		`name="school" value="evocation" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="true" aria-pressed="true"`,
		`>Runes <span data-tui-filterbar-count class="ml-0.5 tabular-nums opacity-70">1</span>`)
	wantNone(t, body, "Fire Shape", "<html", `type="search"`)
}
