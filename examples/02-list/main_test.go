package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, url string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", url, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, req)
	return rec
}

// inOrder fails unless every string appears in body, each after the one before.
func inOrder(t *testing.T, body string, want ...string) {
	t.Helper()
	at := 0
	for _, s := range want {
		i := strings.Index(body[at:], s)
		if i < 0 {
			t.Fatalf("%q missing or out of order (want order %q)", s, want)
		}
		at += i + len(s)
	}
}

func TestHomeRedirectsToList(t *testing.T) {
	rec := get(t, "/", nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/books" {
		t.Fatalf("GET / = %d to %q, want 303 to /books", rec.Code, rec.Header().Get("Location"))
	}
}

func TestListDefaultsToTitleOrderFirstPage(t *testing.T) {
	rec := get(t, "/books", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /books = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	inOrder(t, body, "<title>Books — Books</title>", "Dracula", "Frankenstein", "Jane Eyre", "Little Women")
	if strings.Contains(body, "Middlemarch") {
		t.Error("page 1 shows Middlemarch, which belongs on page 2")
	}
	if !strings.Contains(body, "of 9 books") {
		t.Error("pagination summary missing \"of 9 books\"")
	}
}

func TestSortAndPageFromQueryString(t *testing.T) {
	body := get(t, "/books?sort=year&dir=desc", nil).Body.String()
	inOrder(t, body, "Dracula", "The Time Machine", "Treasure Island", "Middlemarch")

	body = get(t, "/books?page=3", nil).Body.String()
	if !strings.Contains(body, "Treasure Island") || strings.Contains(body, "Dracula") {
		t.Error("page 3 should hold only Treasure Island")
	}
}

func TestHtmxSortGetsOnlyTheResults(t *testing.T) {
	rec := get(t, "/books?sort=author", map[string]string{"HX-Request": "true", "HX-Target": "list-results"})
	body := rec.Body.String()
	if strings.Contains(body, "<html") || strings.Contains(body, "<h1") {
		t.Error("htmx results request got the whole page")
	}
	inOrder(t, body, `id="list-results"`, "Bram Stoker", "Charlotte Brontë")
}
