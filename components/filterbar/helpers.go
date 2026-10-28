package filterbar

import (
	"sort"
	"strings"
)

// WithSelected returns a copy of filters with the selected state applied
// from values, keyed by FilterConfig.ID: a TypeSelect or TypeToggle option
// is Selected when values[f.ID] contains its Value, and a TypeSearch filter
// gets Value set to the first non-empty values[f.ID] entry. values is the
// whole truth: options not named in it come back deselected even if the
// input marked them. Toggle values may arrive either as repeated entries or
// comma-joined in one entry (the form filterbar.min.js submits); both are
// understood.
//
// values is typically listquery.Spec.Filters (plus "q" for the search) or a
// url.Values, which is assignable to map[string][]string. The input slice
// and its Options are never modified; the result shares nothing with them.
func WithSelected(filters []FilterConfig, values map[string][]string) []FilterConfig {
	if filters == nil {
		return nil
	}
	out := make([]FilterConfig, len(filters))
	for i, f := range filters {
		c := f
		vals := values[f.ID]
		switch f.Type {
		case TypeSearch:
			c.Value = ""
			for _, v := range vals {
				if v != "" {
					c.Value = v
					break
				}
			}
		case TypeToggle:
			c.Options = markSelected(f.Options, splitComma(vals))
		default:
			c.Options = markSelected(f.Options, vals)
		}
		if f.Type != TypeSearch && f.Options == nil {
			c.Options = nil
		}
		out[i] = c
	}
	return out
}

// markSelected copies opts, marking each whose Value appears in vals.
func markSelected(opts []FilterOption, vals []string) []FilterOption {
	if opts == nil {
		return nil
	}
	set := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		set[v] = struct{}{}
	}
	out := make([]FilterOption, len(opts))
	for i, o := range opts {
		_, o.Selected = set[o.Value]
		out[i] = o
	}
	return out
}

// splitComma expands "a,b" entries into separate values; plain entries pass
// through. Empty parts are dropped.
func splitComma(vals []string) []string {
	var out []string
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// sortedKeys returns m's keys in ascending order so hidden inputs render
// deterministically. nil or empty m yields nil.
func sortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
