package assets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	Mount(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandler_ServesOutputCSS(t *testing.T) {
	rec := get(t, "/static/css/output.css")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css prefix", ct)
	}
}

func TestHandler_ServesHTMX(t *testing.T) {
	rec := get(t, "/static/js/htmx.min.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandler_RejectsTraversal(t *testing.T) {
	// Hit the handler directly: ServeMux would redirect (307) a non-clean
	// path before the handler ever sees it.
	req := httptest.NewRequest(http.MethodGet, "/static/../embed.go", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 400 or 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "go:embed") {
		t.Error("response body leaked embed.go")
	}
}

func TestURL_CacheBusts(t *testing.T) {
	if got := URL("/static/js/x.js"); !strings.Contains(got, "?v=") {
		t.Errorf("URL = %q, want ?v= query", got)
	}
}
