package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHome(t *testing.T) {
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	for _, want := range []string{"<title>Home — Hello</title>", "Hello, webframe", "/static/css/output.css"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("page is missing %q", want)
		}
	}
}
