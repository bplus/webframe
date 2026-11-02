package listquery

import "sort"

// Page is one page of an in-memory list after Apply. Spec carries the page
// actually served (clamped to the last page) so pagination renders truthfully.
type Page[T any] struct {
	Items []T
	Total int
	Spec  Spec
}

// Apply filters, sorts and pages items in memory. match keeps the items for
// which it returns true; a nil match keeps every item. sorters maps a sort key
// to a less function; a Spec.Sort with no sorter (or nil sorters) leaves the
// input order. Desc reverses the less function and the sort is stable, so
// equal items keep their input order in both directions. The caller's slice
// is never reordered. A Page beyond the last page clamps to the last page;
// a zero-value Spec pages with the default size.
func Apply[T any](items []T, s Spec, sorters map[string]func(a, b T) bool, match func(item T, s Spec) bool) Page[T] {
	kept := make([]T, 0, len(items))
	for _, it := range items {
		if match == nil || match(it, s) {
			kept = append(kept, it)
		}
	}

	if less, ok := sorters[s.Sort]; ok && less != nil && s.Sort != "" {
		if s.Dir == Desc {
			sort.SliceStable(kept, func(i, j int) bool { return less(kept[j], kept[i]) })
		} else {
			sort.SliceStable(kept, func(i, j int) bool { return less(kept[i], kept[j]) })
		}
	}

	total := len(kept)
	out := s
	out.PageSize = s.pageSize()
	out.Page = s.page()
	if last := s.Pages(total); out.Page > last {
		out.Page = last
	}
	start := (out.Page - 1) * out.PageSize
	if start > total {
		start = total
	}
	end := start + out.PageSize
	if end > total {
		end = total
	}
	return Page[T]{Items: kept[start:end], Total: total, Spec: out}
}
