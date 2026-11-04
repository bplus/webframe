package formpage_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/formpage"
)

// One declaration serves both the empty form and a failed submit. On a
// failed POST the handler puts the submitted values and the errors back in
// and calls Respond again, which answers 422 with the form alone when htmx
// sent the request.
func Example() {
	app := &appshell.App{Name: "Demo"}

	newItem := func(w http.ResponseWriter, r *http.Request) {
		p := formpage.Props{
			Action: "/items",
			HxPost: true,
			Sections: []formpage.Section{{
				Fields: []formfield.Field{{Name: "name", Label: "Name", Required: true}},
			}},
		}
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			if r.FormValue("name") != "" {
				// Save the item here, then:
				appshell.Redirect(w, r, "/items")
				return
			}
			p.Values = map[string]string{"name": r.FormValue("name")}
			p.Errors = map[string]string{"name": "Name is required."}
		}
		_ = formpage.Respond(w, r, "New item", p)
	}
	handler := app.Middleware(http.HandlerFunc(newItem))

	r := httptest.NewRequest("POST", "/items", strings.NewReader(url.Values{"name": {""}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", "form") // what an HxPost submit sends
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	body := w.Body.String()
	fmt.Println(w.Code)
	fmt.Println("form only:", strings.HasPrefix(strings.TrimSpace(body), "<form"))
	fmt.Println("shows error:", strings.Contains(body, "Name is required."))
	// Output:
	// 422
	// form only: true
	// shows error: true
}
