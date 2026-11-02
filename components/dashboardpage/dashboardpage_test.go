package dashboardpage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/stattile"
)

func text(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Page(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// inOrder fails unless each needle appears after the previous one.
func inOrder(t *testing.T, out string, needles ...string) {
	t.Helper()
	pos := 0
	for _, n := range needles {
		i := strings.Index(out[pos:], n)
		if i < 0 {
			t.Fatalf("missing %q after offset %d in:\n%s", n, pos, out)
		}
		pos += i + len(n)
	}
}

func dashboard() Props {
	return Props{
		ID:     "dash",
		Header: pageheader.Props{Title: "Overview", Subtitle: "The forge at a glance"},
		Tiles: []Tile{
			{Props: stattile.Props{Label: "Items", Value: "42"}},
			{Props: stattile.Props{Label: "Published", Value: "11"}},
			{Props: stattile.Props{Label: "Drafts", Value: "3"}},
			{Props: stattile.Props{Label: "Packs", Value: "7"}},
		},
		Sections: []Section{
			{ID: "recent", Title: "Recent changes", Description: "Last 24 hours", Content: text("<p>RECENT CONTENT</p>")},
			{ID: "newest", Title: "Newest items", Content: text("<p>NEWEST CONTENT</p>")},
		},
	}
}

func TestPage_HeaderPresent(t *testing.T) {
	out := render(t, dashboard())
	if !strings.Contains(out, "<header") || !strings.Contains(out, "<h1") || !strings.Contains(out, "Overview") {
		t.Errorf("missing pageheader h1: %s", out)
	}
	if !strings.Contains(out, "The forge at a glance") {
		t.Errorf("missing subtitle: %s", out)
	}
	inOrder(t, out, "<h1", "Overview", "wf-dashgrid")
}

func TestPage_TileSpansDefaultThree(t *testing.T) {
	out := render(t, dashboard())
	if n := strings.Count(out, "grid-column:span 3;"); n != 4 {
		t.Errorf("want 4 tile slots spanning 3, got %d: %s", n, out)
	}
	for _, label := range []string{"Items", "Published", "Drafts", "Packs"} {
		if !strings.Contains(out, label) {
			t.Errorf("missing tile %q", label)
		}
	}
}

func TestPage_TileSpanExplicit(t *testing.T) {
	p := Props{Tiles: []Tile{
		{Props: stattile.Props{Value: "1"}, ColSpan: 6},
		{Props: stattile.Props{Value: "2"}, ColSpan: 4},
		{Props: stattile.Props{Value: "3"}},
	}}
	out := render(t, p)
	inOrder(t, out, "grid-column:span 6;", "grid-column:span 4;", "grid-column:span 3;")
}

func TestPage_SectionSpansDefaultTwelve(t *testing.T) {
	out := render(t, dashboard())
	if n := strings.Count(out, "grid-column:span 12;"); n != 2 {
		t.Errorf("want 2 section slots spanning 12, got %d: %s", n, out)
	}
}

func TestPage_SectionSpanExplicit(t *testing.T) {
	p := Props{Sections: []Section{
		{Title: "Wide", ColSpan: 8},
		{Title: "Narrow", ColSpan: 4},
	}}
	out := render(t, p)
	inOrder(t, out, "grid-column:span 8;", "Wide", "grid-column:span 4;", "Narrow")
	if strings.Contains(out, "grid-column:span 12;") {
		t.Errorf("explicit spans must not fall back to 12: %s", out)
	}
}

func TestPage_SectionHeadingLevel(t *testing.T) {
	out := render(t, dashboard())
	if n := strings.Count(out, "<h2"); n != 2 {
		t.Errorf("sections default to h2; want 2, got %d: %s", n, out)
	}
	if strings.Contains(out, "<h3") {
		t.Errorf("no h3 expected by default: %s", out)
	}

	p := Props{Sections: []Section{{Title: "Nested", Level: 3}, {Title: "Deeper", Level: 4}}}
	out = render(t, p)
	inOrder(t, out, "<h3", "Nested", "<h4", "Deeper")
	if strings.Contains(out, "<h2") {
		t.Errorf("explicit Level must not render h2: %s", out)
	}
}

func TestPage_SectionContentIDAndActions(t *testing.T) {
	p := dashboard()
	p.Sections[0].Actions = []action.Action{{Label: "See all", Href: "/changes"}}
	out := render(t, p)
	inOrder(t, out, `id="recent"`, "Recent changes", "Last 24 hours", "See all", "RECENT CONTENT", `id="newest"`, "NEWEST CONTENT")
	if !strings.Contains(out, `href="/changes"`) {
		t.Errorf("section action missing: %s", out)
	}
	if n := strings.Count(out, "<section"); n != 2 {
		t.Errorf("want 2 layout.Section elements, got %d", n)
	}
}

func TestPage_OrderTilesBeforeSections(t *testing.T) {
	out := render(t, dashboard())
	inOrder(t, out, "<header", "wf-dashgrid", "Items", "Packs", "Recent changes", "Newest items")
}

func TestPage_GridFlushAndGap(t *testing.T) {
	out := render(t, dashboard())
	if !strings.Contains(out, "padding:0px") {
		t.Errorf("grid should be Flush (the page gutter already insets it): %s", out)
	}
	if !strings.Contains(out, "gap:16px") {
		t.Errorf("default gap 16: %s", out)
	}
	if !strings.Contains(out, "repeat(12,1fr)") {
		t.Errorf("12-column grid: %s", out)
	}
	p := dashboard()
	p.Gap = 24
	if out := render(t, p); !strings.Contains(out, "gap:24px") {
		t.Errorf("Gap not passed through: %s", out)
	}
}

func TestPage_NoGridWhenEmpty(t *testing.T) {
	out := render(t, Props{Header: pageheader.Props{Title: "Empty"}})
	if strings.Contains(out, "wf-dashgrid") {
		t.Errorf("no grid without tiles or sections: %s", out)
	}
	if !strings.Contains(out, "<h1") {
		t.Errorf("header still renders: %s", out)
	}
}

func TestPage_ClassAndAttributes(t *testing.T) {
	p := dashboard()
	p.Class = "wf-dash"
	p.Attributes = templ.Attributes{"data-x": "1"}
	out := render(t, p)
	if !strings.Contains(out, "wf-dash") || !strings.Contains(out, `data-x="1"`) {
		t.Errorf("Class/Attributes not on root: %s", out)
	}
	if !strings.Contains(out, `id="dash"`) {
		t.Errorf("ID not on root: %s", out)
	}
}

// TestPhoneCollapse pins how the grid stacks on a phone. Spans are inline
// styles, so the collapse is a pair of !important rules in
// static/css/input.css keyed on wf-dashgrid / wf-dashslot. Both halves must
// hold: the markup carries the classes, and the stylesheet still has the
// rules.
func TestPhoneCollapse(t *testing.T) {
	out := render(t, dashboard())
	if n := strings.Count(out, "wf-dashgrid"); n != 1 {
		t.Errorf("want one wf-dashgrid root, got %d", n)
	}
	if n := strings.Count(out, "wf-dashslot"); n != 6 {
		t.Errorf("want wf-dashslot on all 6 slots (4 tiles + 2 sections), got %d", n)
	}
	if n := strings.Count(out, "min-w-0 wf-dashslot") + strings.Count(out, "wf-dashslot min-w-0"); n != 6 {
		t.Errorf("every slot must be min-w-0 so tile content cannot widen its cell, got %d", n)
	}

	css, err := os.ReadFile(filepath.Join("..", "..", "static", "css", "input.css"))
	if err != nil {
		t.Fatalf("read input.css: %v", err)
	}
	src := string(css)
	// Tailwind's md is 48rem; max-width 47.99rem keeps the two from
	// both applying.
	start := strings.Index(src, "@media (max-width: 47.99rem)")
	if start < 0 {
		t.Fatalf("input.css has no @media (max-width: 47.99rem) block")
	}
	end := strings.Index(src[start:], "\n}")
	if end < 0 {
		t.Fatalf("unterminated media block")
	}
	block := regexp.MustCompile(`\s+`).ReplaceAllString(src[start:start+end], " ")
	for _, rule := range []string{
		".wf-dashgrid { grid-template-columns: 1fr !important; }",
		".wf-dashslot { grid-column: span 1 !important; grid-row: span 1 !important; }",
	} {
		if !strings.Contains(block, rule) {
			t.Errorf("phone collapse rule missing from input.css media block:\nwant %s\nin %s", rule, block)
		}
	}
}

func testApp() *appshell.App {
	return &appshell.App{Name: "Demo", Nav: []appshell.NavItem{{Label: "Dashboard", Href: "/dashboard"}}}
}

func serve(t *testing.T, app *appshell.App, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Respond(w, r, "Overview", dashboard()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	if app != nil {
		h = app.Middleware(h)
	}
	req := httptest.NewRequest("GET", "/dashboard", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRespond_FullDocument(t *testing.T) {
	rec := serve(t, testApp(), nil)
	out := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, out)
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(out)), "<!doctype") {
		t.Errorf("want full document: %s", out[:min(len(out), 200)])
	}
	for _, want := range []string{"Overview", "wf-dashgrid", "RECENT CONTENT", "Dashboard"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestRespond_Fragment(t *testing.T) {
	rec := serve(t, testApp(), map[string]string{"HX-Request": "true"})
	out := rec.Body.String()
	if strings.Contains(strings.ToLower(out), "<!doctype") {
		t.Errorf("plain htmx request should get the bare body: %s", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "<h1") || !strings.Contains(out, "wf-dashgrid") {
		t.Errorf("fragment should carry the header and grid: %s", out)
	}
}

func TestRespond_NoApp(t *testing.T) {
	rec := serve(t, nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 without Middleware, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Middleware") {
		t.Errorf("error should name Middleware: %s", rec.Body.String())
	}
}
