// Package listquery is the query-string contract shared by sortable table
// headers, filter bars and pagination: one list's state (sort, direction,
// page, page size, search, filters) read from a URL and written back into
// derived URLs. It renders nothing and knows no database; handlers translate
// a Spec into their own query (Offset/Limit for SQL callers, Apply for
// in-memory slices).
//
// Parameter names are fixed: sort, dir, page, per, q, plus filter keys as
// named. Options.Prefix namespaces every one of them ("items" → items.sort,
// items.page …) so two independent lists can share a page; a prefixed and an
// unprefixed Spec never read each other's parameters.
//
// Parsing is hostile-input safe: unknown sorts fall back to DefaultSort,
// unknown directions to DefaultDir, non-numeric or non-positive pages to 1,
// pages above MaxPage to MaxPage, and page sizes clamp to 1..MaxPageSize.
// Nothing panics on any input.
package listquery

import (
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Dir is a sort direction.
type Dir string

const (
	Asc  Dir = "asc"
	Desc Dir = "desc"
)

const (
	defaultPageSize = 25
	defaultMaxSize  = 100
)

// MaxPage is the ceiling on Spec.Page: ParseValues caps a larger page
// parameter here, and Offset caps Page and PageSize here again before
// multiplying, so a hand-typed ?page=4611686018427387904&per=100 yields a
// large offset (no rows) instead of overflowing into a negative one.
// (MaxPage-1)*MaxPage fits in an int64; on a 32-bit int the offset tops out
// at math.MaxInt.
const MaxPage = math.MaxInt32

// Spec is one list's state as read from the URL. The zero value means page 1,
// the default page size (25), no sort, no search, no filters.
type Spec struct {
	Sort     string
	Dir      Dir
	Page     int // 1-based
	PageSize int
	Search   string              // the q parameter
	Filters  map[string][]string // only keys listed in Options.FilterKeys; empty values dropped
	prefix   string              // from Options.Prefix; copied into every derived Spec
	// defaultSize is the effective default page size ParseValues resolved
	// (Options.DefaultPageSize, 0 → 25, clamped to MaxPageSize). URL omits
	// per when PageSize equals it. Zero on a hand-built Spec: per is then
	// written whenever PageSize > 0.
	defaultSize int
}

// Options controls Parse. Zero values mean: no default sort, Asc, page size 25,
// max page size 100, only DefaultSort allowed, no filters read, no prefix.
type Options struct {
	DefaultSort     string
	DefaultDir      Dir      // "" → Asc
	DefaultPageSize int      // 0 → 25
	MaxPageSize     int      // 0 → 100
	AllowedSorts    []string // unknown sort → DefaultSort (DefaultSort is always allowed)
	// FilterKeys lists the filter parameters read into Filters; any other
	// key is ignored. Filter values are comma-free tokens: a value may
	// arrive repeated (status=a&status=b, the form URL emits) or
	// comma-joined (status=a,b, the form filterbar.min.js submits), and
	// both parse to ["a","b"]. Each token is trimmed of surrounding
	// spaces; empty tokens are dropped.
	FilterKeys []string
	Prefix     string // "" → bare names; "items" → items.sort, items.page …
}

// Parse reads the list state from r's query string. See ParseValues.
func Parse(r *http.Request, o Options) Spec {
	if r == nil || r.URL == nil {
		return ParseValues(nil, o)
	}
	return ParseValues(r.URL.Query(), o)
}

// CurrentURL returns the URL the user is looking at: the HX-Current-URL
// header htmx sends with every request, parsed, or a copy of r.URL when the
// header is absent or does not parse. It is never nil (a nil request yields
// an empty URL). A mutation issued from inside a list (a row's DELETE, a
// bulk POST) has a request URL that says nothing about the list's state;
// the page's URL holds it.
func CurrentURL(r *http.Request) *url.URL {
	if r == nil {
		return &url.URL{}
	}
	if cur := r.Header.Get("HX-Current-URL"); cur != "" {
		if u, err := url.Parse(cur); err == nil {
			return u
		}
	}
	if r.URL == nil {
		return &url.URL{}
	}
	u := *r.URL
	return &u
}

// ParseCurrent reads the list state from CurrentURL(r)'s query: the
// HX-Current-URL header's query when htmx sent one, else r's own query, with
// the same validation as ParseValues. Handlers that mutate from inside a
// list re-render it for this Spec.
func ParseCurrent(r *http.Request, o Options) Spec {
	return ParseValues(CurrentURL(r).Query(), o)
}

// ParseValues reads the list state from v. Every value is validated: 1 <=
// Page <= MaxPage, 1 <= PageSize <= MaxPageSize, Dir ∈ {asc, desc}, Sort ∈
// AllowedSorts ∪ {DefaultSort}, Filters ⊆ FilterKeys. Repeated scalar
// parameters take the first value; Search is kept verbatim as text (escaping
// is the renderer's job).
func ParseValues(v url.Values, o Options) Spec {
	s := Spec{prefix: o.Prefix}
	if v == nil {
		v = url.Values{}
	}

	// Sort.
	s.Sort = o.DefaultSort
	if raw := v.Get(s.Param("sort")); raw != "" && sortAllowed(raw, o) {
		s.Sort = raw
	}

	// Dir. DefaultDir belongs to DefaultSort: a URL that names a different
	// sort without a dir sorts ascending, the same as a fresh header click
	// (WithSort), so a hand-typed ?sort=name never inherits a Desc default
	// meant for "updated".
	s.Dir = o.DefaultDir
	if s.Dir != Asc && s.Dir != Desc {
		s.Dir = Asc
	}
	if s.Sort != o.DefaultSort {
		s.Dir = Asc
	}
	switch Dir(strings.ToLower(v.Get(s.Param("dir")))) {
	case Asc:
		s.Dir = Asc
	case Desc:
		s.Dir = Desc
	}

	// Page size, clamped to 1..max.
	maxSize := o.MaxPageSize
	if maxSize < 1 {
		maxSize = defaultMaxSize
	}
	size := o.DefaultPageSize
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxSize {
		size = maxSize
	}
	s.defaultSize = size
	if n, ok := positiveInt(v.Get(s.Param("per"))); ok {
		size = n
	}
	if size > maxSize {
		size = maxSize
	}
	s.PageSize = size

	// Page, capped at MaxPage.
	s.Page = 1
	if n, ok := positiveInt(v.Get(s.Param("page"))); ok {
		s.Page = min(n, MaxPage)
	}

	// Search, verbatim.
	s.Search = v.Get(s.Param("q"))

	// Filters: listed keys only; each value split on "," into trimmed
	// tokens, empty tokens dropped.
	for _, key := range o.FilterKeys {
		if key == "" {
			continue
		}
		var vals []string
		for _, raw := range v[s.Param(key)] {
			for _, val := range strings.Split(raw, ",") {
				if val = strings.TrimSpace(val); val != "" {
					vals = append(vals, val)
				}
			}
		}
		if len(vals) > 0 {
			if s.Filters == nil {
				s.Filters = make(map[string][]string, len(o.FilterKeys))
			}
			s.Filters[key] = vals
		}
	}
	return s
}

func sortAllowed(key string, o Options) bool {
	if key == o.DefaultSort {
		return true
	}
	for _, k := range o.AllowedSorts {
		if k == key {
			return true
		}
	}
	return false
}

// positiveInt parses raw as an int >= 1. Anything else (empty, non-numeric,
// zero, negative, overflow) reports false.
func positiveInt(raw string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

// Param returns the query parameter name for name under this Spec's prefix:
// "sort" → "sort", or "items.sort" when the prefix is "items".
func (s Spec) Param(name string) string {
	if s.prefix == "" {
		return name
	}
	return s.prefix + "." + name
}

// Prefix returns the namespace this Spec was parsed with ("" when none).
func (s Spec) Prefix() string { return s.prefix }

// page is Page with the zero value normalised to 1.
func (s Spec) page() int {
	if s.Page < 1 {
		return 1
	}
	return s.Page
}

// pageSize is PageSize with the zero value normalised to the default.
func (s Spec) pageSize() int {
	if s.PageSize < 1 {
		return defaultPageSize
	}
	return s.PageSize
}

// Offset returns (Page-1)*PageSize for SQL callers. Page and PageSize are
// each capped at MaxPage first and the product is computed in 64 bits, so a
// hand-built Spec with absurd values yields a large offset, never a wrapped
// negative one.
func (s Spec) Offset() int {
	page := int64(min(s.page(), MaxPage))
	size := int64(min(s.pageSize(), MaxPage))
	off := (page - 1) * size
	if off > math.MaxInt {
		off = math.MaxInt
	}
	return int(off)
}

// Limit returns PageSize for SQL callers (25 for the zero value).
func (s Spec) Limit() int { return s.pageSize() }

// Pages returns the number of pages needed for total items, at least 1.
func (s Spec) Pages(total int) int {
	if total < 1 {
		return 1
	}
	size := s.pageSize()
	return (total + size - 1) / size
}

// Filter returns the first value for key, or "".
func (s Spec) Filter(key string) string {
	if vals := s.Filters[key]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// HasFilters reports whether a search or any filter is active.
func (s Spec) HasFilters() bool {
	if s.Search != "" {
		return true
	}
	for _, vals := range s.Filters {
		if len(vals) > 0 {
			return true
		}
	}
	return false
}

// IsSorted reports whether the list is currently sorted by key.
func (s Spec) IsSorted(key string) bool { return key != "" && s.Sort == key }
