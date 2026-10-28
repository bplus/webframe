package wflint

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// Fixtures use the .templ.txt extension so that `templ generate` (run over
// the whole module by `make generate`) neither generates code for them nor
// fails on the deliberately broken one. Lint accepts explicit file paths of
// any extension.

func lintOne(t *testing.T, path string) []Violation {
	t.Helper()
	rep, err := Lint([]string{path})
	if err != nil {
		t.Fatalf("Lint(%q) error: %v", path, err)
	}
	return rep.Violations
}

func TestLint_Clean(t *testing.T) {
	vs := lintOne(t, "testdata/clean.templ.txt")
	if len(vs) != 0 {
		t.Errorf("clean fixture: got %d violations, want 0: %+v", len(vs), vs)
	}
}

func TestLint_Elements(t *testing.T) {
	vs := lintOne(t, "testdata/elements.templ.txt")
	want := []struct {
		line, col int
		elem      string
		hint      string
	}{
		{4, 2, "div", "layout.Stack or layout.Inline"},
		{5, 3, "h1", "pageheader"},
		{6, 3, "table", "datatable (batch 2) or table"},
		{7, 3, "button", "action.Button"},
	}
	if len(vs) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, w := range want {
		v := vs[i]
		if v.Rule != "element" {
			t.Errorf("[%d] Rule = %q, want element", i, v.Rule)
		}
		if v.Line != w.line || v.Col != w.col {
			t.Errorf("[%d] %s at %d:%d, want %d:%d", i, w.elem, v.Line, v.Col, w.line, w.col)
		}
		if v.Hint != w.hint {
			t.Errorf("[%d] Hint = %q, want %q", i, v.Hint, w.hint)
		}
		if !strings.Contains(v.Message, "<"+w.elem+">") {
			t.Errorf("[%d] Message = %q, want it to name <%s>", i, v.Message, w.elem)
		}
		if v.File != "testdata/elements.templ.txt" {
			t.Errorf("[%d] File = %q", i, v.File)
		}
	}
}

func TestLint_Attrs(t *testing.T) {
	vs := lintOne(t, "testdata/attrs.templ.txt")
	want := []struct {
		line int
		rule string
	}{
		{4, "attr-class"},
		{5, "attr-style"},
		{6, "attr-spread"},
		{7, "attr-dynamic-key"},
		{8, "attr-unknown"},
		{9, "style-element"},
		{14, "attr-class"},
		{14, "attr-style"},
	}
	if len(vs) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, w := range want {
		if vs[i].Rule != w.rule || vs[i].Line != w.line {
			t.Errorf("[%d] got %s at line %d, want %s at line %d", i, vs[i].Rule, vs[i].Line, w.rule, w.line)
		}
	}
	// Line 12 (hx-/x-/data-/aria- attributes) and line 13 (a <script> with
	// defer, nonce and src) must be clean: scripts are allowed, and only
	// class/style are checked on them.
	for _, v := range vs {
		if v.Line == 12 || v.Line == 13 {
			t.Errorf("line %d should be clean, got %+v", v.Line, v)
		}
	}
	if !strings.Contains(vs[4].Message, "onclick") {
		t.Errorf("attr-unknown message %q should name onclick", vs[4].Message)
	}
}

// TestLint_ElementsNoAttrRulesOnForbidden: the <div> on line 4 carries class
// and style and the <button> on line 7 carries onclick, but attribute rules
// apply only to allowed elements, so only the element violations appear.
func TestLint_ElementsNoAttrRulesOnForbidden(t *testing.T) {
	for _, v := range lintOne(t, "testdata/elements.templ.txt") {
		if v.Rule != "element" {
			t.Errorf("forbidden element produced %s at line %d", v.Rule, v.Line)
		}
	}
}

func TestLint_Branches(t *testing.T) {
	vs := lintOne(t, "testdata/branches.templ.txt")
	want := []struct {
		line int
		rule string
		site string
	}{
		{7, "element", "else if branch"},
		{9, "element", "else branch"},
		{12, "element", "for loop"},
		{16, "element", "switch case"},
		{19, "element", "template call children"},
		{25, "attr-style", "conditional attribute else"},
	}
	if len(vs) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, w := range want {
		if vs[i].Line != w.line || vs[i].Rule != w.rule {
			t.Errorf("%s: got %s at line %d, want %s at line %d", w.site, vs[i].Rule, vs[i].Line, w.rule, w.line)
		}
	}
}

func TestLint_ParseError(t *testing.T) {
	vs := lintOne(t, "testdata/broken.templ.txt")
	if len(vs) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(vs), vs)
	}
	v := vs[0]
	if v.Rule != "parse-error" || v.Line != 1 || v.File != "testdata/broken.templ.txt" {
		t.Errorf("got %+v, want parse-error at line 1 of broken fixture", v)
	}
	if v.Message == "" {
		t.Errorf("parse-error message is empty")
	}
}

func TestLint_NonexistentPath(t *testing.T) {
	if _, err := Lint([]string{"testdata/nope"}); err == nil {
		t.Fatal("Lint on a nonexistent path returned nil error")
	}
}

func TestLint_SkipsThirdParty(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"third_party", "node_modules", "vendor", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		src := "package x\n\ntempl X() {\n\t<div></div>\n}\n"
		if err := os.WriteFile(filepath.Join(dir, sub, "x.templ"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := Lint([]string{dir})
	if err != nil {
		t.Fatalf("Lint error: %v", err)
	}
	if len(rep.Violations) != 0 {
		t.Errorf("got %d violations, want 0: %+v", len(rep.Violations), rep.Violations)
	}
}

func TestLint_DirWalkOnlyTemplAndSorted(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("b.templ", "package x\n\ntempl B() {\n\t<div></div>\n\t<section></section>\n}\n")
	write("a.templ", "package x\n\ntempl A() {\n\t<nav></nav>\n}\n")
	write("ignored.templ.txt", "package x\n\ntempl C() {\n\t<div></div>\n}\n")
	// Lint b.templ explicitly too, then the dir: order of inputs must not
	// matter and the file must not be reported twice.
	rep, err := Lint([]string{filepath.Join(dir, "b.templ"), dir})
	if err != nil {
		t.Fatalf("Lint error: %v", err)
	}
	if len(rep.Violations) != 3 {
		t.Fatalf("got %d violations, want 3: %+v", len(rep.Violations), rep.Violations)
	}
	sorted := sort.SliceIsSorted(rep.Violations, func(i, j int) bool {
		a, b := rep.Violations[i], rep.Violations[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	if !sorted {
		t.Errorf("violations not sorted: %+v", rep.Violations)
	}
	if !strings.HasSuffix(rep.Violations[0].File, "a.templ") {
		t.Errorf("first violation file = %q, want a.templ", rep.Violations[0].File)
	}
}

// TestLint_Fenced: inside a CustomPage/CustomBlock fence every rule is
// suppressed (block elements, class/style, <style>, Class:, templ.Raw,
// :class); nested fences are each reported; the escaped-quote reason
// round-trips; a fence with no reason is the single violation.
func TestLint_Fenced(t *testing.T) {
	rep, err := Lint([]string{"testdata/fenced.templ.txt"})
	if err != nil {
		t.Fatalf("Lint error: %v", err)
	}
	if len(rep.Violations) != 1 {
		t.Fatalf("got %d violations, want 1: %+v", len(rep.Violations), rep.Violations)
	}
	if v := rep.Violations[0]; v.Rule != "fence-reason" || v.Line != 17 {
		t.Errorf("got %s at line %d, want fence-reason at line 17", v.Rule, v.Line)
	}
	if m := rep.Violations[0].Message; !strings.Contains(m, "double-quoted string literal") {
		t.Errorf("fence-reason message %q should require a double-quoted string literal", m)
	}
	want := []Fence{
		{File: "testdata/fenced.templ.txt", Line: 4, Kind: "page", Reason: `landing page with "hero"`},
		{File: "testdata/fenced.templ.txt", Line: 9, Kind: "block", Reason: "inner"},
		{File: "testdata/fenced.templ.txt", Line: 17, Kind: "block", Reason: ""},
	}
	if len(rep.Fences) != len(want) {
		t.Fatalf("got %d fences, want %d: %+v", len(rep.Fences), len(want), rep.Fences)
	}
	for i, w := range want {
		if rep.Fences[i] != w {
			t.Errorf("fence[%d] = %+v, want %+v", i, rep.Fences[i], w)
		}
	}
}

func TestLint_GoCode(t *testing.T) {
	vs := lintOne(t, "testdata/gocode.templ.txt")
	want := []struct {
		line int
		rule string
	}{
		{5, "html-from-go"},  // top-level Go section
		{8, "props-class"},   // Class: in a component call
		{9, "attrs-map"},     // templ.Attributes{"style": ...} in a component call
		{10, "html-from-go"}, // { templ.Raw(s) }
		{11, "attr-binding"}, // :class
		{12, "attr-binding"}, // x-bind:style
		{14, "html-from-go"}, // @templ.Raw(s)
		{15, "html-from-go"}, // {{ templ.FromGoHTML }}
		{17, "html-from-go"}, // {! templ.Raw(s) }
		{18, "props-class"},  // {! card.Card(card.Props{Class: ...}) }
		{19, "props-class"},  // {{ p := card.Props{Class: ...} }}
		{21, "attrs-map"},    // {{ a := templ.Attributes{"class": ...} }}
		{22, "props-class"},  // Class:"p-2" with no space
		{23, "attrs-map"},    // map[string]any{"class": ...}
		{24, "attrs-map"},    // "Style" key, any case
		{25, "attrs-map"},    // "}" inside a string does not end the literal
		{28, "props-class"},  // Class: on its own line of a multi-line Props literal
		{32, "props-class"},  // {{ dashboardgrid.SlotProps{Class: ...} }}: any *Props literal
		{35, "props-class"},  // top-level var hero = card.Card(card.Props{Class: ...})
	}
	if len(vs) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, w := range want {
		if vs[i].Rule != w.rule || vs[i].Line != w.line {
			t.Errorf("[%d] got %s at line %d, want %s at line %d", i, vs[i].Rule, vs[i].Line, w.rule, w.line)
		}
	}
	// Line 13 (x-show), line 16 (templ.Attributes with only hx-get), line 20
	// (@card.Card(p)) and line 31 (Class: in a non-Props struct in Go code)
	// are clean.
	for _, v := range vs {
		if v.Line == 13 || v.Line == 16 || v.Line == 20 || v.Line == 31 {
			t.Errorf("line %d should be clean, got %+v", v.Line, v)
		}
	}
}

// TestLint_GoFiles: a hand-written .go file is scanned for html-from-go,
// Class: inside a Props literal, and class/style keys in attribute maps;
// Class: on other structs is not flagged.
func TestLint_GoFiles(t *testing.T) {
	vs := lintOne(t, "testdata/handler.go")
	want := []struct {
		line int
		rule string
	}{
		{7, "html-from-go"},
		{13, "props-class"}, // multi-line card.Props literal
		{19, "attrs-map"},   // map[string]any{"style": ...}
		{22, "props-class"}, // dropdown.ContentProps{Class: ...}: any *Props literal
	}
	if len(vs) != len(want) {
		t.Fatalf("handler.go: got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, w := range want {
		if vs[i].Rule != w.rule || vs[i].Line != w.line || vs[i].File != "testdata/handler.go" {
			t.Errorf("[%d] got %s at line %d, want %s at line %d", i, vs[i].Rule, vs[i].Line, w.rule, w.line)
		}
	}
}

// TestLint_HTMLFromGoString: a Go string literal holding a class= or
// style= attribute or a forbidden element's opening tag is reported at the
// literal, in .go files and in the Go sections of .templ files; comments,
// rune literals, allowed elements and look-alikes are not.
func TestLint_HTMLFromGoString(t *testing.T) {
	for file, wantLines := range map[string][]int{
		"testdata/htmlstring.templ.txt": {3, 6, 12, 14},
		"testdata/htmlstring.go":        {7, 13},
	} {
		vs := lintOne(t, file)
		var got []int
		for _, v := range vs {
			if v.Rule != "html-from-go-string" {
				t.Errorf("%s: unexpected %+v", file, v)
				continue
			}
			if v.Hint != "render a webframe component instead" {
				t.Errorf("%s: Hint = %q", file, v.Hint)
			}
			got = append(got, v.Line)
		}
		if !slices.Equal(got, wantLines) {
			t.Errorf("%s: html-from-go-string at lines %v, want %v", file, got, wantLines)
		}
	}
}

// TestLint_GeneratedGoNeedsNameAndHeader: a .go file is skipped as generated
// only when it is named *_templ.go AND carries the templ header, so a
// hand-written file cannot opt out by adding the comment. The *_templ.go
// cases live in a temp dir because `templ generate` deletes an orphan
// *_templ.go from the source tree.
func TestLint_GeneratedGoNeedsNameAndHeader(t *testing.T) {
	// Header only: scanned.
	vs := lintOne(t, "testdata/generated.go")
	if len(vs) != 1 || vs[0].Rule != "html-from-go" || vs[0].Line != 7 {
		t.Errorf("generated.go (header, wrong name): got %+v, want one html-from-go at line 7", vs)
	}
	dir := t.TempDir()
	body := "package x\n\nvar v = templ.Raw(\"<b>\")\n"
	header := "// Code generated by templ - DO NOT EDIT.\n\n"
	for name, src := range map[string]string{"real_templ.go": header + body, "fake_templ.go": body} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if vs := lintOne(t, filepath.Join(dir, "real_templ.go")); len(vs) != 0 {
		t.Errorf("real_templ.go (name and header): got %+v, want 0", vs)
	}
	if vs := lintOne(t, filepath.Join(dir, "fake_templ.go")); len(vs) != 1 || vs[0].Rule != "html-from-go" {
		t.Errorf("fake_templ.go (name, no header): got %+v, want one html-from-go", vs)
	}
}

func TestLint_CSSFile(t *testing.T) {
	vs := lintOne(t, "testdata/app.css")
	if len(vs) != 1 || vs[0].Rule != "css-file" || vs[0].Line != 1 || vs[0].File != "testdata/app.css" {
		t.Errorf("got %+v, want one css-file at line 1", vs)
	}
}

// TestLint_DirWalkGoAndCSS: a directory walk collects *.templ, *.go
// (generated files skipped) and *.css.
func TestLint_DirWalkGoAndCSS(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("h.go", "package x\n\nvar v = templ.Raw(\"<b>\")\n")
	write("v_templ.go", "// Code generated by templ - DO NOT EDIT.\n\npackage x\n\nvar v = templ.Raw(\"<b>\")\n")
	write("s.css", "p{}\n")
	write("p.templ", "package x\n\ntempl P() {\n\t<div></div>\n}\n")
	rep, err := Lint([]string{dir})
	if err != nil {
		t.Fatalf("Lint error: %v", err)
	}
	got := map[string]string{}
	for _, v := range rep.Violations {
		got[filepath.Base(v.File)] = v.Rule
	}
	want := map[string]string{"h.go": "html-from-go", "s.css": "css-file", "p.templ": "element"}
	if len(rep.Violations) != 3 || len(got) != 3 {
		t.Fatalf("got %+v, want one violation each in %v", rep.Violations, want)
	}
	for f, r := range want {
		if got[f] != r {
			t.Errorf("%s: got rule %q, want %q", f, got[f], r)
		}
	}
}

func TestFenceHelpers(t *testing.T) {
	for _, c := range []struct {
		expr, kind string
		ok         bool
	}{
		{`custompage.CustomPage(custompage.Props{Reason: "x"})`, "page", true},
		{`  customblock.CustomBlock()`, "block", true},
		{`card.Card()`, "", false},
	} {
		k, ok := fenceOf(c.expr)
		if k != c.kind || ok != c.ok {
			t.Errorf("fenceOf(%q) = %q, %v; want %q, %v", c.expr, k, ok, c.kind, c.ok)
		}
	}
	for _, c := range []struct {
		expr, reason string
		present      bool
	}{
		{`customblock.CustomBlock(customblock.Props{Reason: "a \"b\" c\\d"})`, `a "b" c\d`, true},
		{`customblock.CustomBlock(customblock.Props{Reason: ""})`, "", false},
		{`customblock.CustomBlock(customblock.Props{Reason: "   "})`, "   ", false},
		{`customblock.CustomBlock()`, "", false},
	} {
		r, ok := fenceReason(c.expr)
		if r != c.reason || ok != c.present {
			t.Errorf("fenceReason(%q) = %q, %v; want %q, %v", c.expr, r, ok, c.reason, c.present)
		}
	}
}
