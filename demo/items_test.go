package demo

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bplus/webframe/assets"
)

// itemsHandler builds the server as cmd/demo does (assets, the Items
// routes, the server-wide App.Middleware) on a fresh seeded store.
func itemsHandler(t *testing.T) http.Handler {
	t.Helper()
	Items = NewItemStore(time.Now())
	mux := http.NewServeMux()
	assets.Mount(mux)
	RegisterItemRoutes(mux)
	return App.Middleware(mux)
}

// do issues one request through h and returns the recorder.
func do(h http.Handler, method, target string, form url.Values, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// hx are the headers of a plain htmx request targeting id.
func hx(target string) map[string]string {
	h := map[string]string{"HX-Request": "true"}
	if target != "" {
		h["HX-Target"] = target
	}
	return h
}

func wantAll(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("missing %q in:\n%s", w, body)
		}
	}
}

func wantNone(t *testing.T, body string, nots ...string) {
	t.Helper()
	for _, w := range nots {
		if strings.Contains(body, w) {
			t.Errorf("unexpected %q in:\n%s", w, body)
		}
	}
}

// inOrder fails unless each needle appears after the previous one.
func inOrder(t *testing.T, body string, needles ...string) {
	t.Helper()
	pos := 0
	for _, n := range needles {
		i := strings.Index(body[pos:], n)
		if i < 0 {
			t.Errorf("%q not found after position %d", n, pos)
			return
		}
		pos += i + len(n)
	}
}

var thAsc = regexp.MustCompile(`(?s)<th[^>]*aria-sort="ascending"[^>]*>(.*?)</th>`)

func TestItems_ListSortsAndFilters(t *testing.T) {
	h := itemsHandler(t)
	rec := do(h, "GET", "/items?sort=name&dir=asc&status=draft", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /items = %d", rec.Code)
	}
	body := rec.Body.String()
	// Only the three drafts, in name order.
	inOrder(t, body, "Amulet of Still Water", "Cloak of Whispers", "Helm of the Last Watch")
	wantNone(t, body, "Blade of the Fallen Star", "Ashen Dagger")
	// Name is the sorted column.
	m := thAsc.FindStringSubmatch(body)
	if m == nil || !strings.Contains(m[1], "Name") {
		t.Errorf("no <th aria-sort=\"ascending\"> holding Name; match=%q", m)
	}
	if n := strings.Count(body, `aria-sort="ascending"`); n != 1 {
		t.Errorf("aria-sort=ascending appears %d times, want 1", n)
	}
	// The whole page: sidebar shell, the live intro line, the filter
	// controls, the toolbar and the summary.
	wantAll(t, body,
		"wf-shell--sidebar", "Items — Webframe</title>", `aria-current="page"`,
		"14 items · 11 published · 3 drafts",
		"All types", "Search items", "Published", "Draft",
		"Build mod", "+ Create item", "Publish selected",
		"1–3 of 3 items", `id="items-results"`,
		`hx-delete="/items/2"`, `hx-confirm="Delete this item?"`, `href="/items/2/edit"`,
	)
	// The default list is newest first with every row.
	body = do(h, "GET", "/items", nil, nil).Body.String()
	inOrder(t, body, "Blade of the Fallen Star", "Cloak of Whispers", "Mantle of Ember")
	wantAll(t, body, "1–14 of 14 items", `aria-sort="descending"`)
	// A search narrows by name, case-insensitively.
	body = do(h, "GET", "/items?q=OF+THE", nil, nil).Body.String()
	wantAll(t, body, "Blade of the Fallen Star", "Pendant of the Weave")
	wantNone(t, body, "Ashen Dagger")
	// Hostile input parses to defaults.
	if rec := do(h, "GET", "/items?sort=evil&dir=sideways&page=-3&per=100000", nil, nil); rec.Code != 200 {
		t.Errorf("hostile query = %d", rec.Code)
	}
	// No matches: the header stays, the filtered empty state offers Clear filters.
	body = do(h, "GET", "/items?q=zzzz", nil, nil).Body.String()
	wantAll(t, body, "No matches", "Clear filters", "0 items", "<th")
}

func TestItems_ResultsSwap(t *testing.T) {
	h := itemsHandler(t)
	rec := do(h, "GET", "/items?sort=name", nil, hx("items-results"))
	body := rec.Body.String()
	if rec.Code != 200 {
		t.Fatalf("swap = %d", rec.Code)
	}
	if !strings.HasPrefix(strings.TrimSpace(body), `<div id="items-results"`) {
		t.Errorf("results swap does not start with the results div:\n%s", body[:min(200, len(body))])
	}
	wantAll(t, body, `id="items-results"`, "Blade of the Fallen Star", "1–14 of 14 items",
		// After the results, out of band: the bar's hidden inputs and its
		// Sort select (the phone sort control, in the hidden sort input's
		// place since the columns are sortable) with the served sort, so the
		// next filter change keeps it and the select shows it, and the intro.
		`<input type="hidden" id="items-filterbar-hidden-dir" name="dir" value="asc" data-tui-filterbar-control hx-swap-oob="true">`,
		`<select id="items-filterbar-hidden-sort" name="sort" data-tui-filterbar-control data-tui-filterbar-state hx-swap-oob="true"`,
		`<option value="name" selected data-tui-filterbar-set-dir="asc">Name ↑</option>`,
		`<option value="" data-tui-filterbar-set-dir="">Updated ↓</option>`,
		`<p id="items-intro" class="text-sm text-muted-foreground" hx-swap-oob="true">14 items · 11 published · 3 drafts</p>`)
	wantNone(t, body, "<h1", "<html", "wf-topbar", "wf-sidebar", "Search items", "data-tui-filterbar-hx-get")
	// Any other htmx target gets the page body without the shell.
	body = do(h, "GET", "/items", nil, hx("main")).Body.String()
	wantAll(t, body, "<h1", "Search items", `id="items-filterbar"`, `data-tui-filterbar-drop="page"`, `<p id="items-intro"`)
	wantNone(t, body, "<html", "wf-sidebar", "hx-swap-oob")
	// A boosted request gets the whole document.
	body = do(h, "GET", "/items", nil, map[string]string{"HX-Request": "true", "HX-Boosted": "true"}).Body.String()
	wantAll(t, body, "<!doctype", "wf-shell--sidebar")
}

func TestItems_CreateErrors(t *testing.T) {
	h := itemsHandler(t)
	form := url.Values{"name": {"ab"}, "status": {"draft"}, "notes": {"keep me"}}
	rec := do(h, "POST", "/items", form, hx("item-form"))
	if rec.Code != 422 {
		t.Fatalf("POST /items with errors = %d, want 422 (Document's htmx config swaps error responses)", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(strings.TrimSpace(body), "<form") {
		t.Errorf("form swap does not start with <form:\n%s", body[:min(200, len(body))])
	}
	wantAll(t, body,
		"Please fix the 2 errors below.",
		"Name must be at least 3 characters", "Choose a type",
		`value="ab"`, "keep me", `aria-invalid="true"`,
		`hx-post="/items"`, `hx-target="#item-form"`, `hx-swap="outerHTML"`,
		`hx-post="/items/validate"`,
	)
	wantNone(t, body, "<h1", "<html")
	if n := strings.Count(body, `aria-invalid="true"`); n != 2 {
		t.Errorf("aria-invalid count = %d, want 2", n)
	}
	if rec.Header().Get("HX-Redirect") != "" {
		t.Error("error re-render must not redirect")
	}
	// A plain (non-htmx) POST with errors renders the whole document.
	body = do(h, "POST", "/items", url.Values{"name": {"Blade of the Fallen Star"}, "type": {"weapon"}, "status": {"draft"}}, nil).Body.String()
	wantAll(t, body, "<!doctype", "Please fix the 1 error below.", "already exists", `value="Blade of the Fallen Star"`)
	if total, _, _ := Items.Counts(); total != 14 {
		t.Errorf("store changed on an invalid submit: %d items", total)
	}
}

func TestItems_CreateSuccessRedirects(t *testing.T) {
	h := itemsHandler(t)
	form := url.Values{"name": {"  Sunforged Maul "}, "type": {"weapon"}, "status": {"draft"}, "remixable": {"on"}, "notes": {"Hits hard."}}
	rec := do(h, "POST", "/items", form, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/items/15" {
		t.Fatalf("POST /items = %d Location %q, want 303 /items/15", rec.Code, rec.Header().Get("Location"))
	}
	if strings.Contains(rec.Body.String(), "success") {
		t.Error("success response must carry no success message")
	}
	it, ok := Items.Get("15")
	if !ok || it.Name != "Sunforged Maul" || it.Type != "weapon" || !it.Remixable || it.Notes != "Hits hard." {
		t.Errorf("stored item = %+v", it)
	}
	body := do(h, "GET", "/items/15", nil, nil).Body.String()
	wantAll(t, body, "Sunforged Maul", "Hits hard.")

	// An htmx submit gets HX-Redirect with an empty 200.
	form.Set("name", "Second Maul")
	rec = do(h, "POST", "/items", form, hx("item-form"))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/items/16" {
		t.Errorf("htmx POST /items = %d HX-Redirect %q, want 200 /items/16", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if rec.Body.Len() != 0 {
		t.Errorf("htmx redirect body = %q, want empty", rec.Body.String())
	}

	// Editing: the form carries the values; a save redirects to the item.
	body = do(h, "GET", "/items/15/edit", nil, nil).Body.String()
	wantAll(t, body, `value="Sunforged Maul"`, "Save", `hx-post="/items/validate?id=15"`, `href="/items/15"`, "Hits hard.")
	rec = do(h, "POST", "/items/15", url.Values{"name": {"Sunforged Maul"}, "type": {"armor"}, "status": {"published"}}, hx("item-form"))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/items/15" {
		t.Errorf("htmx POST /items/15 = %d HX-Redirect %q", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if it, _ := Items.Get("15"); it.Type != "armor" || it.Status != "published" || it.Remixable {
		t.Errorf("updated item = %+v", it)
	}
}

func TestItems_Validate(t *testing.T) {
	h := itemsHandler(t)
	cases := []struct {
		name    string
		target  string
		form    url.Values
		headers map[string]string
		want    []string
		notWant []string
	}{
		{"short name", "/items/validate", url.Values{"name": {"ab"}}, hx(""),
			[]string{`id="field-name-feedback"`, "Name must be at least 3 characters", `role="alert"`}, nil},
		{"taken name", "/items/validate", url.Values{"name": {"blade of the fallen star"}}, hx(""),
			[]string{`id="field-name-feedback"`, "already exists"}, nil},
		{"own name while editing", "/items/validate?id=1", url.Values{"name": {"Blade of the Fallen Star"}}, hx(""),
			[]string{`id="field-name-feedback"`}, []string{"already exists", "<p"}},
		{"valid name is an empty slot", "/items/validate", url.Values{"name": {"Totally New"}}, hx(""),
			[]string{`id="field-name-feedback"`}, []string{"<p", "alert"}},
		{"missing type", "/items/validate", url.Values{"type": {""}}, hx(""),
			[]string{`id="field-type-feedback"`, "Choose a type"}, nil},
		{"whole form posted, trigger named", "/items/validate", url.Values{"name": {"Fine Name"}, "type": {""}, "status": {"draft"}},
			map[string]string{"HX-Request": "true", "HX-Trigger-Name": "type"},
			[]string{`id="field-type-feedback"`, "Choose a type"}, []string{"field-name-feedback"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(h, "POST", c.target, c.form, c.headers)
			if rec.Code != 200 {
				t.Fatalf("code = %d", rec.Code)
			}
			body := rec.Body.String()
			wantAll(t, body, c.want...)
			wantNone(t, body, append(c.notWant, "<html", "<form")...)
		})
	}
	if rec := do(h, "POST", "/items/validate", url.Values{"bogus": {"x"}}, hx("")); rec.Code != 400 {
		t.Errorf("unknown field = %d, want 400", rec.Code)
	}
}

func TestItems_Delete(t *testing.T) {
	h := itemsHandler(t)
	headers := hx("items-results")
	headers["HX-Current-URL"] = "http://localhost:3333/items?sort=name&dir=asc"
	// Item 2 (Cloak of Whispers, a draft); item 1 is the protected one
	// whose refused delete feedback_test.go covers.
	rec := do(h, "DELETE", "/items/2", nil, headers)
	if rec.Code != 200 {
		t.Fatalf("DELETE /items/2 = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body, `id="items-results"`, "1–13 of 13 items", `aria-sort="ascending"`)
	wantNone(t, body, "Cloak of Whispers", "<h1", "<html")
	inOrder(t, body, "Amulet of Still Water", "Ashen Dagger", "Blade of the Fallen Star", "Boots of Quiet Steps")
	if _, ok := Items.Get("2"); ok {
		t.Error("item 2 still in the store")
	}
	// The response itself carries the live intro and the bar's hidden
	// state out of band, and the address bar follows the served state.
	inOrder(t, body, `id="items-results"`,
		`<select id="items-filterbar-hidden-sort" name="sort" data-tui-filterbar-control data-tui-filterbar-state hx-swap-oob="true"`,
		`<option value="name" selected data-tui-filterbar-set-dir="asc">Name ↑</option>`,
		`<p id="items-intro" class="text-sm text-muted-foreground" hx-swap-oob="true">13 items · 11 published · 2 drafts</p>`)
	if got := rec.Header().Get("HX-Replace-Url"); got != "/items?dir=asc&sort=name" {
		t.Errorf("HX-Replace-Url = %q", got)
	}
	// And the next full render agrees.
	wantAll(t, do(h, "GET", "/items", nil, nil).Body.String(), "13 items · 11 published · 2 drafts")

	// From the detail page (htmx, no results target): redirect to the list.
	rec = do(h, "DELETE", "/items/3", nil, hx(""))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/items" || rec.Body.Len() != 0 {
		t.Errorf("detail delete = %d HX-Redirect %q body %q", rec.Code, rec.Header().Get("HX-Redirect"), rec.Body.String())
	}
	// Plain request: 303.
	if rec = do(h, "DELETE", "/items/4", nil, nil); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/items" {
		t.Errorf("plain delete = %d Location %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec = do(h, "DELETE", "/items/999", nil, hx("items-results")); rec.Code != 404 {
		t.Errorf("unknown delete = %d, want 404", rec.Code)
	}
}

func TestItems_Publish(t *testing.T) {
	h := itemsHandler(t)
	headers := hx("items-results")
	headers["HX-Current-URL"] = "http://localhost:3333/items?status=draft"
	rec := do(h, "POST", "/items/publish", url.Values{"ids": {"2", "6"}}, headers)
	if rec.Code != 200 {
		t.Fatalf("POST /items/publish = %d", rec.Code)
	}
	body := rec.Body.String()
	// The draft view now holds the one remaining draft; the intro follows.
	wantAll(t, body, `id="items-results"`, "Helm of the Last Watch", "1–1 of 1 items",
		`<p id="items-intro" class="text-sm text-muted-foreground" hx-swap-oob="true">14 items · 13 published · 1 draft</p>`)
	wantNone(t, body, "Cloak of Whispers", "Amulet of Still Water", "<h1")
	// The default sort is omitted, as the page links omit it.
	if got := rec.Header().Get("HX-Replace-Url"); got != "/items?status=draft" {
		t.Errorf("HX-Replace-Url = %q", got)
	}
	for _, id := range []string{"2", "6"} {
		if it, _ := Items.Get(id); it.Status != "published" {
			t.Errorf("item %s status = %q, want published", id, it.Status)
		}
	}
	wantAll(t, do(h, "GET", "/items", nil, nil).Body.String(), "14 items · 13 published · 1 draft")
	// No ids: nothing changes, results still render.
	rec = do(h, "POST", "/items/publish", url.Values{}, headers)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `id="items-results"`) {
		t.Errorf("empty publish = %d", rec.Code)
	}
}

func TestItems_DetailAndHistory(t *testing.T) {
	h := itemsHandler(t)
	rec := do(h, "GET", "/items/1", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /items/1 = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-shell--sidebar", "Blade of the Fallen Star — Webframe</title>",
		"← Items", `href="/items"`, "Published",
		`href="/items/1/edit"`, `hx-delete="/items/1"`, `hx-confirm="Delete this item?"`,
		"<dt", "Type", "Weapon", "Starfall Pack", "Updated", "2d ago", `>1<`,
		"Notes", "Seeded from the 2026-10-21 dashboard sketch.",
		"History", `hx-get="/items/1/history"`, `hx-trigger="click once"`, `hx-target="#history-panel"`,
		`id="history-panel"`,
	)
	// Items stays the active sidebar item on a detail page (Match).
	if !strings.Contains(body, `aria-current="page"`) {
		t.Error("no active sidebar item on /items/1")
	}
	wantNone(t, body, "Status set to")

	// The history fragment: three dated lines, no shell.
	rec = do(h, "GET", "/items/1/history", nil, hx("history-panel"))
	if rec.Code != 200 {
		t.Fatalf("GET /items/1/history = %d", rec.Code)
	}
	body = rec.Body.String()
	if n := strings.Count(body, "<dt"); n != 3 {
		t.Errorf("history has %d <dt, want 3", n)
	}
	wantAll(t, body, "Updated", "Status set to published", "Created")
	wantNone(t, body, "<html", "<h1", "wf-sidebar")

	// An item with no pack shows the muted dash; unknown ids are 404.
	wantAll(t, do(h, "GET", "/items/2", nil, nil).Body.String(), "—", "Draft")
	for _, path := range []string{"/items/999", "/items/999/edit", "/items/999/history"} {
		if rec := do(h, "GET", path, nil, nil); rec.Code != 404 {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
	// The two pages answer with the error page in the sidebar shell; the
	// history fragment stays a bare 404 (a tab panel, not a page).
	for _, path := range []string{"/items/999", "/items/999/edit"} {
		wantAll(t, do(h, "GET", path, nil, nil).Body.String(), "Page not found", "wf-shell--sidebar", `>404<`)
	}
	// The create form.
	body = do(h, "GET", "/items/new", nil, nil).Body.String()
	wantAll(t, body, "New item — Webframe</title>", "Create item", "Cancel", `href="/items"`, "Details", "Notes", "Remixable", `role="switch"`, `hx-post="/items/validate"`)
	wantNone(t, body, "Please fix")
}

func TestItems_SidebarRedirect(t *testing.T) {
	h := itemsHandler(t)
	rec := do(h, "GET", "/sidebar", nil, nil)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/items" {
		t.Errorf("GET /sidebar = %d Location %q, want 302 /items", rec.Code, rec.Header().Get("Location"))
	}
}

func TestItems_RelativeTime(t *testing.T) {
	now := time.Now()
	for _, c := range []struct {
		age  time.Duration
		want string
	}{
		{0, "now"}, {5 * time.Minute, "5m"}, {3 * time.Hour, "3h"}, {2 * day, "2d"},
		{6 * day, "6d"}, {week, "1w"}, {3 * week, "3w"}, {month, "1mo"}, {2 * month, "2mo"}, {400 * day, "1y"},
	} {
		if got := relativeTime(now.Add(-c.age), now); got != c.want {
			t.Errorf("relativeTime(%v) = %q, want %q", c.age, got, c.want)
		}
	}
}
