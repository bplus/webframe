package appshell

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/commandpalette"
)

const triggerMarker = `data-wf-palette-trigger="wf-palette"`

func palette() *commandpalette.Props {
	return &commandpalette.Props{Items: []commandpalette.Item{
		{Label: "Home", Href: "/"},
		{Label: "Movies", Href: "/movies", Group: "Browse"},
	}}
}

func TestShell_PaletteTriggerOnlyWhenSet(t *testing.T) {
	for _, layout := range []Layout{LayoutTopbar, LayoutSidebar} {
		got := renderShell(t, Props{AppName: "A", Layout: layout}, nil)
		if strings.Contains(got, "data-wf-palette-trigger") {
			t.Errorf("%q: trigger rendered without Props.Palette:\n%s", layout, got)
		}
		got = renderShell(t, Props{AppName: "A", Layout: layout, Palette: palette()}, nil)
		// Topbar: one trigger. Sidebar: two, the footer copy for md and up
		// and a phone-only copy on the "Menu" row (the footer is folded
		// away below md), each shown by CSS at its own widths.
		want := 1
		if layout == LayoutSidebar {
			want = 2
		}
		if n := strings.Count(got, triggerMarker); n != want {
			t.Errorf("%q: want %d trigger(s), got %d in:\n%s", layout, want, n, got)
		}
		// The trigger is a button, not the palette: Shell never renders the dialog.
		if strings.Contains(got, "data-tui-dialog-content") {
			t.Errorf("%q: Shell rendered the palette dialog itself:\n%s", layout, got)
		}
	}
	// Topbar: the trigger sits in the right controls; sidebar: in the footer.
	got := renderShell(t, Props{AppName: "A", Palette: palette()}, nil)
	right := strings.Index(got, `class="wf-topbar-right"`)
	trig := strings.Index(got, triggerMarker)
	if right < 0 || trig < right {
		t.Errorf("topbar trigger not inside wf-topbar-right (right=%d trig=%d):\n%s", right, trig, got)
	}
	got = renderShell(t, Props{AppName: "A", Layout: LayoutSidebar, Palette: palette()}, nil)
	phone := strings.Index(got, `class="wf-sidebar-phone-actions"`)
	footer := strings.Index(got, `class="wf-sidebar-footer"`)
	first := strings.Index(got, triggerMarker)
	last := strings.LastIndex(got, triggerMarker)
	if phone < 0 || first < phone || footer < 0 || last < footer {
		t.Errorf("sidebar triggers: want one after wf-sidebar-phone-actions (%d) and one after wf-sidebar-footer (%d), got %d and %d:\n%s", phone, footer, first, last, got)
	}
}

func TestDocument_PaletteOnceBeforeToastContainer(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, templ.Raw("<p>hi</p>"))
	if strings.Contains(got, "data-wf-palette") {
		t.Errorf("palette rendered without DocProps.Palette:\n%s", got)
	}
	got = renderDoc(t, DocProps{AppName: "A", Palette: palette()}, templ.Raw("<p>hi</p>"))
	const root = `data-dialog-instance="wf-palette"`
	// The root div, the backdrop and the panel each carry the instance id.
	if n := strings.Count(got, root); n != 3 {
		t.Fatalf("want exactly one palette (3 instance attrs), got %d in:\n%s", n, got)
	}
	if n := strings.Count(got, `id="wf-palette"`); n != 1 {
		t.Errorf("want one #wf-palette, got %d", n)
	}
	child := strings.Index(got, "<p>hi</p>")
	pal := strings.Index(got, `id="wf-palette"`)
	toast := strings.Index(got, `id="toast-container"`)
	confirm := strings.Index(got, `id="wf-confirm"`)
	if !(child < pal && pal < toast && toast < confirm) {
		t.Errorf("want children < palette < toast-container < wf-confirm; got %d %d %d %d", child, pal, toast, confirm)
	}
	for _, want := range []string{`data-tui-dialog-hidden="true"`, `role="listbox"`, `href="/movies"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestApp_RenderWiresPalette(t *testing.T) {
	app := &App{Name: "A", Palette: palette()}
	body := templ.Raw("<p>page</p>")
	srv := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := app.Render(w, r, "T", body); err != nil {
			t.Fatalf("render: %v", err)
		}
	}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	got := rec.Body.String()
	if n := strings.Count(got, triggerMarker); n != 1 {
		t.Errorf("want one trigger, got %d in:\n%s", n, got)
	}
	if n := strings.Count(got, `id="wf-palette"`); n != 1 {
		t.Errorf("want one palette, got %d in:\n%s", n, got)
	}

	// A fragment response carries neither.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "true")
	srv.ServeHTTP(rec, req)
	if got := rec.Body.String(); strings.Contains(got, "wf-palette") {
		t.Errorf("fragment should not carry the palette:\n%s", got)
	}

	// RenderBare (no shell) has no trigger and no palette.
	rec = httptest.NewRecorder()
	if err := app.RenderBare(rec, httptest.NewRequest(http.MethodGet, "/login", nil), "Sign in", body); err != nil {
		t.Fatalf("render bare: %v", err)
	}
	if got := rec.Body.String(); strings.Contains(got, "wf-palette") {
		t.Errorf("RenderBare should not carry the palette:\n%s", got)
	}

	// Without App.Palette nothing palette-related renders.
	plain := &App{Name: "A"}
	rec = httptest.NewRecorder()
	plain.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := plain.Render(w, r, "T", body); err != nil {
			t.Fatalf("render: %v", err)
		}
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Body.String(); strings.Contains(got, "wf-palette") {
		t.Errorf("palette rendered without App.Palette:\n%s", got)
	}
}

// TestApp_PaletteFor: PaletteFor builds the palette per request and wins
// over Palette; it is called exactly once per shell render and its result
// feeds both the trigger and the dialog; two requests get their own item
// sets; a nil result renders no palette; a fragment render never calls it.
func TestApp_PaletteFor(t *testing.T) {
	calls := 0
	app := &App{
		Name:    "A",
		Palette: palette(), // ignored while PaletteFor is set
		PaletteFor: func(r *http.Request) *commandpalette.Props {
			calls++
			if r.URL.Path == "/none" {
				return nil
			}
			return &commandpalette.Props{Items: []commandpalette.Item{{Label: "Only " + r.URL.Query().Get("q"), Href: "/" + r.URL.Query().Get("q")}}}
		},
	}
	body := templ.Raw("<p>page</p>")
	srv := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := app.Render(w, r, "T", body); err != nil {
			t.Fatalf("render: %v", err)
		}
	}))
	get := func(target string, hx bool) string {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if hx {
			req.Header.Set("HX-Request", "true")
		}
		srv.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	first := get("/?q=alpha", false)
	if calls != 1 {
		t.Fatalf("PaletteFor called %d times for one shell render, want 1", calls)
	}
	second := get("/?q=beta", false)
	if calls != 2 {
		t.Fatalf("PaletteFor called %d times for two shell renders, want 2", calls)
	}
	for _, c := range []struct{ body, want, not string }{{first, "alpha", "beta"}, {second, "beta", "alpha"}} {
		if !strings.Contains(c.body, `data-label="Only `+c.want+`"`) || strings.Contains(c.body, "Only "+c.not) {
			t.Errorf("want only the %s item set:\n%s", c.want, c.body)
		}
		// Palette (the static one) is ignored: none of its items render.
		if strings.Contains(c.body, `href="/movies"`) {
			t.Errorf("static Palette rendered although PaletteFor is set:\n%s", c.body)
		}
		if n := strings.Count(c.body, triggerMarker); n != 1 {
			t.Errorf("want one trigger, got %d", n)
		}
		if n := strings.Count(c.body, `id="wf-palette"`); n != 1 {
			t.Errorf("want one palette, got %d", n)
		}
	}
	// nil result: neither trigger nor dialog.
	if got := get("/none", false); strings.Contains(got, "wf-palette") {
		t.Errorf("nil PaletteFor result still rendered a palette:\n%s", got)
	}
	// A fragment render has no shell and does not call PaletteFor.
	before := calls
	if got := get("/?q=gamma", true); strings.Contains(got, "wf-palette") {
		t.Errorf("fragment carries the palette:\n%s", got)
	}
	if calls != before {
		t.Errorf("PaletteFor called on a fragment render")
	}
}
