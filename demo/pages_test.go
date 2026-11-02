package demo

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// pagesHandler is fullHandler with the account reset too, so every page
// test starts from the seeded store and the seeded account.
func pagesHandler(t *testing.T) http.Handler {
	t.Helper()
	Settings = NewAccount()
	return fullHandler(t)
}

func TestPages_Dashboard(t *testing.T) {
	h := pagesHandler(t)
	rec := do(h, "GET", "/dashboard", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /dashboard = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-shell--sidebar", "Dashboard — Webframe</title>", `aria-current="page"`,
		`id="dashboard"`, "wf-dashgrid",
		// The four tiles from the store: labels and values.
		">Items<", ">14<", "11 published · 3 drafts", `href="/items"`,
		">Published<", ">11<", `href="/items?status=published"`,
		">Drafts<", ">3<", `href="/items?status=draft"`,
		">Packs<", ">2<", "Starfall Pack · Netherese Relics",
		// The two sections and their components.
		`id="recent-changes"`, "Recent changes", `id="recent-changes-timeline"`, "<ol", "<time datetime=", "Blade of the Fallen Star updated", "Published weapon · 2d ago", "In Starfall Pack", "bg-success",
		`id="newest-items"`, "Newest items", `id="newest-items-grid"`, "sm:grid-cols-2", `src="/static/img/placeholder-wide.svg"`, `href="/items/1/edit"`, "Updated 2d ago", "Not in a pack",
	)
	// Six slots: four tiles and two sections.
	if n := strings.Count(body, "wf-dashslot"); n != 6 {
		t.Errorf("wf-dashslot appears %d times, want 6", n)
	}
	// Four cards, the newest items in order; the timeline holds eight events.
	inOrder(t, body, `id="card-item-1"`, `id="card-item-2"`, `id="card-item-3"`, `id="card-item-4"`)
	// Two columns at most in the 8-column section.
	wantNone(t, body, `id="card-item-5"`, "lg:grid-cols-3", "xl:grid-cols-4")
	if n := strings.Count(body, " updated</a>"); n != dashboardRecent {
		t.Errorf("timeline has %d events, want %d", n, dashboardRecent)
	}
	// The dashboard follows the store.
	Items.Publish([]string{"2", "6", "11"})
	body = do(h, "GET", "/dashboard", nil, nil).Body.String()
	wantAll(t, body, "14 published · 0 drafts", ">0<")
	// An empty store shows both empty states, no grid of cards.
	Items = &ItemStore{next: 1}
	body = do(h, "GET", "/dashboard", nil, nil).Body.String()
	wantAll(t, body, "No changes yet", "No items yet", "+ Create item", "No item is in a pack")
	wantNone(t, body, `id="card-item-`)
}

func TestPages_DashboardHelpers(t *testing.T) {
	now := time.Now()
	items := newestFirst(NewItemStore(now).All())
	if items[0].Name != "Blade of the Fallen Star" || items[len(items)-1].Name != "Mantle of Ember" {
		t.Errorf("newestFirst order: %s … %s", items[0].Name, items[len(items)-1].Name)
	}
	if got := packNames(items); len(got) != 2 || got[0] != "Starfall Pack" || got[1] != "Netherese Relics" {
		t.Errorf("packNames = %v", got)
	}
	if got := packsLine(nil); got != "No item is in a pack" {
		t.Errorf("packsLine(nil) = %q", got)
	}
}

func TestPages_SettingsForm(t *testing.T) {
	h := pagesHandler(t)
	rec := do(h, "GET", "/settings", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /settings = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-shell--sidebar", "Settings — Webframe</title>", `aria-current="page"`,
		`id="settings"`, `hx-post="/settings"`, `hx-target="#settings"`, `hx-swap="outerHTML"`, `action="/settings"`,
		// Groups, fields and the stored values.
		`id="profile"`, "Profile", `name="display_name"`, `value="Morgan"`, `name="email"`, `type="email"`, `value="morgan@example.com"`,
		`id="notifications"`, "Notifications", `role="switch"`, `name="digest"`, `name="mentions"`,
		// The save bar, always present, with both buttons; the submit button
		// sits outside the form and names it.
		"wf-settings-bar", "sticky bottom-0", "Save changes", `type="submit"`, `form="settings"`, "Cancel", `href="/dashboard"`,
		// The Danger zone after the form: a destructive action that confirms.
		`id="danger-zone"`, "Danger zone", "Delete account", `hx-post="/settings/delete-account"`,
		`hx-confirm="Every item and pack you own goes with it. This cannot be undone."`, `data-wf-confirm-title="Delete your account?"`,
		`data-wf-confirm-label="Delete account"`, `data-wf-confirm-destructive="true"`,
	)
	wantNone(t, body, "Please fix", `aria-invalid="true"`)
	// The digest toggle is on, mentions off.
	inOrder(t, body, `role="switch" checked id="field-digest"`, `role="switch" id="field-mentions"`)
	// The Environment group is the form's last section (its editor is the
	// group's Content, posted by the Save button); the Danger zone comes
	// after the form and before the save bar, which is the page's last child
	// so it stays stuck across both. One form.
	inOrder(t, body, `id="settings"`, `id="notifications"`, `id="environment"`, `name="env.key"`, "</form>", `id="danger-zone"`, "wf-settings-bar")
	if n := strings.Count(body, "</form>"); n != 1 {
		t.Errorf("%d forms, want 1 (the Environment editor is inside the settings form)", n)
	}
}

// TestPages_SettingsTimeAndEnv: the quiet-hours time field round-trips through
// the settings form, and the Environment editor (the Environment group's
// Content, inside the settings form) saves its rows with the fields through
// the one Save button (POST /settings).
func TestPages_SettingsTimeAndEnv(t *testing.T) {
	h := pagesHandler(t)
	body := do(h, "GET", "/settings", nil, nil).Body.String()
	wantAll(t, body,
		// The time field with the stored value.
		"Quiet hours start", `type="time"`, `name="quiet_start"`, `value="22:00"`,
		// The Environment editor inside the form: the two rows as
		// env.key/env.value pairs, the cap, the add button.
		`id="environment"`, "Environment",
		`name="env.key"`, `name="env.value"`, `value="REGION"`, `value="eu-west"`, `value="LOG_LEVEL"`, `value="info"`, `data-max="8"`, "Add variable",
	)
	// No form or save button of its own, no environment route.
	wantNone(t, body, `id="environment-form"`, "/settings/environment", "Save environment")
	inOrder(t, body, `id="settings"`, `id="environment"`, `name="env.key"`, "</form>", `id="danger-zone"`)
	// A bad time is an error; the display name rule still applies; the
	// submitted rows come back in the re-rendered form and nothing is stored.
	rec := do(h, "POST", "/settings", url.Values{"display_name": {"Morgan"}, "quiet_start": {"ten pm"}, "env.key": {"TYPED"}, "env.value": {"kept"}}, hx("settings"))
	if rec.Code != 422 {
		t.Fatalf("POST /settings with a bad time = %d, want 422", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Enter the time as HH:MM", `value="ten pm"`, `value="TYPED"`, `value="kept"`)
	if rows := Settings.Env(); len(rows) != 2 || rows[0].Key != "REGION" {
		t.Errorf("env rows changed on an invalid submit: %+v", rows)
	}
	// A good time saves, and the rows with it: they zip in order, a blank
	// key is dropped, and an htmx submit goes back to the page (HX-Redirect),
	// which shows the saved time and rows.
	rec = do(h, "POST", "/settings", url.Values{"display_name": {"Morgan"}, "quiet_start": {"23:30"}, "env.key": {"REGION", "  ", "DEBUG"}, "env.value": {"us-east", "ignored", "1"}}, hx(""))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/settings" || rec.Body.Len() != 0 {
		t.Fatalf("htmx POST /settings = %d HX-Redirect %q body %q", rec.Code, rec.Header().Get("HX-Redirect"), rec.Body.String())
	}
	if got := Settings.Values()["quiet_start"]; got != "23:30" {
		t.Errorf("quiet_start = %q, want 23:30", got)
	}
	if rows := Settings.Env(); len(rows) != 2 || rows[0].Key != "REGION" || rows[0].Value != "us-east" || rows[1].Key != "DEBUG" || rows[1].Value != "1" {
		t.Errorf("env rows = %+v", rows)
	}
	body = do(h, "GET", "/settings", nil, nil).Body.String()
	wantAll(t, body, `value="23:30"`, `value="us-east"`, `value="DEBUG"`)
	wantNone(t, body, `value="LOG_LEVEL"`, "Environment saved")
	// A plain POST: 303 back to the page; more than eight rows are cut.
	form := url.Values{"display_name": {"Morgan"}, "quiet_start": {"23:30"}}
	for i := 0; i < 10; i++ {
		form.Add("env.key", "K"+strconv.Itoa(i))
		form.Add("env.value", "v")
	}
	if rec = do(h, "POST", "/settings", form, nil); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings" {
		t.Errorf("plain POST /settings = %d Location %q", rec.Code, rec.Header().Get("Location"))
	}
	if rows := Settings.Env(); len(rows) != maxEnvRows {
		t.Errorf("%d env rows, want %d", len(rows), maxEnvRows)
	}
}

func TestPages_SettingsErrors(t *testing.T) {
	h := pagesHandler(t)
	form := url.Values{"display_name": {"   "}, "email": {"new@example.com"}, "mentions": {"on"}}
	rec := do(h, "POST", "/settings", form, hx("settings"))
	if rec.Code != 422 {
		t.Fatalf("POST /settings with a blank name = %d, want 422", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(strings.TrimSpace(body), "<form") {
		t.Errorf("form swap does not start with <form:\n%s", body[:min(200, len(body))])
	}
	wantAll(t, body,
		"Please fix the 1 error below.", "Display name is required", `aria-invalid="true"`,
		// Every submitted value kept: the typed email, mentions now on, digest off.
		`value="new@example.com"`, `role="switch" checked id="field-mentions"`, `role="switch" id="field-digest"`,
	)
	// The part is the form alone: the save bar and the Danger zone stay on
	// the page untouched.
	wantNone(t, body, "<h1", "<html", `value="Morgan"`, "Danger zone", "wf-settings-bar", "Save changes", "Cancel")
	if n := strings.Count(body, `aria-invalid="true"`); n != 1 {
		t.Errorf("aria-invalid count = %d, want 1", n)
	}
	if rec.Header().Get("HX-Redirect") != "" || rec.Header().Get("Location") != "" {
		t.Error("error re-render must not redirect")
	}
	if got := Settings.Values()["display_name"]; got != "Morgan" {
		t.Errorf("store changed on an invalid submit: display_name = %q", got)
	}
	// A plain (non-htmx) POST with errors renders the whole document, still 422.
	rec = do(h, "POST", "/settings", url.Values{"display_name": {""}}, nil)
	if rec.Code != 422 {
		t.Errorf("plain POST /settings with errors = %d, want 422", rec.Code)
	}
	wantAll(t, rec.Body.String(), "<!doctype", "Please fix the 1 error below.", "Danger zone", "wf-settings-bar")
	inOrder(t, rec.Body.String(), "</form>", "Danger zone", "wf-settings-bar")
}

func TestPages_SettingsSaveRedirects(t *testing.T) {
	h := pagesHandler(t)
	form := url.Values{"display_name": {"  Morgan Reyes "}, "email": {"k@example.com"}, "digest": {""}, "mentions": {"on"}}
	rec := do(h, "POST", "/settings", form, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings" {
		t.Fatalf("POST /settings = %d Location %q, want 303 /settings", rec.Code, rec.Header().Get("Location"))
	}
	if body := rec.Body.String(); strings.Contains(body, `role="status"`) {
		t.Error("redirect response should not render a banner itself")
	}
	vals := Settings.Values()
	if vals["display_name"] != "Morgan Reyes" || vals["email"] != "k@example.com" || vals["digest"] != "" || vals["mentions"] != "on" {
		t.Errorf("stored values = %v", vals)
	}
	// The next render shows the saved values and nothing else.
	body := do(h, "GET", "/settings", nil, nil).Body.String()
	wantAll(t, body, `value="Morgan Reyes"`, `value="k@example.com"`, `role="switch" checked id="field-mentions"`, `role="switch" id="field-digest"`)
	// No success message anywhere: the flash slot is empty, no banner. The
	// one role="status" on the page is the command palette's own "No
	// matches" line, which every page carries.
	wantAll(t, body, `<div id="wf-flash"></div>`)
	wantNone(t, body, "Please fix", `role="alert"`, "Settings saved")
	if n := strings.Count(body, `role="status"`); n != 1 {
		t.Errorf("%d role=status regions, want 1 (the palette's)", n)
	}

	// An htmx submit gets HX-Redirect with an empty 200.
	form.Set("display_name", "Morgan")
	rec = do(h, "POST", "/settings", form, hx("settings"))
	if rec.Code != 200 || rec.Header().Get("HX-Redirect") != "/settings" {
		t.Errorf("htmx POST /settings = %d HX-Redirect %q, want 200 /settings", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if rec.Body.Len() != 0 {
		t.Errorf("htmx redirect body = %q, want empty", rec.Body.String())
	}
}

func TestPages_DeleteAccountRefused(t *testing.T) {
	h := pagesHandler(t)
	// Over htmx (the confirmed Danger zone button): a 409 whose only body is
	// the error toast, so nothing on the page changes.
	rec := do(h, "POST", "/settings/delete-account", url.Values{}, hx(""))
	if rec.Code != http.StatusConflict || rec.Header().Get("HX-Redirect") != "" {
		t.Fatalf("htmx POST /settings/delete-account = %d HX-Redirect %q, want 409 and none", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	body := rec.Body.String()
	wantAll(t, body, `hx-swap-oob="beforeend:#toast-container"`, `role="alert"`, "Could not delete the account")
	wantNone(t, body, "<html", "<h1", "<form")
	if strings.Count(body, "wf-toast") != 1 {
		t.Errorf("want exactly one toast, got %d", strings.Count(body, "wf-toast"))
	}
	// A plain request: an error flash and a 303 back to the settings page,
	// where the shell shows it once.
	rec = do(h, "POST", "/settings/delete-account", url.Values{}, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/settings" {
		t.Fatalf("plain POST /settings/delete-account = %d Location %q, want 303 /settings", rec.Code, rec.Header().Get("Location"))
	}
	c := flashCookie(t, rec)
	body = do(h, "GET", "/settings", nil, map[string]string{"Cookie": c.String()}).Body.String()
	wantAll(t, body, `<div id="wf-flash">`, `role="alert"`, "Could not delete the account", `value="Morgan"`)
}

func TestPages_BrowseList(t *testing.T) {
	h := pagesHandler(t)
	rec := do(h, "GET", "/browse", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /browse = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-shell--sidebar", "Browse — Webframe</title>", `aria-current="page"`,
		`id="browse"`, `id="browse-detail"`, `aria-label="Item details"`, `<nav aria-label="Items">`, `<ul id="browse-list"`,
		// Every item is a splitpane.List row: a link carrying LinkAttrs, the
		// name as the title, the type as the meta line.
		`href="/browse/1"`, `data-wf-splitpane-link="true"`, `href="/browse/14"`, `hx-target="#browse-detail"`, `hx-push-url="true"`,
		"Blade of the Fallen Star", `title="Blade of the Fallen Star"`, "Mantle of Ember", ">Weapon<", ">Accessory<", "text-muted-foreground",
		// Nothing selected: the placeholder in the detail region, no detail,
		// no back link.
		"Select an item",
	)
	wantNone(t, body, "<dl", "All items", "Open item")
	if n := strings.Count(body, `hx-target="#browse-detail"`); n != 14 {
		t.Errorf("%d list links carry the detail target, want 14", n)
	}
	// No item is marked current on the list route.
	if n := strings.Count(body, `aria-current="page"`); n != 1 {
		t.Errorf("aria-current appears %d times, want 1 (the sidebar's Browse entry)", n)
	}
	list := body[strings.Index(body, `<nav aria-label="Items">`):strings.Index(body, "</nav>")]
	wantNone(t, list, "bg-muted", "aria-current")
}

func TestPages_BrowseItem(t *testing.T) {
	h := pagesHandler(t)
	// A plain request (a phone's list click, a reload, a bookmark): the
	// whole document with the list, the detail and the phone back link.
	rec := do(h, "GET", "/browse/3", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /browse/3 = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-shell--sidebar", "Ring of Hoarded Light — Browse — Webframe</title>",
		`<ul id="browse-list"`, "Mantle of Ember",
		`id="browse-detail"`, "<dl", "Accessory", "Starfall Pack", ">Published<", "5d ago",
		"All items", `href="/browse"`, "md:hidden", `href="/items/3"`, "Open item",
	)
	// The shown item is the selected row: aria-current and bg-muted on its
	// link, and only on its link.
	sel := body[strings.LastIndex(body[:strings.Index(body, `href="/browse/3"`)], "<a"):]
	sel = sel[:strings.Index(sel, "</a>")]
	wantAll(t, sel, `aria-current="page"`, "bg-muted", "Ring of Hoarded Light")
	list := body[strings.Index(body, `<nav aria-label="Items">`):strings.Index(body, "</nav>")]
	if n := strings.Count(list, "bg-muted"); n != 1 {
		t.Errorf("bg-muted appears %d times in the list, want 1 (the selected row)", n)
	}
	if n := strings.Count(list, `aria-current="page"`); n != 1 {
		t.Errorf("aria-current appears %d times in the list, want 1", n)
	}
	// The list comes before the detail region, which holds the back link
	// and the description list.
	inOrder(t, body, `<ul id="browse-list"`, `id="browse-detail"`, "All items", "<dl")
	// Browse stays the active sidebar entry on a detail URL (Match).
	wantAll(t, body, `aria-current="page">Browse`)

	// A list click from md up (htmx targeting the detail region): the
	// detail region's content alone, back link first.
	rec = do(h, "GET", "/browse/3", nil, hx("browse-detail"))
	if rec.Code != 200 {
		t.Fatalf("htmx GET /browse/3 = %d", rec.Code)
	}
	body = rec.Body.String()
	// A <title> first (htmx applies it after the swap), then the back link.
	if !strings.HasPrefix(strings.TrimSpace(body), `<title>Ring of Hoarded Light — Browse — Webframe</title><div class="md:hidden mb-4">`) {
		t.Errorf("detail part does not start with the title then the back link:\n%s", body[:min(200, len(body))])
	}
	// The detail content, then the list out of band (the selected item
	// follows the pick on md and up); never the detail region or the shell.
	wantAll(t, body, "All items", "<dl", "Ring of Hoarded Light", "Starfall Pack", "Open item",
		`<div id="browse-listcol" hx-swap-oob="true">`, `<ul id="browse-list"`, `aria-current="page"`)
	wantNone(t, body, `id="browse-detail"`, "<html", "<h1", "wf-sidebar")
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Target") {
		t.Errorf("Vary %q lacks HX-Target", v)
	}
	// Any other htmx target gets the pane without the shell.
	body = do(h, "GET", "/browse/3", nil, hx("main")).Body.String()
	wantAll(t, body, `id="browse"`, `<ul id="browse-list"`, "<dl")
	wantNone(t, body, "<html", "wf-sidebar")
	// A boosted request gets the whole document.
	body = do(h, "GET", "/browse/3", nil, map[string]string{"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "browse-detail"}).Body.String()
	wantAll(t, body, "<!doctype", "wf-shell--sidebar", `<ul id="browse-list"`)
	// Unknown ids are the 404 page in the sidebar shell.
	rec = do(h, "GET", "/browse/999", nil, nil)
	if rec.Code != 404 {
		t.Errorf("GET /browse/999 = %d, want 404", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Page not found", "wf-shell--sidebar", `>404<`)
}

func TestPages_TrafficOnStreampage(t *testing.T) {
	h := pagesHandler(t)
	rec := do(h, "GET", "/traffic", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /traffic = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-topbar", "Traffic — Webframe</title>", `id="traffic"`, "<h1", "Traffic Stream",
		"A live operator surface", `/api/traffic/stream`, "trace-demo-001", "POST /v1/messages", "Completed upstream round-trip",
		"anthropic", "proxy.forward_end", "/static/js/trafficstream.min.js",
	)
	wantNone(t, body, "wf-sidebar")
	if n := strings.Count(body, "/static/js/trafficstream.min.js"); n != 1 {
		t.Errorf("trafficstream script loaded %d times, want 1", n)
	}
	// The stream props are the page's: 560px tall.
	if !strings.Contains(body, "560") {
		t.Error("stream height 560 not in the page")
	}
}

func TestPages_MethodNotAllowed(t *testing.T) {
	h := pagesHandler(t)
	// A GET-only page asked with POST: the styled 405 in the shell with the
	// Allow header the mux would have sent.
	rec := do(h, "POST", "/traffic", url.Values{}, nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /traffic = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("Allow = %q, want \"GET, HEAD\"", allow)
	}
	body := rec.Body.String()
	wantAll(t, body, "<!doctype", "wf-topbar", `>405<`, "Not allowed here", "This address does not accept that request.", "Go home", `href="/"`)
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	// An item that takes GET, POST and DELETE but not PUT; the Allow header
	// is the mux's own (errorpage.Handler copies it), which sorts the
	// methods.
	rec = do(h, "PUT", "/items/1", nil, nil)
	if rec.Code != 405 || rec.Header().Get("Allow") != "DELETE, GET, HEAD, POST" {
		t.Errorf("PUT /items/1 = %d Allow %q", rec.Code, rec.Header().Get("Allow"))
	}
	// Over htmx the 405 page is the body only, still with its status.
	rec = do(h, "POST", "/dashboard", url.Values{}, hx("main"))
	if rec.Code != 405 {
		t.Errorf("htmx POST /dashboard = %d", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Not allowed here")
	wantNone(t, rec.Body.String(), "<html")
	// A path nothing accepts is still the 404 page (the wrapper replaced the
	// catch-all), and real routes are untouched.
	rec = do(h, "POST", "/no-such-page", url.Values{}, nil)
	if rec.Code != 404 || rec.Header().Get("Allow") != "" {
		t.Errorf("POST /no-such-page = %d Allow %q, want 404 and none", rec.Code, rec.Header().Get("Allow"))
	}
	wantAll(t, rec.Body.String(), "Page not found", "wf-topbar")
	for _, path := range []string{"/", "/items", "/dashboard", "/browse", "/settings", "/traffic"} {
		if rec := do(h, "GET", path, nil, nil); rec.Code != 200 {
			t.Errorf("GET %s = %d", path, rec.Code)
		}
	}
	// The one method GET patterns also answer: HEAD.
	if rec := do(h, "HEAD", "/traffic", nil, nil); rec.Code != 200 {
		t.Errorf("HEAD /traffic = %d", rec.Code)
	}
}

func TestPages_BoomIsPlainText(t *testing.T) {
	h := pagesHandler(t)
	rec := do(h, "GET", "/api/boom", nil, hx("boom-result"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("GET /api/boom = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain (htmx-errors.js refuses to swap it)", ct)
	}
	body := rec.Body.String()
	wantAll(t, body, "boom")
	wantNone(t, body, "<", "wf-topbar")
}

// TestPages_ShowcaseSections pins each page-type section's id and the markup
// its demo relies on.
func TestPages_ShowcaseSections(t *testing.T) {
	h := pagesHandler(t)
	body := do(h, "GET", "/", nil, nil).Body.String()
	wantAll(t, body,
		// Contents links.
		`href="#cardgrid"`, `href="#timeline"`, `href="#splitpane"`, `href="#dashboardpage"`, `href="#settingspage"`, `href="#streampage"`, `href="#methodnotallowed"`, `href="#errorswap"`,
		// Card grid: a live grid of four cards, one card with a footer action that posts.
		`id="cardgrid"`, `id="showcase-cards"`, "xl:grid-cols-4", "lg:grid-cols-3", "Circlet of Borrowed Thought and Other Long Names", "Bottled Bolt", ">Remixable<", `href="/items/1/edit"`,
		// Timeline: two day groups, UTC, every tone.
		`id="timeline"`, `id="showcase-timeline"`, `datetime="2026-10-30"`, `datetime="2026-10-29"`, "Friday, October 30, 2026", "bg-success", "bg-info", "bg-muted-foreground", "bg-destructive", "bg-warning", "trace-demo-014",
		// The page links.
		`id="splitpane"`, `href="/browse"`, `href="/browse/3"`,
		`id="dashboardpage"`, `href="/dashboard"`,
		`id="settingspage"`, `href="/settings"`,
		`id="streampage"`, `href="/traffic"`,
		// The 405 form: a tiny form that POSTs to a GET-only address.
		`id="methodnotallowed"`, `id="post-get-only"`, `action="/traffic"`, `method="post"`, "POST to a GET-only address",
		// The error-swap guard: a button whose hx-get hits the plain 500, a target that must not change.
		`id="errorswap"`, `hx-get="/api/boom"`, `hx-target="#boom-result"`, `<span id="boom-result">Nothing here has changed.</span>`, "must not change",
	)
	// Every showcase card and every timeline event is there once.
	if n := strings.Count(body, "Circlet of Borrowed Thought and Other Long Names"); n != 1 {
		t.Errorf("long card title appears %d times, want 1", n)
	}
	// Counted inside the timeline section only: the log viewer's lines
	// carry <time> elements of their own.
	timeline := body[strings.Index(body, `id="showcase-timeline"`):strings.Index(body, `id="splitpane"`)]
	if n := strings.Count(timeline, `datetime="2026-10-`); n != 7 {
		t.Errorf("%d timeline datetimes (2 day headings + 5 events), want 7", n)
	}
	// The shell loads the guard script exactly once.
	if n := strings.Count(body, "/static/js/htmx-errors.js"); n != 1 {
		t.Errorf("htmx-errors.js loaded %d times, want 1", n)
	}
	// The old hand-written traffic page is gone: the topbar's Traffic entry
	// now opens the streampage route, which the tests above cover.
	wantAll(t, body, `href="/traffic"`)
}
