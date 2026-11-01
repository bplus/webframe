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
	"html-from-go-string": {"a Go string literal holding HTML markup (a quoted class= or style= attribute, or a block element such as <div> with markup around it) is not allowed in app code", "render a webframe component instead"},
}

// proseImportPath is the one package whose HTML values may be built from
// string literals in app code: prose is the sanctioned route for trusted
// content markup (its HTML type is the one templ.Raw in the library).
const proseImportPath = "github.com/bplus/webframe/components/prose"

// proseImportRe finds the file's import of prose, with its alias when it
// has one: a one-line import statement or a line of an import block. The
// import keyword is tried before the line start so it is never read as the
// alias.
var proseImportRe = regexp.MustCompile(`(?m)(?:\bimport|^)[ \t]*(?:([A-Za-z_]\w*)[ \t]+)?"` + regexp.QuoteMeta(proseImportPath) + `"`)

// htmlKeyRe finds the HTML: key of a prose.Props literal.
var htmlKeyRe = regexp.MustCompile(`\bHTML:\s*`)

// proseScope knows how one file refers to the prose package, so the
// html-from-go-string rule can leave a literal alone when it is content
// handed to prose: the argument of a prose.HTML(...) conversion or the value
// of the HTML: key in a prose.Props{...} literal, under whatever name the
// file imports the package. A nil *proseScope means the file does not
// import prose (or imports it blank), so nothing is exempt.
type proseScope struct {
	conv *regexp.Regexp // alias.HTML(
	lit  *regexp.Regexp // alias.Props{
}

// proseScopeOf resolves the file's import of prose from the whole source
// (a .go file, or a .templ file whose import block is one of its Go
// sections). A dot import is not resolved: the exemption needs a selector.
func proseScopeOf(src string) *proseScope {
	m := proseImportRe.FindStringSubmatch(src)
	if m == nil {
		return nil
	}
	alias := m[1]
	switch alias {
	case "":
		alias = "prose"
	case "_":
		return nil
	}
	q := `\b` + regexp.QuoteMeta(alias)
	return &proseScope{
		conv: regexp.MustCompile(q + `\.HTML\(`),
		lit:  regexp.MustCompile(q + `\.Props\{`),
	}
}

// exempt returns the byte ranges of src whose string literals are content
// handed to prose: the whole argument list of each alias.HTML(...) and the
// value expression after each HTML: key inside an alias.Props{...} literal
// (up to the next comma or closing brace at the same depth).
func (s *proseScope) exempt(src string) [][2]int {
	if s == nil {
		return nil
	}
	var out [][2]int
	for _, loc := range s.conv.FindAllStringIndex(src, -1) {
		open := loc[1] - 1
		out = append(out, [2]int{open, open + len(delimBody(src[open:], '(', ')'))})
	}
	for _, loc := range s.lit.FindAllStringIndex(src, -1) {
		open := loc[1] - 1
		body := delimBody(src[open:], '{', '}')
		for _, m := range htmlKeyRe.FindAllStringIndex(body, -1) {
			start := open + m[1]
			out = append(out, [2]int{start, start + exprEnd(body[m[1]:])})
		}
	}
	return out
}

// inRanges reports whether off falls inside one of ranges.
func inRanges(off int, ranges [][2]int) bool {
	for _, r := range ranges {
		if off >= r[0] && off < r[1] {
			return true
		}
	}
	return false
}

// quotedValue matches the start of a quoted attribute value: = followed by a
// quote, which an interpreted literal escapes ("class=\"x\"").
const quotedValue = `=\\?["'` + "`" + `]`

// markupAttrRe finds a class or style attribute inside a Go string literal:
// either with a quoted value, or with its = ending the literal (the value is
// concatenated from outside: "<div class=" + q + ">"). Case-sensitive, and
// anything else after the = is not markup, so a log format ("class=%s") or
// SQL ("style= ?") does not match. It runs on the literal's contents without
// the surrounding quotes, so $ is the end of the contents.
var markupAttrRe = regexp.MustCompile(`\s(?:class|style)(?:` + quotedValue + `|=$)`)

// markupTagRe finds an opening tag of a forbidden element (every element
// that has a component Hint) inside a Go string literal. Case-sensitive:
// "<Label>" in usage text is not a tag.
var markupTagRe = func() *regexp.Regexp {
	names := make([]string, 0, len(Hints))
	for n := range Hints {
		names = append(names, regexp.QuoteMeta(n))
	}
	sort.Strings(names)
	return regexp.MustCompile(`<(?:` + strings.Join(names, "|") + `)(?:[\s/>]|$)`)
}()

// markupAroundRe finds what separates markup from a mention of a tag: a
// closing tag, a quoted attribute value, or one tag directly after another
// ("<table><tr>"). A forbidden tag is reported only alongside one of these,
// so usage text such as "<input> is not allowed" or "<div> → layout.Stack"
// passes.
var markupAroundRe = regexp.MustCompile(`</|` + quotedValue + `|><`)

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
// string literals holding HTML markup are reported as html-from-go-string,
// except the literals prose exempts (content handed to prose.HTML or to
// the HTML field of prose.Props; see proseScope).
func goFindings(src string, component bool, prose *proseScope) []goFinding {
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
		exempt := prose.exempt(src)
		for i, v := range vals {
			if inRanges(offs[i], exempt) {
				continue
			}
			if markupAttrRe.MatchString(v) || (markupTagRe.MatchString(v) && markupAroundRe.MatchString(v)) {
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
	for _, f := range goFindings(src, true, nil) {
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
	return delimBody(s, '{', '}')
}

// delimBody returns s (which starts with open) up to its matching close, or
// all of s when unbalanced. Delimiters inside interpreted, raw and rune
// literals are skipped.
func delimBody(s string, open, close byte) string {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			i = skipQuoted(s, i)
		case '`':
			i = skipRaw(s, i)
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}
	return s
}

// exprEnd returns the length of the Go expression that starts s: up to the
// first ',', '}' or ')' at depth zero outside string and rune literals, or
// len(s) when none closes it.
func exprEnd(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			i = skipQuoted(s, i)
		case '`':
			i = skipRaw(s, i)
		case '{', '(', '[':
			depth++
		case '}', ')', ']':
			if depth == 0 {
				return i
			}
			depth--
		case ',':
			if depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

// skipQuoted returns the index of the closing quote of the interpreted
// string or rune literal opening at s[i] (or of the line end, for an
// unterminated one).
func skipQuoted(s string, i int) int {
	q := s[i]
	for i++; i < len(s) && s[i] != q && s[i] != '\n'; i++ {
		if s[i] == '\\' {
			i++
		}
	}
	return i
}

// skipRaw returns the index of the closing backquote of the raw string
// literal opening at s[i] (or len(s) for an unterminated one).
func skipRaw(s string, i int) int {
	for i++; i < len(s) && s[i] != '`'; i++ {
	}
	return i
}

// reportGo reports the Go-side rules breached by expr, each at the line and
// column of the match. component marks a @component(...) or {! } call
// expression (see goFindings).
func (w *walker) reportGo(expr parser.Expression, component bool) {
	base := int(expr.Range.From.Line) + 1
	col := int(expr.Range.From.Col) + 1
	for _, f := range goFindings(expr.Value, component, w.prose) {
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
// narrow Props-literal form of props-class. A *_test.go file is exempt from
// html-from-go-string only.
func lintGo(file, src string) []Violation {
	if isGenerated(file, src) {
		return nil
	}
	w := &walker{file: file, prose: proseScopeOf(src)}
	w.reportGo(parser.Expression{Value: src}, false)
	if !strings.HasSuffix(file, "_test.go") {
		return w.out
	}
	// Tests assert on rendered HTML, so markup in their string literals is
	// expected; every other rule still applies.
	out := w.out[:0]
	for _, v := range w.out {
		if v.Rule != "html-from-go-string" {
			out = append(out, v)
		}
	}
	return out
}
