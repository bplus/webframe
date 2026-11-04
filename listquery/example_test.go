package listquery_test

import (
	"fmt"
	"net/http/httptest"

	"github.com/bplus/webframe/listquery"
)

// A handler reads the list state from the request, then builds the links
// for the next page and for a sort header from it.
func ExampleParse() {
	opts := listquery.Options{
		DefaultSort:  "name",
		AllowedSorts: []string{"name", "created"},
		FilterKeys:   []string{"status"},
	}
	r := httptest.NewRequest("GET", "/items?sort=created&dir=desc&page=2&status=open", nil)

	spec := listquery.Parse(r, opts)
	fmt.Println(spec.Sort, spec.Dir, spec.Page, spec.Filters["status"])
	fmt.Println("SQL:", spec.Offset(), spec.Limit())

	fmt.Println(spec.URL("/items", listquery.WithPage(3)))
	// Sorting by the current key flips the direction.
	fmt.Println(spec.URL("/items", listquery.WithSort("created")))
	// The default order (name, ascending) is left out of the URL.
	fmt.Println(spec.URL("/items", listquery.WithSort("name")))
	// Output:
	// created desc 2 [open]
	// SQL: 25 25
	// /items?dir=desc&page=3&sort=created&status=open
	// /items?dir=asc&sort=created&status=open
	// /items?status=open
}
