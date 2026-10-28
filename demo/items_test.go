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
	wantAll(t, body, `id="items-results"`, "Blade of the Fallen Star", "1–14 of 14 items")
	wantNone(t, body, "<h1", "<html", "wf-topbar", "wf-sidebar", "Search items")
	// Any other htmx target gets the page body without the shell.
	body = do(h, "GET", "/items", nil, hx("main")).Body.String()
	wantAll(t, body, "<h1", "Search items")
	wantNone(t, body, "<html", "wf-sidebar")
	// A boosted request gets the whole document.
	body = do(h, "GET", "/items", nil, map[string]string{"HX-Request": "true", "HX-Boosted": "true"}).Body.String()
	wantAll(t, body, "<!doctype", "wf-shell--sidebar")
}

func TestItems_Delete(t *testing.T) {
	h := itemsHandler(t)
	headers := hx("items-results")
	headers["HX-Current-URL"] = "http://localhost:3333/items?sort=name&dir=asc"
	rec := do(h, "DELETE", "/items/1", nil, headers)
	if rec.Code != 200 {
		t.Fatalf("DELETE /items/1 = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body, `id="items-results"`, "1–13 of 13 items", `aria-sort="ascending"`)
	wantNone(t, body, "Blade of the Fallen Star", "<h1", "<html")
	inOrder(t, body, "Amulet of Still Water", "Ashen Dagger", "Boots of Quiet Steps")
	if _, ok := Items.Get("1"); ok {
		t.Error("item 1 still in the store")
	}
	// The intro line on the next full render is live.
	wantAll(t, do(h, "GET", "/items", nil, nil).Body.String(), "13 items · 10 published · 3 drafts")

	// From the detail page (htmx, no results target): redirect to the list.
	rec = do(h, "DELETE", "/items/2", nil, hx(""))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/items" || rec.Body.Len() != 0 {
		t.Errorf("detail delete = %d HX-Redirect %q body %q", rec.Code, rec.Header().Get("HX-Redirect"), rec.Body.String())
	}
	// Plain request: 303.
	if rec = do(h, "DELETE", "/items/3", nil, nil); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/items" {
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
	// The draft view now holds the one remaining draft.
	wantAll(t, body, `id="items-results"`, "Helm of the Last Watch", "1–1 of 1 items")
	wantNone(t, body, "Cloak of Whispers", "Amulet of Still Water", "<h1")
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
