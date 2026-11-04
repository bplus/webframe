package wflint

import (
	"regexp"
	"strings"
)

// fencePackages are the sanctioned escape hatches: the package path, the
// name it has when imported without an alias, the fenced function and the
// Fence kind.
var fencePackages = []struct {
	re      *regexp.Regexp
	def, fn string
	kind    string
}{
	{importNameRe("github.com/bplus/webframe/components/custompage"), "custompage", "CustomPage", "page"},
	{importNameRe("github.com/bplus/webframe/components/customblock"), "customblock", "CustomBlock", "block"},
}

// fencePrefix is the call text that opens one kind of fence in a file.
type fencePrefix struct{ prefix, kind string }

// fencePrefixesOf resolves how src names each fence package, the same way
// proseScopeOf resolves prose: under its import alias when the file has
// one, otherwise under the default name (a file with no import of the
// package, such as a test fixture, still has its default-name calls
// recognized; there they could not compile under any other meaning).
func fencePrefixesOf(src string) []fencePrefix {
	out := make([]fencePrefix, 0, len(fencePackages))
	for _, f := range fencePackages {
		name, ok := importName(src, f.re, f.def)
		if !ok {
			name = f.def
		}
		out = append(out, fencePrefix{name + "." + f.fn + "(", f.kind})
	}
	return out
}

// fenceOf reports whether a templ element expression (the text after @) is a
// CustomPage or CustomBlock call under the names in prefixes, and which kind.
func fenceOf(expr string, prefixes []fencePrefix) (kind string, ok bool) {
	expr = strings.TrimSpace(expr)
	for _, f := range prefixes {
		if strings.HasPrefix(expr, f.prefix) {
			return f.kind, true
		}
	}
	return "", false
}

var reasonRe = regexp.MustCompile(`Reason:\s*"((?:[^"\\]|\\.)*)"`)

// fenceReason extracts the Reason string literal from a fence expression,
// unescaping \" and \\. present is false when the field is absent, empty, or
// only whitespace.
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
	return reason, strings.TrimSpace(reason) != ""
}
