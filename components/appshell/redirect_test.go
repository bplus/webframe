package appshell

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectPlainRequestIs303(t *testing.T) {
	rec := httptest.NewRecorder()
	Redirect(rec, httptest.NewRequest("POST", "/books", nil), "/books")
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/books" {
		t.Errorf("Location = %q, want /books", got)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "" {
		t.Errorf("plain request got HX-Redirect %q", got)
	}
}

func TestRedirectHtmxRequestIsHXRedirect(t *testing.T) {
	req := httptest.NewRequest("POST", "/books", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	Redirect(rec, req, "/books/7")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("HX-Redirect"); got != "/books/7" {
		t.Errorf("HX-Redirect = %q, want /books/7", got)
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Errorf("htmx request also got Location %q", got)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}
