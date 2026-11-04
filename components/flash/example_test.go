package flash_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"

	"github.com/bplus/webframe/components/flash"
)

// A handler sets a flash before redirecting. On the next request Take reads
// the cookie, returns the banner to render and clears the cookie, so the
// message shows once. In an app, setting App.Flash = flash.Take does the
// second half on every page.
func ExampleTake() {
	// Request 1: save, set the flash, redirect.
	w := httptest.NewRecorder()
	flash.Success(w, httptest.NewRequest("POST", "/items", nil), "Saved", "Item 7 was updated.")
	cookie := w.Result().Cookies()[0]
	fmt.Println("set:", cookie.Name)

	// Request 2: the page the browser was sent to.
	next := httptest.NewRequest("GET", "/items/7", nil)
	next.AddCookie(cookie)
	w = httptest.NewRecorder()
	banner := flash.Take(w, next)

	var html strings.Builder
	_ = banner.Render(context.Background(), &html)
	fmt.Println("shows title:", strings.Contains(html.String(), "Saved"))
	fmt.Println("cleared:", w.Result().Cookies()[0].MaxAge < 0)
	// Output:
	// set: wf_flash
	// shows title: true
	// cleared: true
}
