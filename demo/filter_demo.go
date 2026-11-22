package demo

import (
	"net/http"

	"github.com/bplus/webframe/demo/pages"
)

// HandleFilterDemo answers the showcase's filter bar (GET
// /api/filter-demo?fq=...&school=...) with the content of its results
// region. The data and the markup are in demo/pages (filterdemo.go,
// filterbar.templ).
func HandleFilterDemo(w http.ResponseWriter, r *http.Request) {
	_, found, total := pages.FilterDemo(r.URL.Query())
	fragment(w, r, "filter-demo", pages.FilterDemoResults(found, total))
}
