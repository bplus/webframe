package detailpage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/tabs"
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

func renderWithChildren(t *testing.T, p Props, children templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), children)
	if err := Page(p).Render(ctx, &buf); err != nil {
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

func itemPage() Props {
	return Props{
		ID:     "item",
		Header: pageheader.Props{Title: "Blade of the Fallen Star"},
		Summary: []descriptionlist.Item{
			{Label: "Type", Value: "Weapon"},
			{Label: "Status", Value: "Published"},
		},
		Tabs: []Tab{
			{ID: "notes", Label: "Notes", Content: text("<p>Some notes</p>")},
			{ID: "history", Label: "History", HxGet: "/items/1/history"},
		},
	}
}

func TestPage_SummaryAndTabs(t *testing.T) {
	out := render(t, itemPage())
	if !strings.Contains(out, "<h1") || !strings.Contains(out, "Blade of the Fallen Star") {
		t.Errorf("missing h1 title: %s", out)
	}
	if n := strings.Count(out, "<dt"); n != 2 {
		t.Errorf("want 2 <dt, got %d: %s", n, out)
	}
	if n := strings.Count(out, "data-tui-tabs-trigger"); n != 2 {
		t.Errorf("want 2 tab triggers, got %d: %s", n, out)
	}
	// The first trigger is the active one when ActiveTab is "".
	first := strings.Index(out, "data-tui-tabs-trigger")
	second := strings.Index(out[first+1:], "data-tui-tabs-trigger") + first + 1
	if !strings.Contains(out[first:second], `data-tui-tabs-state="active"`) {
		t.Errorf("first trigger not active: %s", out[first:second])
	}
	if !strings.Contains(out[second:], `data-tui-tabs-state="inactive"`) {
		t.Errorf("second trigger not inactive: %s", out[second:])
	}
	if !strings.Contains(out, "<p>Some notes</p>") {
		t.Errorf("inline tab content missing: %s", out)
	}
	// Order: header, summary, tabs.
	inOrder(t, out, "<h1", "<dl", "data-tui-tabs-trigger")
	if !strings.Contains(out, `id="item"`) || !strings.Contains(out, `data-tui-tabs-id="item-tabs"`) {
		t.Errorf("root id / tabs group id missing: %s", out)
	}
}

func TestPage_ActiveTabSelectsByID(t *testing.T) {
	p := itemPage()
	p.ActiveTab = "history"
	out := render(t, p)
	i := strings.Index(out, `data-tui-tabs-value="history"`)
	if i < 0 {
		t.Fatalf("history trigger missing: %s", out)
	}
	// The attribute order on Trigger is value then state.
	if !strings.Contains(out[i:i+120], `data-tui-tabs-state="active"`) {
		t.Errorf("history trigger not active: %s", out[i:i+120])
	}
}

func TestPage_LazyTab(t *testing.T) {
	out := render(t, itemPage())
	for _, want := range []string{
		`hx-get="/items/1/history"`,
		`hx-trigger="click once"`,
		`hx-target="#history-panel"`,
		`id="history-panel"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in: %s", want, out)
		}
	}
	// The inline tab carries no htmx attributes.
	notes := out[strings.Index(out, `data-tui-tabs-value="notes"`):]
	notes = notes[:strings.Index(notes, "</button>")]
	if strings.Contains(notes, "hx-") {
		t.Errorf("inline trigger has hx- attributes: %s", notes)
	}
	// The lazy panel is empty: nothing between its opening tag and </div>.
	i := strings.Index(out, `id="history-panel"`)
	panel := out[i:]
	open := strings.Index(panel, ">")
	if body := panel[open+1 : strings.Index(panel, "</div>")]; strings.TrimSpace(body) != "" {
		t.Errorf("lazy panel not empty: %q", body)
	}
}

func TestPage_ActiveLazyTabLoadsOnPageLoad(t *testing.T) {
	p := itemPage()
	p.ActiveTab = "history"
	out := render(t, p)
	if !strings.Contains(out, `hx-trigger="load"`) {
		t.Errorf("active lazy tab should load on page load: %s", out)
	}
	if strings.Contains(out, `hx-trigger="click once"`) {
		t.Errorf("active lazy tab should not also fire on click: %s", out)
	}
}

func TestPage_TabsSizingPassesThrough(t *testing.T) {
	out := render(t, itemPage())
	if !strings.Contains(out, `data-wf-tabs-sizing="tallest"`) {
		t.Errorf("default sizing should be tallest: %s", out)
	}
	p := itemPage()
	p.TabsSizing = tabs.SizingFixed
	out = render(t, p)
	if !strings.Contains(out, `data-wf-tabs-sizing="fixed"`) {
		t.Errorf("sizing not passed through: %s", out)
	}
}

func TestPage_NoTabsNoSummary(t *testing.T) {
	out := render(t, Props{Header: pageheader.Props{Title: "Bare"}})
	if strings.Contains(out, "<dl") || strings.Contains(out, "data-tui-tabs") {
		t.Errorf("empty summary/tabs should render nothing: %s", out)
	}
	if !strings.Contains(out, "Bare") {
		t.Errorf("title missing: %s", out)
	}
}

func TestPage_SummaryColumns(t *testing.T) {
	p := itemPage()
	p.SummaryColumns = 2
	out := render(t, p)
	if !strings.Contains(out, "md:grid-cols-2") {
		t.Errorf("two-column summary missing: %s", out)
	}
}

func TestPage_Aside(t *testing.T) {
	p := itemPage()
	p.Aside = text("<p>ASIDE TEXT</p>")
	p.AsideWidth = layout.AsideS
	p.AsideSticky = true
	out := render(t, p)
	inOrder(t, out, "<h1", "<dl", "data-tui-tabs-trigger", "<aside", "ASIDE TEXT")
	for _, want := range []string{"md:flex-row", "md:w-64", "md:sticky"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s: %s", want, out)
		}
	}
	// The header is outside the two-column wrapper.
	if strings.Index(out, "<h1") > strings.Index(out, "md:flex-row") {
		t.Errorf("header should precede the two-column wrapper: %s", out)
	}
}

func TestPage_NoAsideNoTwoColumn(t *testing.T) {
	out := render(t, itemPage())
	if strings.Contains(out, "<aside") || strings.Contains(out, "md:flex-row") {
		t.Errorf("no Aside should mean no TwoColumn: %s", out)
	}
}

func TestPage_ChildrenRendered(t *testing.T) {
	out := renderWithChildren(t, itemPage(), text("<p>FREE CONTENT</p>"))
	// Children come after the tabs.
	inOrder(t, out, "data-tui-tabs-trigger", "</button>", "FREE CONTENT")
	last := strings.LastIndex(out, "data-tui-tabs-content")
	if strings.Index(out, "FREE CONTENT") < last {
		t.Errorf("children should follow the tab panels: %s", out)
	}

	// Without tabs, children come right after the summary.
	p := itemPage()
	p.Tabs = nil
	out = renderWithChildren(t, p, text("<p>FREE CONTENT</p>"))
	inOrder(t, out, "</dl>", "FREE CONTENT")

	// With an Aside, children sit inside the main column, before the aside.
	p = itemPage()
	p.Aside = text("<p>ASIDE TEXT</p>")
	out = renderWithChildren(t, p, text("<p>FREE CONTENT</p>"))
	inOrder(t, out, "FREE CONTENT", "<aside", "ASIDE TEXT")
}

func TestPage_ClassAndAttributes(t *testing.T) {
	p := itemPage()
	p.Class = "wf-detail"
	p.Attributes = templ.Attributes{"data-x": "1"}
	out := render(t, p)
	if !strings.Contains(out, "wf-detail") || !strings.Contains(out, `data-x="1"`) {
		t.Errorf("Class/Attributes not on root: %s", out)
	}
}

func testApp() *appshell.App {
	return &appshell.App{Name: "Demo", Nav: []appshell.NavItem{{Label: "Items", Href: "/items"}}}
}

func serve(t *testing.T, app *appshell.App, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Respond(w, r, "Blade", itemPage()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	if app != nil {
		h = app.Middleware(h)
	}
	req := httptest.NewRequest("GET", "/items/1", nil)
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
	for _, want := range []string{"Blade of the Fallen Star", `hx-get="/items/1/history"`, "Items"} {
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
	if !strings.Contains(out, "<h1") {
		t.Errorf("fragment should still carry the page header: %s", out)
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
