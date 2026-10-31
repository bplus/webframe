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

// HiddenID is the id FilterBar gives the hidden input named name in the bar
// whose Props.ID is barID: barID+"-hidden-"+name, with every character of
// name outside [A-Za-z0-9_-] replaced by "-" ("items.sort" →
// "bar-hidden-items-sort"). The id has to work in a CSS selector because
// htmx resolves an hx-swap-oob target as querySelectorAll("#"+id), where a
// "." would read as a class. HiddenOOB applies the same rule, so the two
// always agree. "" when barID is "".
func HiddenID(barID, name string) string {
	if barID == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(barID) + 8 + len(name))
	b.WriteString(barID)
	b.WriteString("-hidden-")
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
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

// stateControl returns the control in filters promoted to carry the list
// state named key (see Props.Hidden): the first TypeSelect or TypeSearch
// FilterConfig whose ID is key. ok is false when there is none; a
// TypeToggle with that ID does not count.
func stateControl(filters []FilterConfig, key string) (f FilterConfig, ok bool) {
	for _, c := range filters {
		if c.ID == key && (c.Type == TypeSelect || c.Type == TypeSearch) {
			return c, true
		}
	}
	return FilterConfig{}, false
}
