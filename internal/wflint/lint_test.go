package wflint

import (
	"os"
	"path/filepath"
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
	}
	if len(vs) != len(want) {
		t.Fatalf("got %d violations, want %d: %+v", len(vs), len(want), vs)
	}
	for i, w := range want {
		if vs[i].Rule != w.rule || vs[i].Line != w.line {
			t.Errorf("[%d] got %s at line %d, want %s at line %d", i, vs[i].Rule, vs[i].Line, w.rule, w.line)
		}
	}
	// The hx-/x-/data-/aria- line (12) must be clean.
	for _, v := range vs {
		if v.Line == 12 {
			t.Errorf("prefixed attributes produced a violation: %+v", v)
		}
	}
	if !strings.Contains(vs[4].Message, "onclick") {
		t.Errorf("attr-unknown message %q should name onclick", vs[4].Message)
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
	for _, sub := range []string{"third_party", "node_modules", ".hidden"} {
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
