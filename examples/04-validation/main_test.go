package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func post(t *testing.T, title string, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/books", strings.NewReader(url.Values{"title": {title}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if htmx { // what htmx sends when the form submits itself
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "form")
	}
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, req)
	return rec
}

func TestNewBookFormShows(t *testing.T) {
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, httptest.NewRequest("GET", "/books/new", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /books/new = %d, want 200", rec.Code)
	}
	for _, want := range []string{`action="/books"`, `name="title"`, "New book"} {
		if !strings.Contains(body, want) {
			t.Errorf("form page is missing %q", want)
		}
	}
}

func TestListLinksToNewBook(t *testing.T) {
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, httptest.NewRequest("GET", "/books", nil))
	if !strings.Contains(rec.Body.String(), `href="/books/new"`) {
		t.Error("list page has no link to /books/new")
	}
}

func TestCreateRedirectsAndFlashes(t *testing.T) {
	rec := post(t, "  A Christmas Carol ", false)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/books" {
		t.Fatalf("POST /books = %d to %q, want 303 to /books", rec.Code, rec.Header().Get("Location"))
	}

	// Follow the redirect the way a browser does: GET /books with the cookie.
	req := httptest.NewRequest("GET", "/books", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	next := httptest.NewRecorder()
	routes().ServeHTTP(next, req)
	body := next.Body.String()
	for _, want := range []string{"Saved", "A Christmas Carol was added.", "<td", "A Christmas Carol</"} {
		if !strings.Contains(body, want) {
			t.Errorf("list after saving is missing %q", want)
		}
	}

	// The flash shows once: a GET without the cookie has no banner.
	again := httptest.NewRecorder()
	routes().ServeHTTP(again, httptest.NewRequest("GET", "/books", nil))
	if strings.Contains(again.Body.String(), "was added.") {
		t.Error("the Saved banner showed a second time")
	}
}

func TestHtmxCreateUsesHXRedirect(t *testing.T) {
	rec := post(t, "Emma", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Redirect") != "/books" {
		t.Fatalf("htmx POST /books = %d, HX-Redirect %q; want 200 and /books", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if rec.Header().Get("Location") != "" {
		t.Error("htmx POST also sent a Location header")
	}
}

func TestEmptyTitleIsRefused(t *testing.T) {
	booksLock.Lock()
	before := len(books)
	booksLock.Unlock()

	for _, htmx := range []bool{false, true} {
		rec := post(t, "   ", htmx)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("POST blank title (htmx %v) = %d, want 422", htmx, rec.Code)
		}
		if rec.Header().Get("Location") != "" || rec.Header().Get("HX-Redirect") != "" {
			t.Errorf("refused POST (htmx %v) still redirects", htmx)
		}
		body := rec.Body.String()
		for _, want := range []string{"Enter a title.", `aria-invalid="true"`, `value="   "`} {
			if !strings.Contains(body, want) {
				t.Errorf("refused form (htmx %v) is missing %q", htmx, want)
			}
		}
		if htmx && !strings.HasPrefix(strings.TrimSpace(body), "<form") {
			t.Error("htmx re-render should be the form alone")
		}
	}

	booksLock.Lock()
	after := len(books)
	booksLock.Unlock()
	if after != before {
		t.Errorf("a refused POST saved a book: %d books before, %d after", before, after)
	}
}
