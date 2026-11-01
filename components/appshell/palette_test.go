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
		if n := strings.Count(got, triggerMarker); n != 1 {
			t.Errorf("%q: want one trigger, got %d in:\n%s", layout, n, got)
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
	footer := strings.Index(got, `class="wf-sidebar-footer"`)
	trig = strings.Index(got, triggerMarker)
	if footer < 0 || trig < footer {
		t.Errorf("sidebar trigger not inside wf-sidebar-footer (footer=%d trig=%d):\n%s", footer, trig, got)
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
