package wflint

import (
	"regexp"
	"sort"
	"strings"

	"github.com/a-h/templ/parser/v2"
)

var (
	htmlFromGoRe = regexp.MustCompile(`\btempl\.(Raw|FromGoHTML|ComponentFunc)\(`)
	// propsClassRe finds a Class field anywhere; used inside component call
	// expressions, where any Class: is a Props field.
	propsClassRe = regexp.MustCompile(`\bClass:\s*`)
	// propsLitRe finds the opening of any *Props composite literal (Props{,
	// card.Props{, dashboardgrid.SlotProps{, dropdown.ContentProps{); in
	// plain Go code Class: is reported only inside one, since other structs
	// may legitimately have a Class field.
	propsLitRe = regexp.MustCompile(`\b(?:[A-Za-z_]\w*\.)?\w*Props\{`)
	// attrsMapRe finds the opening of an attribute-map literal.
	attrsMapRe = regexp.MustCompile(`\btempl\.Attributes\{|\bmap\[string\](?:any|interface\{\})\{`)
	// classStyleKey finds a quoted class or style key, in any case.
	classStyleKey = regexp.MustCompile(`(?i)"(class|style)"\s*:`)
)

var goRuleText = map[string][2]string{
	"html-from-go":        {"building HTML from Go strings (templ.Raw, templ.FromGoHTML, templ.ComponentFunc) is not allowed in app code", "prose for content HTML"},
	"props-class":         {"Class: is not allowed in webframe Props in app code; styling belongs inside webframe components", ""},
	"attrs-map":           {`an attribute map (templ.Attributes or map[string]any) with a "class" or "style" key is not allowed in app code`, ""},
	"html-from-go-string": {"a Go string literal holding HTML markup (a class= or style= attribute, or a block element such as <div>) is not allowed in app code", "render a webframe component instead"},
}

// markupAttrRe finds a class or style attribute inside a Go string literal.
var markupAttrRe = regexp.MustCompile(`(?i)\s(?:class|style)=`)

// markupTagRe finds an opening tag of a forbidden element (every element
// that has a component Hint) inside a Go string literal.
var markupTagRe = func() *regexp.Regexp {
	names := make([]string, 0, len(Hints))
	for n := range Hints {
		names = append(names, regexp.QuoteMeta(n))
	}
	sort.Strings(names)
	return regexp.MustCompile(`(?i)<(?:` + strings.Join(names, "|") + `)(?:[\s/>]|$)`)
}()

// goStringLits returns the byte offset of every interpreted ("...") and raw
// (`...`) string literal in Go source src, with its contents. Comments and
// rune literals are skipped.
func goStringLits(src string) (offs []int, vals []string) {
	for i := 0; i < len(src); i++ {
		switch c := src[i]; {
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return
			}
			i += end + 3
		case c == '\'':
			for i++; i < len(src) && src[i] != '\'' && src[i] != '\n'; i++ {
				if src[i] == '\\' {
					i++
				}
			}
		case c == '"':
			start := i
			for i++; i < len(src) && src[i] != '"' && src[i] != '\n'; i++ {
				if src[i] == '\\' {
					i++
				}
			}
			offs, vals = append(offs, start), append(vals, src[start+1:min(i, len(src))])
		case c == '`':
			start := i
			for i++; i < len(src) && src[i] != '`'; i++ {
			}
			offs, vals = append(offs, start), append(vals, src[start+1:min(i, len(src))])
		}
	}
	return
}

// goFinding is one rule breach at a byte offset into a Go source.
type goFinding struct {
	off  int
	rule string
}

// goFindings returns every breach in src, sorted by offset. With component
// (src is a @component(...) or {! } call expression), any Class: is a
// props-class; otherwise only a Class: inside a Props literal is, and Go
// string literals holding HTML markup are reported as html-from-go-string.
func goFindings(src string, component bool) []goFinding {
	var out []goFinding
	seen := map[goFinding]bool{}
	add := func(off int, rule string) {
		f := goFinding{off, rule}
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	for _, loc := range htmlFromGoRe.FindAllStringIndex(src, -1) {
		add(loc[0], "html-from-go")
	}
	if component {
		for _, loc := range propsClassRe.FindAllStringIndex(src, -1) {
			add(loc[0], "props-class")
		}
	} else {
		offs, vals := goStringLits(src)
		for i, v := range vals {
			if markupAttrRe.MatchString(v) || markupTagRe.MatchString(v) {
				add(offs[i], "html-from-go-string")
			}
		}
		for _, loc := range propsLitRe.FindAllStringIndex(src, -1) {
			open := loc[1] - 1
			body := braceBody(src[open:])
			for _, m := range propsClassRe.FindAllStringIndex(body, -1) {
				add(open+m[0], "props-class")
			}
		}
	}
	for _, loc := range attrsMapRe.FindAllStringIndex(src, -1) {
		open := loc[1] - 1
		if m := classStyleKey.FindStringIndex(braceBody(src[open:])); m != nil {
			add(open+m[0], "attrs-map")
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].off < out[j].off })
	return out
}

// goCodeViolations returns the names of the rules a component call
// expression breaches, each at most once, in the order html-from-go,
// props-class, attrs-map.
func goCodeViolations(src string) []string {
	found := map[string]bool{}
	for _, f := range goFindings(src, true) {
		found[f.rule] = true
	}
	var rules []string
	for _, r := range []string{"html-from-go", "props-class", "attrs-map"} {
		if found[r] {
			rules = append(rules, r)
		}
	}
	return rules
}

// braceBody returns s (which starts with '{') up to its matching '}', or all
// of s when unbalanced. Braces inside interpreted, raw and rune literals are
// skipped.
func braceBody(s string) string {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			for i++; i < len(s) && s[i] != c && s[i] != '\n'; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		case '`':
			for i++; i < len(s) && s[i] != '`'; i++ {
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}
	return s
}

// reportGo reports the Go-side rules breached by expr, each at the line and
// column of the match. component marks a @component(...) or {! } call
// expression (see goFindings).
func (w *walker) reportGo(expr parser.Expression, component bool) {
	base := int(expr.Range.From.Line) + 1
	col := int(expr.Range.From.Col) + 1
	for _, f := range goFindings(expr.Value, component) {
		before := expr.Value[:f.off]
		line := base + strings.Count(before, "\n")
		c := col + f.off
		if nl := strings.LastIndexByte(before, '\n'); nl >= 0 {
			c = f.off - nl
		}
		w.out = append(w.out, Violation{File: w.file, Line: line, Col: c, Rule: f.rule,
			Message: goRuleText[f.rule][0], Hint: goRuleText[f.rule][1]})
	}
}

// isGenerated reports whether a Go file is templ output: it must be named
// *_templ.go and carry the templ generated-code header before its package
// clause, so a hand-written file cannot opt out by adding the comment.
func isGenerated(file, src string) bool {
	if !strings.HasSuffix(file, "_templ.go") {
		return false
	}
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "// Code generated by templ") {
			return true
		}
		if strings.HasPrefix(line, "package ") {
			return false
		}
	}
	return false
}

// lintGo scans a non-generated Go file for the Go-side rules, with the
// narrow Props-literal form of props-class.
func lintGo(file, src string) []Violation {
	if isGenerated(file, src) {
		return nil
	}
	w := &walker{file: file}
	w.reportGo(parser.Expression{Value: src}, false)
	return w.out
}
