package demo

import (
	"net/http"

	"github.com/bplus/webframe/demo/pages"
)

// HandleFilterDemo answers the showcase's filter bar (GET
// /api/filter-demo?fq=...&school=...) with the content of its results
// region and, out of band, the School group again, so each school's count
// is that of the search. The data and the markup are in demo/pages
// (filterdemo.go, filterbar.templ).
func HandleFilterDemo(w http.ResponseWriter, r *http.Request) {
	filters, found, total := pages.FilterDemo(r.URL.Query())
	fragment(w, r, "filter-demo", pages.FilterDemoResponse(filters, found, total))
}
