package flash

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
)

// TestShell_FlashShowsOnceOnNextFullPage drives the real wiring: a handler
// sets the flash and redirects; the next full render shows the banner inside
// #wf-flash and clears the cookie; a reload shows nothing; an htmx body-only
// render never consumes it.
func TestShell_FlashShowsOnceOnNextFullPage(t *testing.T) {
	app := &appshell.App{Name: "Movies", HomeHref: "/"}
	app.Flash = Take
	body := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<p>page body</p>")
		return err
	})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /movies/7/delete", func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, "Could not delete", "Still in two lists.")
		http.Redirect(w, r, "/movies", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /movies", func(w http.ResponseWriter, r *http.Request) {
		if err := app.Render(w, r, "Movies", body); err != nil {
			t.Errorf("render: %v", err)
		}
	})
	h := app.Middleware(mux)

	// 1. The action sets the cookie and redirects.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/movies/7/delete", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("redirect status = %d", rec.Code)
	}
	set := rec.Result().Cookies()
	if len(set) != 1 || set[0].Name != CookieName || set[0].MaxAge != 300 {
		t.Fatalf("want one wf_flash cookie with MaxAge 300, got %v", set)
	}

	// 2. An htmx body-only render with the cookie leaves it alone.
	rec = httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/movies", nil)
	r.Header.Set("HX-Request", "true")
	r.AddCookie(set[0])
	h.ServeHTTP(rec, r)
	if out := rec.Body.String(); strings.Contains(out, "Could not delete") || strings.Contains(out, "wf-flash") {
		t.Errorf("body-only render consumed the flash:\n%s", out)
	}
	if sc := rec.Result().Header.Values("Set-Cookie"); len(sc) != 0 {
		t.Errorf("body-only render touched the cookie: %v", sc)
	}

	// 3. The next full page shows the banner in #wf-flash and clears it.
	rec = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/movies", nil)
	r.AddCookie(set[0])
	h.ServeHTTP(rec, r)
	out := rec.Body.String()
	slot := strings.Index(out, `<div id="wf-flash">`)
	banner := strings.Index(out, `role="alert"`)
	pageBody := strings.Index(out, "<p>page body</p>")
	if slot < 0 || banner < slot || pageBody < banner {
		t.Errorf("banner not inside #wf-flash above the body (slot=%d banner=%d body=%d):\n%s", slot, banner, pageBody, out)
	}
	if strings.Count(out, "Could not delete") != 1 {
		t.Errorf("flash rendered %d times, want 1", strings.Count(out, "Could not delete"))
	}
	cleared := rec.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != CookieName || cleared[0].MaxAge >= 0 || cleared[0].Path != "/" {
		t.Errorf("full render did not clear the cookie: %v", cleared)
	}

	// 4. A reload (the browser dropped the cookie) shows the empty slot.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/movies", nil))
	out = rec.Body.String()
	if !strings.Contains(out, `<div id="wf-flash"></div>`) || strings.Contains(out, "Could not delete") {
		t.Errorf("reload did not render an empty slot:\n%s", out)
	}
}
