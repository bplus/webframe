package filterbar

import (
	"sort"
	"strconv"
	"strings"
)

// Count returns a pointer to n, for FilterOption.Count:
//
//	filterbar.FilterOption{Label: "Evocation", Value: "evocation", Count: filterbar.Count(54)}
func Count(n int) *int { return &n }

// optionText is a select option's text: the label, then the count in
// parentheses when there is one.
func optionText(opt FilterOption) string {
	if opt.Count == nil {
		return opt.Label
	}
	return opt.Label + " (" + strconv.Itoa(*opt.Count) + ")"
}

// choiceSelected is the index of the option a TypeChoice group shows as
// selected: the first with Selected set, -1 when there is none.
func choiceSelected(opts []FilterOption) int {
	for i, o := range opts {
		if o.Selected {
			return i
		}
	}
	return -1
}

// WithSelected returns a copy of filters with the selected state applied
// from values, keyed by FilterConfig.ID: a TypeSelect or TypeToggle option
// is Selected when values[f.ID] contains its Value, a TypeChoice option when
// its Value is the first non-empty values[f.ID] entry (one at most), and a
// TypeSearch filter gets Value set to the first non-empty values[f.ID]
// entry. values is the whole truth: options not named in it come back
// deselected. Toggle values
// may be repeated entries or comma-joined in one entry (as filterbar.min.js
// submits them). values is typically a url.Values or listquery.Spec.Filters.
// The input is never modified.
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
		case TypeChoice:
			c.Options = markSelected(f.Options, firstNonEmpty(vals))
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

// firstNonEmpty returns the first non-empty entry of vals alone, or nil.
func firstNonEmpty(vals []string) []string {
	for _, v := range vals {
		if v != "" {
			return []string{v}
		}
	}
	return nil
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
// name outside [A-Za-z0-9_-] replaced by "-" ("items.sort" becomes
// "bar-hidden-items-sort"), because htmx resolves an hx-swap-oob target as a
// CSS selector, where "." would read as a class. It returns "" when barID is
// "".
func HiddenID(barID, name string) string {
	return partID(barID, "-hidden-", name)
}

// ControlID is the id FilterBar gives the control named name (its
// FilterConfig.ID) in the bar whose Props.ID is barID:
// barID+"-control-"+name, with name made safe as HiddenID makes it. The
// id is on the <select>, the search <input>, or the element that holds a
// toggle's or a choice group's buttons, and it is what ControlsOOB replaces.
// A control that carries list state (see Props.Hidden) has HiddenID
// instead. It returns "" when barID is "".
func ControlID(barID, name string) string {
	return partID(barID, "-control-", name)
}

// partID is barID+infix+name with every character of name outside
// [A-Za-z0-9_-] replaced by "-"; "" when barID is "".
func partID(barID, infix, name string) string {
	if barID == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(barID) + len(infix) + len(name))
	b.WriteString(barID)
	b.WriteString(infix)
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
