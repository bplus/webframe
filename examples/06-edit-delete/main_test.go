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

func TestShortTitleIsRefusedAndKept(t *testing.T) {
	booksLock.Lock()
	before := len(books)
	booksLock.Unlock()

	for _, htmx := range []bool{false, true} {
		rec := post(t, "Hi", htmx)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("POST short title (htmx %v) = %d, want 422", htmx, rec.Code)
		}
		if rec.Header().Get("Location") != "" || rec.Header().Get("HX-Redirect") != "" {
			t.Errorf("refused POST (htmx %v) still redirects", htmx)
		}
		body := rec.Body.String()
		for _, want := range []string{"Enter at least 3 characters.", `aria-invalid="true"`, `value="Hi"`} {
			if !strings.Contains(body, want) {
				t.Errorf("refused form (htmx %v) is missing %q", htmx, want)
			}
		}
	}

	booksLock.Lock()
	after := len(books)
	booksLock.Unlock()
	if after != before {
		t.Errorf("a refused POST saved a book: %d books before, %d after", before, after)
	}
}

func TestThreeCharacterTitleIsSaved(t *testing.T) {
	// Three characters after trimming is enough, counted as characters:
	// "Ève" is four bytes.
	rec := post(t, "  Ève ", false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST three-character title = %d, want 303", rec.Code)
	}
}

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestDetailPageShowsTheBook(t *testing.T) {
	rec := get(t, "/books/2")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /books/2 = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Frankenstein", "Mary Shelley", "1818", "Author", "Year", `href="/books"`} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page is missing %q", want)
		}
	}
}

func TestUnknownBookIsStyled404(t *testing.T) {
	rec := get(t, "/books/999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /books/999 = %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"<html", "Page not found", "Go home"} {
		if !strings.Contains(body, want) {
			t.Errorf("the 404 page is missing %q", want)
		}
	}
}

func TestListLinksToDetail(t *testing.T) {
	body := get(t, "/books").Body.String()
	// Sorted by title, page 1 starts with Dracula (id 9).
	if !strings.Contains(body, `href="/books/9"`) {
		t.Error("list page has no link to /books/9")
	}
}

func TestNewBookGetsItsOwnPage(t *testing.T) {
	rec := post(t, "Northanger Abbey", false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /books = %d, want 303", rec.Code)
	}
	var id string
	booksLock.Lock()
	for _, book := range books {
		if book.Title == "Northanger Abbey" {
			id = book.ID
		}
	}
	booksLock.Unlock()
	if id == "" {
		t.Fatal("the new book was not saved")
	}
	if page := get(t, "/books/"+id); page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Northanger Abbey") {
		t.Errorf("GET /books/%s = %d, want 200 with the title", id, page.Code)
	}
}

// send makes a request the way the browser would: a form body for POST, the
// htmx headers when htmx sends it, and the cookies from an earlier answer.
func send(t *testing.T, method, path, title string, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if method == "POST" {
		req = httptest.NewRequest(method, path, strings.NewReader(url.Values{"title": {title}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	routes().ServeHTTP(rec, req)
	return rec
}

// follow loads path with the cookies rec set, as a browser does after a redirect.
func follow(t *testing.T, rec *httptest.ResponseRecorder, path string) string {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	next := httptest.NewRecorder()
	routes().ServeHTTP(next, req)
	return next.Body.String()
}

func bookTitle(id string) string {
	book, _ := findBook(id)
	return book.Title
}

func TestDetailPageHasEditAndDelete(t *testing.T) {
	body := get(t, "/books/3").Body.String()
	for _, want := range []string{`href="/books/3/edit"`, `hx-delete="/books/3"`, `hx-confirm="This cannot be undone."`, `data-wf-confirm-title="Delete Jane Eyre?"`, "data-wf-confirm-destructive"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page is missing %q", want)
		}
	}
}

func TestEditFormIsPrefilled(t *testing.T) {
	rec := get(t, "/books/4/edit")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /books/4/edit = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Edit book", `action="/books/4"`, `value="Moby-Dick"`, `href="/books/4"`} {
		if !strings.Contains(body, want) {
			t.Errorf("edit form is missing %q", want)
		}
	}
	if get(t, "/books/999/edit").Code != http.StatusNotFound {
		t.Error("the edit form of an unknown book should be 404")
	}
}

func TestEditSavesAndRedirects(t *testing.T) {
	rec := send(t, "POST", "/books/5", " Little Women, Part One ", false)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/books/5" {
		t.Fatalf("POST /books/5 = %d to %q, want 303 to /books/5", rec.Code, rec.Header().Get("Location"))
	}
	if got := bookTitle("5"); got != "Little Women, Part One" {
		t.Errorf("title after edit = %q", got)
	}
	body := follow(t, rec, "/books/5")
	for _, want := range []string{"Saved", "Little Women, Part One was updated.", "Louisa May Alcott"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page after saving is missing %q", want)
		}
	}

	htmx := send(t, "POST", "/books/5", "Little Women", true)
	if htmx.Code != http.StatusOK || htmx.Header().Get("HX-Redirect") != "/books/5" {
		t.Errorf("htmx POST /books/5 = %d, HX-Redirect %q; want 200 and /books/5", htmx.Code, htmx.Header().Get("HX-Redirect"))
	}
}

func TestEditStillRefusesShortTitle(t *testing.T) {
	for _, htmx := range []bool{false, true} {
		rec := send(t, "POST", "/books/6", "Hi", htmx)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("POST short title on edit (htmx %v) = %d, want 422", htmx, rec.Code)
		}
		body := rec.Body.String()
		for _, want := range []string{"Enter at least 3 characters.", `value="Hi"`, `action="/books/6"`} {
			if !strings.Contains(body, want) {
				t.Errorf("refused edit (htmx %v) is missing %q", htmx, want)
			}
		}
	}
	if got := bookTitle("6"); got != "Middlemarch" {
		t.Errorf("a refused edit changed the title to %q", got)
	}
	if send(t, "POST", "/books/999", "Anything", false).Code != http.StatusNotFound {
		t.Error("saving an unknown book should be 404")
	}
}

func TestDeleteRemovesAndFlashes(t *testing.T) {
	// htmx sends the delete from the detail page and follows HX-Redirect.
	rec := send(t, "DELETE", "/books/7", "", true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Redirect") != "/books" {
		t.Fatalf("htmx DELETE /books/7 = %d, HX-Redirect %q; want 200 and /books", rec.Code, rec.Header().Get("HX-Redirect"))
	}
	if _, found := findBook("7"); found {
		t.Error("book 7 is still there")
	}
	body := follow(t, rec, "/books")
	for _, want := range []string{"Deleted", "Treasure Island was deleted."} {
		if !strings.Contains(body, want) {
			t.Errorf("list after deleting is missing %q", want)
		}
	}
	if get(t, "/books/7").Code != http.StatusNotFound {
		t.Error("the deleted book's page should be 404")
	}

	// Without htmx (curl, say) the answer is a plain 303.
	plain := send(t, "DELETE", "/books/8", "", false)
	if plain.Code != http.StatusSeeOther || plain.Header().Get("Location") != "/books" {
		t.Errorf("DELETE /books/8 = %d to %q, want 303 to /books", plain.Code, plain.Header().Get("Location"))
	}

	// A second delete of the same book (from another tab) is not an error:
	// back to the list with a notice.
	again := send(t, "DELETE", "/books/8", "", true)
	if again.Header().Get("HX-Redirect") != "/books" {
		t.Fatalf("second DELETE /books/8: HX-Redirect %q, want /books", again.Header().Get("HX-Redirect"))
	}
	if !strings.Contains(follow(t, again, "/books"), "Already deleted") {
		t.Error("second delete did not say the book was already gone")
	}
}
