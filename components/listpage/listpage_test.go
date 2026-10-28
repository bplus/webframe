package listpage

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/datatable"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/listquery"
)

func renderPage(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Page(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render page: %v", err)
	}
	return buf.String()
}

func renderResults(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Results(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render results: %v", err)
	}
	return buf.String()
}

func wantAll(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func wantNone(t *testing.T, out string, bads ...string) {
	t.Helper()
	for _, b := range bads {
		if strings.Contains(out, b) {
			t.Errorf("unexpected %q in:\n%s", b, out)
		}
	}
}

// wantOrder asserts the substrings appear in out in the given order.
func wantOrder(t *testing.T, out string, parts ...string) {
	t.Helper()
	pos := 0
	for _, part := range parts {
		i := strings.Index(out[pos:], part)
		if i < 0 {
			t.Fatalf("missing %q after offset %d in:\n%s", part, pos, out)
		}
		pos += i + len(part)
	}
}

var itemOptions = listquery.Options{
	DefaultSort:  "updated",
	DefaultDir:   listquery.Desc,
	AllowedSorts: []string{"name", "type", "status", "updated"},
	FilterKeys:   []string{"type", "status"},
}

func itemSpec(t *testing.T, query string) listquery.Spec {
	t.Helper()
	v, err := url.ParseQuery(query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return listquery.ParseValues(v, itemOptions)
}

func itemFilters() []filterbar.FilterConfig {
	return []filterbar.FilterConfig{
		{ID: "type", Label: "TYPEFILTER", Type: filterbar.TypeSelect, Placeholder: "Any type",
			Options: []filterbar.FilterOption{{Label: "Weapon", Value: "weapon"}, {Label: "Armor", Value: "armor"}}},
		{ID: "status", Label: "STATUSFILTER", Type: filterbar.TypeToggle,
			Options: []filterbar.FilterOption{{Label: "Published", Value: "published"}, {Label: "Draft", Value: "draft"}}},
	}
}

func itemRows() []datatable.Row {
	return []datatable.Row{
		{ID: "1", Href: "/items/1", Cells: []datatable.Cell{{Text: "ROWALPHA"}, {Text: "published"}}},
		{ID: "2", Cells: []datatable.Cell{{Text: "ROWBETA"}, {Text: "draft"}}},
		{ID: "3", Cells: []datatable.Cell{{Text: "ROWGAMMA"}, {Text: "draft"}}},
	}
}

func itemProps(t *testing.T, query string) Props {
	t.Helper()
	return Props{
		Header:  pageheader.Props{Title: "PAGETITLE"},
		Intro:   "INTROLINE",
		Filters: itemFilters(),
		Search:  "Search items",
		Toolbar: []action.Action{{Label: "BUILDMOD", Href: "#"}},
		Create:  &action.Action{Label: "CREATEITEM", Href: "/items/new"},
		Columns: []datatable.Column{
			{Key: "name", Label: "NAMECOL", Sortable: true},
			{Key: "status", Label: "STATUSCOL", Shrink: true},
		},
		Rows:  itemRows(),
		Spec:  itemSpec(t, query),
		Total: 3,
		Noun:  "items",
		Base:  "/items",
	}
}

func TestPage_Composition(t *testing.T) {
	out := renderPage(t, itemProps(t, ""))
	wantOrder(t, out,
		"PAGETITLE",
		"INTROLINE",
		"TYPEFILTER",
		"BUILDMOD",
		"CREATEITEM",
		`id="list-results"`,
		"NAMECOL",
		"ROWALPHA",
		"1–3 of 3 items",
	)
	wantAll(t, out, `id="list"`, `id="list-table"`, `role="toolbar"`, `aria-label="Pagination"`)
}

func TestPage_DefaultsAndRoot(t *testing.T) {
	var p Props
	if p.ResultsID() != "list-results" {
		t.Errorf("zero Props ResultsID = %q", p.ResultsID())
	}
	p.ID = "items"
	if p.ResultsID() != "items-results" {
		t.Errorf("ResultsID = %q", p.ResultsID())
	}
	out := renderPage(t, Props{ID: "items", Class: "mt-2", Attributes: templ.Attributes{"data-x": "1"}, Header: pageheader.Props{Title: "T"}})
	wantAll(t, out, `id="items"`, "mt-2", `data-x="1"`, `id="items-results"`, `id="items-table"`)
	// No filters and no search: no filter bar. No toolbar content: no toolbar.
	wantNone(t, out, "data-tui-filterbar", `role="toolbar"`)
}

func TestResults_NoHeader(t *testing.T) {
	out := renderResults(t, itemProps(t, ""))
	wantAll(t, out, `id="list-results"`, "NAMECOL", "ROWALPHA", "1–3 of 3 items")
	wantNone(t, out, "PAGETITLE", "<h1", "INTROLINE", "data-tui-filterbar", `role="toolbar"`, "CREATEITEM")
	if !strings.HasPrefix(out, `<div id="list-results"`) {
		t.Errorf("results should start with the results div, got:\n%s", out)
	}
}

func TestPage_FiltersSelectedFromSpec(t *testing.T) {
	out := renderPage(t, itemProps(t, "status=draft&type=armor"))
	// Toggle pill "Draft" is selected, "Published" is not.
	wantAll(t, out, `value="draft" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="true"`)
	wantAll(t, out, `value="published" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="false"`)
	// Select option "armor" is selected.
	wantAll(t, out, `<option value="armor" selected>Armor</option>`)
	wantNone(t, out, `<option value="weapon" selected>`)
}

func TestPage_SearchFromSpec(t *testing.T) {
	out := renderPage(t, itemProps(t, "q=blade"))
	wantAll(t, out, `name="q"`, `value="blade"`, `placeholder="Search items"`)
}

func TestPage_HiddenSortRidesAlong(t *testing.T) {
	out := renderPage(t, itemProps(t, "sort=name&dir=asc&page=2&per=10&q=x&status=draft"))
	wantAll(t, out,
		`<input type="hidden" name="sort" value="name" data-tui-filterbar-control>`,
		`<input type="hidden" name="dir" value="asc" data-tui-filterbar-control>`,
		`<input type="hidden" name="per" value="10" data-tui-filterbar-control>`,
	)
	// page, q and filters never ride along as hidden inputs: a filter change
	// lands on page 1 and the controls themselves carry q and the filters.
	wantNone(t, out, `type="hidden" name="page"`, `type="hidden" name="q"`, `type="hidden" name="status"`)
}

func TestPage_FilterbarTargetsResults(t *testing.T) {
	out := renderPage(t, itemProps(t, ""))
	wantAll(t, out,
		`data-tui-filterbar-hx-get="/items"`,
		`data-tui-filterbar-hx-target="#list-results"`,
		`data-tui-filterbar-reset`,
	)
	// Sort and page links target the results region too.
	wantAll(t, out, `hx-target="#list-results"`, `hx-push-url="true"`)
}

func TestPage_PrefixedSpecNamesControls(t *testing.T) {
	p := itemProps(t, "items.sort=name&items.status=draft&items.q=hi")
	o := itemOptions
	o.Prefix = "items"
	v, _ := url.ParseQuery("items.sort=name&items.status=draft&items.q=hi")
	p.Spec = listquery.ParseValues(v, o)
	out := renderPage(t, p)
	wantAll(t, out,
		`name="items.sort" value="name"`,
		`name="items.status"`,
		`name="items.type"`,
		`name="items.q"`,
		`value="hi"`,
		`value="draft" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="true"`,
	)
	wantNone(t, out, `name="status"`, `name="q"`, `name="sort"`)
}

func TestPage_BulkActionsForm(t *testing.T) {
	p := itemProps(t, "")
	p.Selection = true
	p.BulkActions = []action.Action{
		{Label: "PUBLISHSEL", HxPost: "/items/publish"},
		{Label: "EXPORTSEL", Href: "/items/export"},
	}
	out := renderPage(t, p)
	wantAll(t, out,
		`<form id="list-bulk"`,
		`hx-post="/items/publish"`,
		`hx-target="#list-results"`,
		`href="/items/export"`,
		// Row checkboxes submit with the bulk form, repeated under "ids".
		`name="ids"`,
		`form="list-bulk"`,
		`value="1"`,
	)
	// The form sits between the left and right toolbar groups.
	wantOrder(t, out, "BUILDMOD", `<form id="list-bulk"`, "PUBLISHSEL", "EXPORTSEL", "CREATEITEM")
	// An explicit HxTarget on a bulk action is kept.
	p.BulkActions = []action.Action{{Label: "X", HxPost: "/x", HxTarget: "#elsewhere"}}
	out = renderPage(t, p)
	wantAll(t, out, `hx-target="#elsewhere"`)
}

func TestPage_BulkActionsNeedSelection(t *testing.T) {
	p := itemProps(t, "")
	p.BulkActions = []action.Action{{Label: "PUBLISHSEL", HxPost: "/items/publish"}}
	out := renderPage(t, p)
	wantNone(t, out, "list-bulk", "PUBLISHSEL")
}

func TestPage_CreateDefaultsToPrimary(t *testing.T) {
	out := renderPage(t, itemProps(t, ""))
	// button.VariantDefault is the filled primary style.
	i := strings.Index(out, "CREATEITEM")
	if i < 0 {
		t.Fatal("no create button")
	}
	start := strings.LastIndex(out[:i], "<a")
	wantAll(t, out[start:i], "bg-primary", `href="/items/new"`)
}

func TestPage_Aside(t *testing.T) {
	p := itemProps(t, "")
	p.Aside = templ.Raw("ASIDETEXT")
	out := renderPage(t, p)
	// The header stays above the two columns; everything else is inside.
	wantOrder(t, out, "PAGETITLE", "md:flex-row", "INTROLINE", "data-tui-filterbar", `id="list-results"`, "<aside", "ASIDETEXT")
	without := renderPage(t, itemProps(t, ""))
	wantNone(t, without, "<aside", "md:flex-row")
}

func TestResults_EmptyDistinguishesFilters(t *testing.T) {
	p := itemProps(t, "")
	p.Rows = nil
	p.Total = 0
	out := renderResults(t, p)
	wantAll(t, out, "Nothing here yet", "0 items", "NAMECOL")

	p = itemProps(t, "status=draft")
	p.Rows = nil
	p.Total = 0
	out = renderResults(t, p)
	wantAll(t, out, "No matches", "filters", "0 items", "NAMECOL", "Clear filters")
	wantNone(t, out, "Nothing here yet")

	// A caller-provided Empty wins in both cases.
	p.Empty = emptystate.Props{Title: "CUSTOMEMPTY"}
	out = renderResults(t, p)
	wantAll(t, out, "CUSTOMEMPTY")
	wantNone(t, out, "No matches")
}

func TestResults_ClearFiltersLink(t *testing.T) {
	p := itemProps(t, "sort=name&dir=asc&page=2&per=10&q=blade&status=draft&type=armor")
	p.Rows = nil
	p.Total = 0
	out := renderResults(t, p)
	// sort/dir/per kept, page reset, q and filters gone; same URL on hx-get.
	clear := "/items?dir=asc&amp;per=10&amp;sort=name"
	wantAll(t, out, "Clear filters", `href="`+clear+`"`, `hx-get="`+clear+`"`, `hx-target="#list-results"`, `hx-push-url="true"`)
	i := strings.Index(out, "Clear filters")
	link := out[strings.LastIndex(out[:i], "<a"):i]
	wantNone(t, link, "q=", "status=", "type=", "page=")
	// Without filters there is no Clear filters action at all.
	p = itemProps(t, "")
	p.Rows = nil
	out = renderResults(t, p)
	wantNone(t, out, "Clear filters")
}

// --- Respond ---------------------------------------------------------------

func testApp() *appshell.App {
	return &appshell.App{
		Name: "Demo",
		Nav:  []appshell.NavItem{{Label: "Items", Href: "/items"}},
	}
}

func serve(t *testing.T, headers map[string]string) (*httptest.ResponseRecorder, error) {
	t.Helper()
	a := testApp()
	var respondErr error
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := itemProps(t, r.URL.RawQuery)
		p.Base = ""
		respondErr = Respond(w, r, "Items", p)
	}))
	req := httptest.NewRequest("GET", "/items?sort=name", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, respondErr
}

func TestRespond_TargetedResults(t *testing.T) {
	rec, err := serve(t, map[string]string{"HX-Request": "true", "HX-Target": "list-results"})
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, `<div id="list-results"`) {
		t.Errorf("body should start with the results div, got:\n%s", body)
	}
	// ?sort=name without a dir sorts ascending (DefaultDir belongs to the
	// default sort key); the header link flips it.
	wantAll(t, body, "ROWALPHA", "1–3 of 3 items", `aria-sort="ascending"`, `href="/items?dir=desc&amp;sort=name"`)
	wantNone(t, body, "<h1", "PAGETITLE", "<html", "wf-topbar", "data-tui-filterbar")
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Target") {
		t.Errorf("Vary = %q", v)
	}
}

func TestRespond_OtherTargetGetsBody(t *testing.T) {
	rec, err := serve(t, map[string]string{"HX-Request": "true", "HX-Target": "main"})
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	body := rec.Body.String()
	wantAll(t, body, "<h1", "PAGETITLE", `id="list-results"`, "ROWALPHA")
	wantNone(t, body, "<html", "wf-topbar", "<title>")
}

func TestRespond_FullDocument(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"plain":           nil,
		"boosted":         {"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "list-results"},
		"history-restore": {"HX-Request": "true", "HX-History-Restore-Request": "true", "HX-Target": "list-results"},
	} {
		rec, err := serve(t, headers)
		if err != nil {
			t.Fatalf("%s: respond: %v", name, err)
		}
		body := rec.Body.String()
		if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
			t.Errorf("%s: missing doctype in:\n%s", name, body)
		}
		wantAll(t, body, "<title>Items — Demo</title>", "wf-topbar", "PAGETITLE", `id="list-results"`, "ROWALPHA",
			// Base "" resolved to the request path for the filter bar.
			`data-tui-filterbar-hx-get="/items"`)
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("%s: Content-Type = %q", name, ct)
		}
	}
}

func TestRespond_NoApp(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/items", nil)
	err := Respond(rec, req, "Items", itemProps(t, ""))
	if err == nil {
		t.Fatal("Respond without Middleware should fail")
	}
	if !strings.Contains(err.Error(), "Middleware") {
		t.Errorf("error %q does not name Middleware", err)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("nothing should be written on error, got %q", rec.Body.String())
	}
}
