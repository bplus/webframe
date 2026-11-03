package demo

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bplus/webframe/assets"
	"github.com/bplus/webframe/components/commandpalette"
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

// mainContent is body without the command palette the shell renders at
// the end of every page: the palette lists every item's name, so a test
// that asserts an item is absent from a filtered list looks here.
func mainContent(body string) string {
	if i := strings.Index(body, `id="`+commandpalette.DefaultID+`"`); i >= 0 {
		return body[:i]
	}
	return body
}

// doMultipart issues one multipart POST through h: fields as ordinary
// parts and, when fileField is set, one file part with the given name and
// bytes.
func doMultipart(h http.Handler, target string, fields url.Values, fileField, fileName string, file []byte, headers map[string]string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, vs := range fields {
		for _, v := range vs {
			if err := mw.WriteField(k, v); err != nil {
				panic(err)
			}
		}
	}
	if fileField != "" {
		fw, err := mw.CreateFormFile(fileField, fileName)
		if err != nil {
			panic(err)
		}
		if _, err := io.Copy(fw, bytes.NewReader(file)); err != nil {
			panic(err)
		}
	}
	if err := mw.Close(); err != nil {
		panic(err)
	}
	req := httptest.NewRequest("POST", target, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// pngBytes is the eight-byte PNG signature plus a few bytes: enough for
// http.DetectContentType to call it image/png.
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

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
	wantNone(t, mainContent(body), "Blade of the Fallen Star", "Ashen Dagger")
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
		`hx-delete="/items/2"`, `hx-confirm="This cannot be undone."`, `data-wf-confirm-destructive="true"`, `href="/items/2/edit"`,
	)
	// The default list is newest first with every row.
	body = do(h, "GET", "/items", nil, nil).Body.String()
	inOrder(t, body, "Blade of the Fallen Star", "Cloak of Whispers", "Mantle of Ember")
	wantAll(t, body, "1–14 of 14 items", `aria-sort="descending"`)
	// A search narrows by name, case-insensitively.
	body = do(h, "GET", "/items?q=OF+THE", nil, nil).Body.String()
	wantAll(t, body, "Blade of the Fallen Star", "Pendant of the Weave")
	wantNone(t, mainContent(body), "Ashen Dagger")
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
	// The redirect carries the "Saved" flash for the next page, not a body.
	if strings.Contains(rec.Body.String(), "Saved") {
		t.Error("redirect response should not render the flash itself")
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
	// Saving an unknown id is the 404 page in the sidebar shell, like the
	// edit form's own GET.
	rec = do(h, "POST", "/items/999", url.Values{"name": {"Ghost"}, "type": {"armor"}, "status": {"draft"}}, nil)
	if rec.Code != 404 {
		t.Errorf("POST /items/999 = %d, want 404", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Page not found", "wf-shell--sidebar", `>404<`)
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
		`href="/items/1/edit"`, `hx-delete="/items/1"`, `hx-confirm="This cannot be undone."`, `data-wf-confirm-destructive="true"`,
		"<dt", "Type", "Weapon", "Starfall Pack", "Updated", "2d ago", `>1<`,
		"Notes", "Seeded demo data.",
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

// TestItems_FormWidgets: the form carries the tags, rating and date
// widgets, a POST with all three round-trips (422 keeps every value as
// chips, stars and the filled date; success stores them), and the detail
// page shows them.
func TestItems_FormWidgets(t *testing.T) {
	h := itemsHandler(t)
	body := do(h, "GET", "/items/new", nil, nil).Body.String()
	wantAll(t, body,
		`enctype="multipart/form-data"`,
		// Tags: the hidden field, the suggestion datalist, the text input.
		`name="tags"`, `id="field-tags-options"`, `<option value="legendary">`, `placeholder="Add a tag"`, `id="field-tags-root"`, `data-max="5"`, "0/5",
		// Release date with today as max; rating radios 1..5; the file drop.
		`type="date"`, `name="released"`,
		`name="rating"`, `id="field-rating-5"`, "Leave every star empty for unrated.",
		`type="file"`, `name="artwork"`, `accept="image/*"`, "Drop a file here or choose",
		"Release", "Plain text; a blank line starts a new paragraph.",
	)
	wantNone(t, body, `id="field-rating-6"`)
	// The max is today, computed by the handler from its own time.Now();
	// asserting the exact date would flake across midnight, so check the
	// shape (an ISO date in this century) on the released field's input.
	if m := regexp.MustCompile(`name="released"[^>]*\bmax="(20\d\d-\d\d-\d\d)"`).FindStringSubmatch(body); m == nil {
		t.Errorf("released max is not an ISO date in:\n%s", body)
	}

	// Errors: a future date and a rating out of range, with the tags kept.
	tomorrow := time.Now().Add(48 * time.Hour).Format("2006-01-02")
	form := url.Values{"name": {"Tidecaller Spear"}, "type": {"weapon"}, "status": {"draft"}, "tags": {"water, reach ,water"}, "released": {tomorrow}, "rating": {"9"}, "notes": {"keep me"}}
	rec := do(h, "POST", "/items", form, hx("item-form"))
	if rec.Code != 422 {
		t.Fatalf("POST /items with a future date = %d, want 422", rec.Code)
	}
	body = rec.Body.String()
	wantAll(t, body,
		"Please fix the 2 errors below.", "The release date cannot be in the future", "Rating must be 0 to 5",
		// The chips come back from the submitted value, the date is echoed.
		`data-tag="water"`, `data-tag="reach"`, `value="`+tomorrow+`"`, "keep me",
	)
	if total, _, _ := Items.Counts(); total != 14 {
		t.Errorf("store changed on an invalid submit: %d items", total)
	}
	// A rating that is not a number and a date that is not a date.
	form.Set("released", "yesterday")
	form.Set("rating", "lots")
	body = do(h, "POST", "/items", form, hx("item-form")).Body.String()
	wantAll(t, body, "Enter the release date as YYYY-MM-DD", "Rating must be 0 to 5")
	// More tags than the cap (Field.MaxTags stops the user in the widget;
	// the validator stops a handcrafted request).
	form.Set("tags", "a,b,c,d,e,f")
	wantAll(t, do(h, "POST", "/items", form, hx("item-form")).Body.String(), "At most 5 tags")
	// Blur validation knows the two fields too.
	body = do(h, "POST", "/items/validate", url.Values{"rating": {"7"}}, hx("")).Body.String()
	wantAll(t, body, `id="field-rating-feedback"`, "Rating must be 0 to 5")
	body = do(h, "POST", "/items/validate", url.Values{"released": {"2026-01-05"}}, hx("")).Body.String()
	wantNone(t, body, "<p")

	// Success: tags deduplicated case-insensitively and trimmed, the
	// rating and date parsed, then shown on the detail page.
	form = url.Values{"name": {"Tidecaller Spear"}, "type": {"weapon"}, "status": {"published"}, "tags": {"water, Reach ,WATER,,"}, "released": {"2026-01-05"}, "rating": {"4"}, "notes": {"First line.\n\nSecond paragraph."}}
	rec = do(h, "POST", "/items", form, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/items/15" {
		t.Fatalf("POST /items = %d Location %q, want 303 /items/15", rec.Code, rec.Header().Get("Location"))
	}
	it, ok := Items.Get("15")
	if !ok || strings.Join(it.Tags, "|") != "water|Reach" || it.Rating != 4 || it.Released.Format("2006-01-02") != "2026-01-05" || it.Artwork != "" {
		t.Errorf("stored item = %+v", it)
	}
	body = do(h, "GET", "/items/15", nil, nil).Body.String()
	wantAll(t, body,
		">water<", ">Reach<", // the tag badges
		`id="item-rating"`, `aria-label="Rating: 4 of 5"`, `role="img"`,
		"Jan 5, 2026", "Artwork", "—",
		// Notes through prose.Text: two paragraphs.
		`id="item-notes"`, "wf-prose", "<p>First line.</p>", "<p>Second paragraph.</p>",
	)
	// The edit form carries them back in the widgets' own shapes.
	body = do(h, "GET", "/items/15/edit", nil, nil).Body.String()
	wantAll(t, body, `value="water,Reach"`, `data-tag="Reach"`, `value="2026-01-05"`, `id="field-rating-4" name="rating" value="4" class="sr-only" checked`)
	// An item with none of them shows the muted dashes and "Not rated".
	body = do(h, "GET", "/items/6", nil, nil).Body.String()
	wantAll(t, body, "Not rated", "Tags")
	wantNone(t, body, `id="item-rating"`)
	// A seeded item shows its tags and stars.
	body = do(h, "GET", "/items/1", nil, nil).Body.String()
	wantAll(t, body, ">legendary<", ">fire<", ">two-handed<", `aria-label="Rating: 5 of 5"`, "Sep 12, 2026")
}

// TestItems_MultipartArtwork: a multipart POST with a small image stores
// its file name, a non-image is refused under the artwork field, and an
// edit without a file keeps the stored name.
func TestItems_MultipartArtwork(t *testing.T) {
	h := itemsHandler(t)
	fields := url.Values{"name": {"Painted Shield"}, "type": {"armor"}, "status": {"draft"}, "tags": {"art"}, "rating": {"3"}, "notes": {"Has a picture."}}
	rec := doMultipart(h, "/items", fields, "artwork", "shield.png", pngBytes, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/items/15" {
		t.Fatalf("multipart POST /items = %d Location %q, want 303 /items/15", rec.Code, rec.Header().Get("Location"))
	}
	it, _ := Items.Get("15")
	if it.Artwork != "shield.png" || it.Rating != 3 || strings.Join(it.Tags, ",") != "art" || it.Notes != "Has a picture." {
		t.Errorf("stored item = %+v", it)
	}
	body := do(h, "GET", "/items/15", nil, nil).Body.String()
	wantAll(t, body, "shield.png")
	// The edit form names the current file; a save with no file keeps it.
	body = do(h, "GET", "/items/15/edit", nil, nil).Body.String()
	wantAll(t, body, "Current file: shield.png")
	fields.Set("name", "Painted Shield II")
	rec = doMultipart(h, "/items/15", fields, "", "", nil, hx("item-form"))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/items/15" {
		t.Errorf("multipart edit without a file = %d HX-Redirect %q", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if it, _ := Items.Get("15"); it.Artwork != "shield.png" || it.Name != "Painted Shield II" {
		t.Errorf("edited item = %+v", it)
	}
	// A text file is not artwork: 422 with the error under the field, the
	// other values kept, nothing stored.
	fields.Set("name", "Scroll of Text")
	rec = doMultipart(h, "/items", fields, "artwork", "notes.txt", []byte("just some words\n"), hx("item-form"))
	if rec.Code != 422 {
		t.Fatalf("multipart POST with a text file = %d, want 422", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Please fix the 1 error below.", "Artwork must be an image", `value="Scroll of Text"`, `data-tag="art"`)
	if total, _, _ := Items.Counts(); total != 15 {
		t.Errorf("store changed on a refused file: %d items", total)
	}
	// A multipart POST with no file part at all is an ordinary create.
	fields.Set("name", "Plain Buckler")
	rec = doMultipart(h, "/items", fields, "", "", nil, nil)
	if rec.Code != http.StatusSeeOther {
		t.Errorf("multipart POST without a file = %d, want 303", rec.Code)
	}
	if it, _ := Items.Get("16"); it.Artwork != "" || it.Name != "Plain Buckler" {
		t.Errorf("item without artwork = %+v", it)
	}
}

// TestItems_HistoryDiffAndJSON: the History tab fragment adds a diff of the
// notes against their previous version and the record as a JSON tree.
func TestItems_HistoryDiffAndJSON(t *testing.T) {
	h := itemsHandler(t)
	rec := do(h, "GET", "/items/1/history", nil, hx("history-panel"))
	if rec.Code != 200 {
		t.Fatalf("GET /items/1/history = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		// The dated lines are still there.
		"Status set to published", "Created",
		// The diff: one file, inline, a deletion and an addition of the
		// status line, the seed's first line as context.
		`id="item-notes-diff"`, `data-mode="inline"`, "items/1/notes.txt", "@@ -1,3 +1,4 @@",
		`data-kind="context"`, `data-kind="del"`, `data-kind="add"`, "Status: draft", "Status: published", "TODO: describe where it drops.",
		// The JSON tree with the item's keys and the copy-path buttons.
		`id="item-json"`, `&#34;name&#34;:`, "Blade of the Fallen Star", `&#34;tags&#34;:`, "legendary", `&#34;rating&#34;:`, "$.tags",
	)
	wantNone(t, body, "<html", "<h1", "wf-sidebar", "0001-01-01")
	// A draft with no release date leaves the key out (omitzero).
	body = do(h, "GET", "/items/2/history", nil, hx("history-panel")).Body.String()
	wantNone(t, body, `&#34;released&#34;:`)
	wantAll(t, body, "Status: draft", `data-kind="add"`)
}

func TestItems_DiffLines(t *testing.T) {
	lines := diffLines([]string{"a", "b", "c"}, []string{"a", "x", "c", "d"})
	var got []string
	for _, l := range lines {
		got = append(got, string(l.Kind)+":"+l.Text)
	}
	if want := ":a del:b add:x :c add:d"; strings.Join(got, " ") != want {
		t.Errorf("diffLines = %q, want %q", strings.Join(got, " "), want)
	}
	if got := hunkHeader(lines); got != "@@ -1,3 +1,4 @@" {
		t.Errorf("hunkHeader = %q", got)
	}
	if got := diffLines(nil, nil); len(got) != 0 {
		t.Errorf("empty diff has %d lines", len(got))
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

// TestItems_SaveFlashesOnce: a successful create sets a "Saved" flash that
// the detail page shows once (role="status" banner in #wf-flash) and a
// reload no longer shows.
func TestItems_SaveFlashesOnce(t *testing.T) {
	h := itemsHandler(t)
	form := url.Values{"name": {"Flash Probe"}, "type": {"weapon"}, "status": {"draft"}}
	rec := do(h, "POST", "/items", form, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /items = %d, want 303", rec.Code)
	}
	cookies := rec.Result().Cookies()
	var flashCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "wf_flash" {
			flashCookie = c
		}
	}
	if flashCookie == nil {
		t.Fatalf("no wf_flash cookie set on save: %v", cookies)
	}
	req := httptest.NewRequest("GET", rec.Header().Get("Location"), nil)
	req.AddCookie(flashCookie)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	body := rec2.Body.String()
	slot := strings.Index(body, `<div id="wf-flash">`)
	if slot < 0 || !strings.Contains(body[slot:], `role="status"`) || !strings.Contains(body, "Flash Probe was created.") {
		t.Errorf("detail page after save lacks the Saved flash:\n%s", body[:min(600, len(body))])
	}
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, httptest.NewRequest("GET", rec.Header().Get("Location"), nil))
	if strings.Contains(rec3.Body.String(), "Flash Probe was created.") {
		t.Errorf("reload still shows the flash")
	}
}
