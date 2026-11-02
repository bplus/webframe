package listquery

import (
	"net/url"
	"strconv"
	"strings"
)

// Override edits a copy of a Spec before URL encodes it.
type Override func(*Spec)

// URL builds base?params for the Spec with overrides applied, in order. Values
// are query-escaped with net/url and keys are emitted in sorted order, so the
// same state always yields the same string. Parameters equal to the zero value
// are omitted to keep URLs short: page 1, an empty search, empty filters, an
// empty sort (and dir, which means nothing without a sort), and a page size
// equal to the default ParseValues resolved from its Options (a hand-built
// Spec has no default, so its page size is written whenever it is set).
//
// The base is split at its first "#" and "?", not concatenated onto: the
// fragment stays last, after the query. Parameters the Spec owns (sort, dir,
// page, per, q and its filter keys, all under its prefix; the filter keys
// are Options.FilterKeys plus any key in Filters) are removed from the base's
// query and written from the Spec, so a base carrying ?page=3 is replaced,
// not duplicated, and an omitted zero value removes it. Every other base
// parameter is kept verbatim and in order, before the Spec's. A base with no
// owned parameters and a Spec with nothing to write is returned unchanged.
// The receiver is never modified; overrides work on a deep copy, and the
// prefix, default size and filter keys carry into every derived Spec.
func (s Spec) URL(base string, overrides ...Override) string {
	d := s.clone()
	for _, o := range overrides {
		if o != nil {
			o(&d)
		}
	}
	q := d.values()

	rest, frag, hasFrag := strings.Cut(base, "#")
	path, rawQuery, _ := strings.Cut(rest, "?")
	owned := d.owned(s)
	var kept []string
	removed := false
	if rawQuery != "" {
		for _, part := range strings.Split(rawQuery, "&") {
			key, _, _ := strings.Cut(part, "=")
			if k, err := url.QueryUnescape(key); err == nil {
				key = k
			}
			if part != "" && owned[key] {
				removed = true
				continue
			}
			kept = append(kept, part)
		}
	}
	if !removed && len(q) == 0 {
		return base
	}
	if enc := q.Encode(); enc != "" {
		kept = append(kept, enc)
	}
	out := path
	if query := strings.Join(kept, "&"); query != "" {
		out += "?" + query
	}
	if hasFrag {
		out += "#" + frag
	}
	return out
}

// owned is the set of query parameter names d writes or clears, under its
// prefix: the fixed names, the filter keys d was parsed with, and every key
// in d's or orig's Filters (orig is the Spec before overrides, so a filter an
// override deleted is still cleared from the base).
func (d Spec) owned(orig Spec) map[string]bool {
	m := map[string]bool{}
	for _, name := range []string{"sort", "dir", "page", "per", "q"} {
		m[d.Param(name)] = true
	}
	for _, k := range d.filterKeys {
		if k != "" {
			m[d.Param(k)] = true
		}
	}
	for _, f := range []map[string][]string{d.Filters, orig.Filters} {
		for k := range f {
			if k != "" {
				m[d.Param(k)] = true
			}
		}
	}
	return m
}

// Values returns the Spec's non-zero parameters under its prefix, the same set
// URL writes. Useful for hidden form inputs that must carry list state.
func (s Spec) Values() url.Values { return s.values() }

func (s Spec) values() url.Values {
	q := url.Values{}
	if s.Sort != "" && !(s.Sort == s.defaultSort && s.Dir == s.defaultDir) {
		q.Set(s.Param("sort"), s.Sort)
		if s.Dir == Asc || s.Dir == Desc {
			q.Set(s.Param("dir"), string(s.Dir))
		}
	}
	if s.Page > 1 {
		q.Set(s.Param("page"), strconv.Itoa(s.Page))
	}
	if s.PageSize > 0 && s.PageSize != s.defaultSize {
		q.Set(s.Param("per"), strconv.Itoa(s.PageSize))
	}
	if s.Search != "" {
		q.Set(s.Param("q"), s.Search)
	}
	for key, vals := range s.Filters {
		if key == "" {
			continue
		}
		for _, v := range vals {
			if v != "" {
				q.Add(s.Param(key), v)
			}
		}
	}
	return q
}

// clone deep-copies the Filters map so overrides never reach the receiver.
func (s Spec) clone() Spec {
	d := s
	if s.Filters != nil {
		d.Filters = make(map[string][]string, len(s.Filters))
		for k, vals := range s.Filters {
			d.Filters[k] = append([]string(nil), vals...)
		}
	}
	return d
}

// WithSort sorts by key: the same key flips the direction, a new key sorts
// ascending. The page resets to 1.
func WithSort(key string) Override {
	return func(s *Spec) {
		if s.Sort == key && key != "" {
			if s.Dir == Desc {
				s.Dir = Asc
			} else {
				s.Dir = Desc
			}
		} else {
			s.Sort = key
			s.Dir = Asc
		}
		s.Page = 1
	}
}

// WithPage moves to page n (values below 1 mean page 1).
func WithPage(n int) Override {
	return func(s *Spec) {
		if n < 1 {
			n = 1
		}
		s.Page = n
	}
}

// WithPageSize changes the page size (values below 1 are ignored) and resets
// the page to 1. Parse re-clamps the size to MaxPageSize on the next request.
func WithPageSize(n int) Override {
	return func(s *Spec) {
		if n >= 1 {
			s.PageSize = n
		}
		s.Page = 1
	}
}

// WithSearch replaces the search text and resets the page to 1.
func WithSearch(q string) Override {
	return func(s *Spec) {
		s.Search = q
		s.Page = 1
	}
}

// WithFilter replaces the values for key; no values, or only empty strings,
// delete the key. The page resets to 1.
func WithFilter(key string, values ...string) Override {
	return func(s *Spec) {
		s.Page = 1
		if key == "" {
			return
		}
		var vals []string
		for _, v := range values {
			if v != "" {
				vals = append(vals, v)
			}
		}
		if len(vals) == 0 {
			delete(s.Filters, key)
			if len(s.Filters) == 0 {
				s.Filters = nil
			}
			return
		}
		if s.Filters == nil {
			s.Filters = make(map[string][]string, 1)
		}
		s.Filters[key] = vals
	}
}

// WithoutFilters clears the search and every filter, keeping sort and page
// size. The page resets to 1 because the result set changes.
func WithoutFilters() Override {
	return func(s *Spec) {
		s.Search = ""
		s.Filters = nil
		s.Page = 1
	}
}
