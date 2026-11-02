package pagination

import (
	"strconv"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/listquery"
)

// Props configures Pagination. Class is merged via utils.TwMerge; Attributes
// spread onto the root <nav> last.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes
	// Total is the number of rows across every page (Page.Total from
	// listquery.Apply, or the SQL count).
	Total int
	// Spec supplies the current Page and PageSize and every other list
	// parameter (sort, search, filters), all of which carry into the page
	// links. Pass the Spec listquery.Apply returned, whose Page has been
	// clamped to the last page; a zero-value Spec means page 1 of 25.
	Spec listquery.Spec
	// Base is the link base ("/items"). "" uses the request path stored by
	// appshell.Middleware (appshell.PathFromContext); when that is empty too,
	// links are query-only ("?page=3") and resolve against the current page.
	Base string
	// HxTarget, when set, makes every link also carry hx-get (same URL),
	// hx-target={HxTarget} and hx-push-url="true" so htmx swaps only the
	// results region and still updates the address bar.
	HxTarget string
	// Noun names the rows in the summary: "items" → "1–25 of 140 items";
	// "" → "1–25 of 140".
	Noun string
}

// maxWindow is the most entries window returns, ellipses included.
const maxWindow = 7

// window returns the page numbers to show for the current page out of pages,
// with 0 standing for an ellipsis, never more than seven entries:
//
//	pages <= 7         → every page
//	page <= 4          → 1 2 3 4 5 … pages
//	page >= pages-3    → 1 … pages-4 pages-3 pages-2 pages-1 pages
//	otherwise          → 1 … page-1 page page+1 … pages
//
// Out-of-range inputs clamp: pages < 1 is one page, page is clamped into
// 1..pages.
func window(page, pages int) []int {
	if pages < 1 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	if pages <= maxWindow {
		out := make([]int, 0, pages)
		for i := 1; i <= pages; i++ {
			out = append(out, i)
		}
		return out
	}
	switch {
	case page <= 4:
		return []int{1, 2, 3, 4, 5, 0, pages}
	case page >= pages-3:
		return []int{1, 0, pages - 4, pages - 3, pages - 2, pages - 1, pages}
	default:
		return []int{1, 0, page - 1, page, page + 1, 0, pages}
	}
}

// state is the resolved paging arithmetic one render works from.
type state struct {
	page, pages, size int
}

func (p Props) state() state {
	size := p.Spec.Limit()
	pages := p.Spec.Pages(p.Total)
	page := p.Spec.Page
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	return state{page: page, pages: pages, size: size}
}

// summary is the always-rendered range line: "1–25 of 140 items", or
// "0 items" when there are no rows. The dash is an en dash.
func (p Props) summary(s state) string {
	noun := ""
	if p.Noun != "" {
		noun = " " + p.Noun
	}
	if p.Total < 1 {
		return "0" + noun
	}
	start := (s.page-1)*s.size + 1
	end := s.page * s.size
	if end > p.Total {
		end = p.Total
	}
	return strconv.Itoa(start) + "–" + strconv.Itoa(end) + " of " + strconv.Itoa(p.Total) + noun
}

// href is the link for page n from base, built by listquery so every other
// list parameter (sort, search, filters, non-default page size) rides along
// and values are query-escaped.
func (p Props) href(base string, n int) string {
	return p.Spec.URL(base, listquery.WithPage(n))
}
