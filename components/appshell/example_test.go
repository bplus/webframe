package appshell_test

import (
	"fmt"
	"net/http/httptest"

	"github.com/bplus/webframe/components/appshell"
)

// After a successful POST, Redirect answers a normal form submit with a
// 303 and an htmx submit with an empty 200 carrying HX-Redirect, so htmx
// loads the new page instead of swapping it into the form.
func ExampleRedirect() {
	plain := httptest.NewRequest("POST", "/items", nil)
	w := httptest.NewRecorder()
	appshell.Redirect(w, plain, "/items/7")
	fmt.Println(w.Code, w.Header().Get("Location"))

	htmx := httptest.NewRequest("POST", "/items", nil)
	htmx.Header.Set("HX-Request", "true")
	w = httptest.NewRecorder()
	appshell.Redirect(w, htmx, "/items/7")
	fmt.Println(w.Code, w.Header().Get("HX-Redirect"))
	// Output:
	// 303 /items/7
	// 200 /items/7
}
