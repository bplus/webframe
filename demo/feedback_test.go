package demo

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bplus/webframe/components/flash"
)

// fullHandler is the server exactly as cmd/demo serves it (Handler) on a
// fresh seeded store, with the slow endpoint's delay removed.
func fullHandler(t *testing.T) http.Handler {
	t.Helper()
	Items = NewItemStore(time.Now())
	slowDelay = 0
	t.Cleanup(func() { slowDelay = 1500 * time.Millisecond })
	return Handler()
}

// flashCookie returns the wf_flash cookie rec set, failing when there is
// not exactly one.
func flashCookie(t *testing.T, rec interface{ Result() *http.Response }) *http.Cookie {
	t.Helper()
	var found []*http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == flash.CookieName {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one %s cookie, got %v", flash.CookieName, found)
	}
	return found[0]
}

func TestFeedback_ToastSendIsHtmxOnly(t *testing.T) {
	h := fullHandler(t)
	for variant, want := range map[string][]string{
		"error":   {`role="alert"`, "Could not save", "border-destructive", "show = false, 4000)"},
		"warning": {`role="status"`, "Nothing was deleted", "border-warning"},
		"info":    {`role="status"`, "Showcase action", "border-info"},
		"success": {`role="status"`, "Export finished", "border-success"},
		"sticky":  {`role="alert"`, "Build failed", `aria-label="Dismiss"`},
		"bogus":   {"Showcase action"},
	} {
		rec := do(h, "POST", "/api/toast", url.Values{"variant": {variant}}, hx(""))
		if rec.Code != 200 {
			t.Errorf("%s: code = %d", variant, rec.Code)
		}
		body := rec.Body.String()
		wantAll(t, body, append(want, `hx-swap-oob="beforeend:#toast-container"`, "wf-toast")...)
		wantNone(t, body, "<html", "wf-topbar")
		if variant == "sticky" {
			wantNone(t, body, "x-init", "show = false, ")
		}
	}
	// A request that is not from htmx gets an empty 200: the toast has no
	// container to land in and Send writes nothing.
	rec := do(h, "POST", "/api/toast", url.Values{"variant": {"error"}}, nil)
	if rec.Code != 200 || rec.Body.Len() != 0 {
		t.Errorf("plain POST /api/toast = %d body %q, want empty 200", rec.Code, rec.Body.String())
	}
}

func TestFeedback_RefusedDeleteToast(t *testing.T) {
	h := fullHandler(t)
	// From the list, over htmx: 409 whose only body is the error toast.
	headers := hx("items-results")
	headers["HX-Current-URL"] = "http://localhost:3333/items"
	rec := do(h, "DELETE", "/items/1", nil, headers)
	if rec.Code != http.StatusConflict {
		t.Fatalf("DELETE /items/1 = %d, want 409", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		`hx-swap-oob="beforeend:#toast-container"`, `role="alert"`,
		"Could not delete Blade of the Fallen Star", "Starfall Pack",
	)
	wantNone(t, body, `id="items-results"`, "<html", "<h1")
	if strings.Count(body, "wf-toast") != 1 {
		t.Errorf("want exactly one toast, got %d in:\n%s", strings.Count(body, "wf-toast"), body)
	}
	if _, ok := Items.Get("1"); !ok {
		t.Error("item 1 was deleted")
	}
	if total, _, _ := Items.Counts(); total != 14 {
		t.Errorf("store changed on a refused delete: %d items", total)
	}
	// From the detail page, over htmx: the same 409 and toast, no redirect.
	rec = do(h, "DELETE", "/items/1", nil, hx(""))
	if rec.Code != http.StatusConflict || rec.Header().Get("HX-Redirect") != "" {
		t.Errorf("detail delete = %d HX-Redirect %q, want 409 and none", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	wantAll(t, rec.Body.String(), "Could not delete Blade of the Fallen Star")
	// A plain request has no page for a toast: an error flash and a 303
	// back to the item, where the sidebar shell shows it once.
	rec = do(h, "DELETE", "/items/1", nil, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/items/1" {
		t.Fatalf("plain delete = %d Location %q, want 303 /items/1", rec.Code, rec.Header().Get("Location"))
	}
	c := flashCookie(t, rec)
	rec = do(h, "GET", "/items/1", nil, map[string]string{"Cookie": c.String()})
	body = rec.Body.String()
	wantAll(t, body, "wf-shell--sidebar", `<div id="wf-flash">`, `role="alert"`, "Could not delete Blade of the Fallen Star")
	if cl := flashCookie(t, rec); cl.MaxAge >= 0 {
		t.Errorf("render did not clear the flash cookie: %q", cl.Raw)
	}
}

func TestFeedback_FlashShowsOnceOnNextPage(t *testing.T) {
	h := fullHandler(t)
	rec := do(h, "POST", "/api/flash", url.Values{}, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("POST /api/flash = %d Location %q, want 303 /", rec.Code, rec.Header().Get("Location"))
	}
	c := flashCookie(t, rec)
	if !c.HttpOnly || c.Path != "/" || c.MaxAge != 300 {
		t.Errorf("flash cookie attributes: %q", c.Raw)
	}
	// The next full page shows the warning banner in the slot above the
	// body, once, and clears the cookie.
	rec = do(h, "GET", "/", nil, map[string]string{"Cookie": c.String()})
	if rec.Code != 200 {
		t.Fatalf("GET / = %d", rec.Code)
	}
	body := rec.Body.String()
	inOrder(t, body, `<div id="wf-flash">`, `role="status"`, "This page was reached through a redirect", "Webframe Components")
	wantAll(t, body, "border-warning", `aria-label="Dismiss"`)
	if n := strings.Count(body, "This page was reached through a redirect"); n != 1 {
		t.Errorf("flash rendered %d times, want 1", n)
	}
	if cl := flashCookie(t, rec); cl.MaxAge >= 0 || cl.Path != "/" {
		t.Errorf("render did not clear the flash cookie: %q", cl.Raw)
	}
	// A reload (the browser dropped the cookie) shows the empty slot.
	body = do(h, "GET", "/", nil, nil).Body.String()
	wantAll(t, body, `<div id="wf-flash"></div>`)
	wantNone(t, body, "This page was reached through a redirect")
}

func TestFeedback_ErrorPages(t *testing.T) {
	h := fullHandler(t)
	for _, c := range []struct {
		path string
		code int
		want []string
	}{
		{"/errors/404", 404, []string{`>404<`, "Page not found", "Go home"}},
		{"/errors/403", 403, []string{`>403<`, "Not allowed", "You do not have access to this page."}},
		{"/errors/500", 500, []string{`>500<`, "Something went wrong", "Request id: " + demoRequestID}},
		{"/errors/418", 404, []string{`>404<`, "Page not found"}},
		// The mux's catch-all and an unknown item both go through errorpage.
		{"/no-such-page", 404, []string{`>404<`, "Page not found", "wf-topbar"}},
		{"/items/999", 404, []string{`>404<`, "Page not found", "wf-shell--sidebar"}},
		{"/items/999/edit", 404, []string{`>404<`, "Page not found", "wf-shell--sidebar"}},
	} {
		rec := do(h, "GET", c.path, nil, nil)
		if rec.Code != c.code {
			t.Errorf("GET %s = %d, want %d", c.path, rec.Code, c.code)
		}
		body := rec.Body.String()
		wantAll(t, body, append(c.want, "<!doctype", `href="/"`)...)
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s Content-Type = %q", c.path, ct)
		}
	}
	// Over htmx the error page is the body only, still with its status.
	rec := do(h, "GET", "/no-such-page", nil, hx("main"))
	if rec.Code != 404 {
		t.Errorf("htmx GET /no-such-page = %d", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Page not found")
	wantNone(t, rec.Body.String(), "<html", "wf-topbar")
	// The showcase still answers 200 and the home route is not shadowed by
	// the catch-all.
	if rec := do(h, "GET", "/", nil, nil); rec.Code != 200 {
		t.Errorf("GET / = %d", rec.Code)
	}
}

func TestFeedback_AuthPages(t *testing.T) {
	h := fullHandler(t)
	rec := do(h, "GET", "/signin", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /signin = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "Sign in — Webframe</title>", "wf-authpage", `<h1`, ">Webframe<",
		`<form`, `action="/signin"`, `method="post"`,
		`name="email"`, `type="email"`, `name="password"`, `type="password"`,
		`type="submit"`, "the demo has no accounts",
	)
	wantNone(t, body, "wf-topbar", "wf-sidebar", "wf-flash", `role="alert"`, "Demo: sign-in is not wired")

	// Every submission is refused: the error banner, the typed email kept,
	// the password never echoed, status 422 (htmx is configured to swap it).
	rec = do(h, "POST", "/signin", url.Values{"email": {"ada@example.com"}, "password": {"hunter2"}}, nil)
	if rec.Code != 422 {
		t.Fatalf("POST /signin = %d", rec.Code)
	}
	body = rec.Body.String()
	wantAll(t, body, `role="alert"`, "Demo: sign-in is not wired", `value="ada@example.com"`)
	wantNone(t, body, "hunter2", "wf-topbar")
	if rec.Header().Get("Location") != "" || rec.Header().Get("HX-Redirect") != "" {
		t.Error("refused sign-in must not redirect")
	}
	// An htmx submit gets the bare card.
	body = do(h, "POST", "/signin", url.Values{"email": {"x@y.z"}}, hx("signin")).Body.String()
	wantAll(t, body, "wf-authpage", "Demo: sign-in is not wired")
	wantNone(t, body, "<html")

	rec = do(h, "GET", "/signed-out", nil, nil)
	if rec.Code != 200 {
		t.Errorf("GET /signed-out = %d", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Signed out — Webframe</title>", "You have been signed out.", "Sign in again", `href="/signin"`)
	wantNone(t, rec.Body.String(), "wf-topbar", "<form")

	rec = do(h, "GET", "/forbidden", nil, nil)
	if rec.Code != 200 {
		t.Errorf("GET /forbidden = %d", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Not allowed — Webframe</title>", "Your account does not have access.", "Go home", `href="/"`)
	wantNone(t, rec.Body.String(), "wf-topbar", "<form")
}

func TestFeedback_SlowFragment(t *testing.T) {
	h := fullHandler(t)
	rec := do(h, "GET", "/api/slow", nil, hx("slow-result"))
	if rec.Code != 200 {
		t.Fatalf("GET /api/slow = %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.HasPrefix(body, "<small>Answered at ") || !strings.HasSuffix(body, "</small>") {
		t.Errorf("slow fragment = %q", body)
	}
}

// TestFeedback_ShowcaseSections pins each new section's id and the markup
// its demo relies on, and that the demo's own toast fragment is gone.
func TestFeedback_ShowcaseSections(t *testing.T) {
	h := fullHandler(t)
	body := do(h, "GET", "/", nil, nil).Body.String()
	wantAll(t, body,
		`id="feedback"`, `href="/#feedback"`,
		`id="toast"`, `hx-post="/api/toast"`, `{&#34;variant&#34;:&#34;sticky&#34;}`,
		`id="flash"`, `id="flash-form"`, `action="/api/flash"`, `method="post"`,
		`id="confirmdialog"`, `hx-confirm="Deleting is permanent; the item leaves every pack that uses it."`,
		`data-wf-confirm-title="Delete this item?"`, `data-wf-confirm-label="Delete"`, `data-wf-confirm-cancel="Keep it"`,
		`data-wf-confirm-destructive="true"`, `hx-confirm="Archive this item? You can restore it later."`,
		`id="skeleton"`, `aria-busy="true"`, `id="skeleton-table"`,
		`id="spinner"`, "htmx-indicator", `hx-get="/api/slow"`, `id="slow-result"`, "Saving…",
		`id="bottomsheet"`, `data-slideover-instance="sheet-demo"`, "Fire a toast", "data-tui-slideover-close",
		`id="errorpage"`, `href="/errors/404"`, `href="/errors/403"`, `href="/errors/500"`,
		`id="authpage"`, `href="/signin"`, `href="/signed-out"`, `href="/forbidden"`,
		`id="stickyheader"`, `id="sticky-capped"`, `id="sticky-page"`, "Example item 30", "sticky top-0",
		`id="phonesort"`, `href="/items"`,
		// The one confirm host and the toast container the shell renders.
		`id="wf-confirm"`, `id="toast-container"`,
	)
	wantNone(t, body, "Toast (hx-swap-oob)", "Operation completed successfully")
	if n := strings.Count(body, `id="wf-confirm"`); n != 1 {
		t.Errorf("confirm host appears %d times, want 1", n)
	}
	// The status chips in the table and description list still render
	// through the badge helpers.
	wantAll(t, body, "Weird", ">Failed<", ">Published<")
}
