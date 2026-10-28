package wflint

import (
	"regexp"
	"strings"
)

// fencePrefixes map the call prefix of each sanctioned escape hatch to its
// Fence kind.
var fencePrefixes = []struct{ prefix, kind string }{
	{"custompage.CustomPage(", "page"},
	{"customblock.CustomBlock(", "block"},
}

// fenceOf reports whether a templ element expression (the text after @) is a
// CustomPage or CustomBlock call, and which kind.
func fenceOf(expr string) (kind string, ok bool) {
	expr = strings.TrimSpace(expr)
	for _, f := range fencePrefixes {
		if strings.HasPrefix(expr, f.prefix) {
			return f.kind, true
		}
	}
	return "", false
}

var reasonRe = regexp.MustCompile(`Reason:\s*"((?:[^"\\]|\\.)*)"`)

// fenceReason extracts the Reason string literal from a fence expression,
// unescaping \" and \\. present is false when the field is absent or empty.
func fenceReason(expr string) (reason string, present bool) {
	m := reasonRe.FindStringSubmatch(expr)
	if m == nil {
		return "", false
	}
	var b strings.Builder
	s := m[1]
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
			i++
		}
		b.WriteByte(s[i])
	}
	reason = b.String()
	return reason, reason != ""
}
