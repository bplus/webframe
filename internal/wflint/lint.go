// Package wflint enforces the webframe component boundary on app code: app
// templates may contain only text-level HTML elements and calls to webframe
// components, never block-level elements, class or style attributes, or
// <style> blocks. It parses templ files with the templ parser and walks the
// AST; it never compiles them.
package wflint

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/a-h/templ/parser/v2"
)

// Violation is one rule breach at a 1-based line and column of a file.
type Violation struct {
	File    string
	Line    int
	Col     int
	Rule    string
	Message string
	Hint    string
}

// Fence records a sanctioned escape hatch (CustomPage / CustomBlock). Kind is
// "page" or "block". Reason is "" when the fence has none (which is also a
// fence-reason violation).
type Fence struct {
	File   string
	Line   int
	Kind   string
	Reason string
}

// Report is the result of a Lint run.
type Report struct {
	Violations []Violation
	Fences     []Fence
}

// skipDirs are directory names never descended into during a walk.
var skipDirs = map[string]bool{"third_party": true, "node_modules": true, "vendor": true}

// trimEllipsis turns a Go-style package pattern ("./...", "demo/...", "...")
// into its directory; walks are already recursive.
func trimEllipsis(p string) string {
	if p != "..." && !strings.HasSuffix(p, "/...") {
		return p
	}
	if d := strings.TrimSuffix(strings.TrimSuffix(p, "..."), "/"); d != "" {
		return d
	}
	return "."
}

// Lint checks every path. A path ending in "/..." means its directory. Directories are walked recursively for *.templ,
// *.go and *.css files (skipping third_party, node_modules, vendor
// and dot-directories); files are taken as given, whatever their extension. A
// *.go file gets the Go-side rules and is skipped when it is templ output
// (named *_templ.go with the templ header); a *.css file is itself a css-file violation; anything else is parsed
// as templ. The error is non-nil only when a
// path does not exist or cannot be read; a file that fails to parse becomes a
// parse-error violation. Violations are sorted by file, line, then column.
func Lint(paths []string) (Report, error) {
	var files []string
	seen := map[string]bool{}
	add := func(p string) {
		p = filepath.Clean(p)
		if !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	for _, p := range paths {
		p = trimEllipsis(p)
		info, err := os.Stat(p)
		if err != nil {
			return Report{}, err
		}
		if !info.IsDir() {
			add(p)
			continue
		}
		root := filepath.Clean(p)
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			if ext := filepath.Ext(d.Name()); ext == ".templ" || ext == ".go" || ext == ".css" {
				add(path)
			}
			return nil
		})
		if err != nil {
			return Report{}, fmt.Errorf("walking %s: %w", p, err)
		}
	}

	var rep Report
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return Report{}, err
		}
		switch filepath.Ext(f) {
		case ".go":
			rep.Violations = append(rep.Violations, lintGo(f, string(src))...)
		case ".css":
			rep.Violations = append(rep.Violations, Violation{
				File: f, Line: 1, Col: 1, Rule: "css-file",
				Message: "CSS files are not allowed in app code; styling belongs inside webframe components",
			})
		default:
			vs, fs := lintSource(f, string(src))
			rep.Violations = append(rep.Violations, vs...)
			rep.Fences = append(rep.Fences, fs...)
		}
	}
	sort.SliceStable(rep.Violations, func(i, j int) bool {
		a, b := rep.Violations[i], rep.Violations[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
	sort.SliceStable(rep.Fences, func(i, j int) bool {
		a, b := rep.Fences[i], rep.Fences[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return rep, nil
}

// lintSource parses one templ source and returns its violations and fences.
func lintSource(file, src string) ([]Violation, []Fence) {
	tf, err := parser.ParseString(src)
	if err != nil {
		return []Violation{{
			File:    file,
			Line:    1,
			Col:     1,
			Rule:    "parse-error",
			Message: err.Error(),
		}}, nil
	}
	w := &walker{file: file}
	for _, h := range tf.Header {
		w.reportGo(h.Expression, false)
	}
	for _, n := range tf.Nodes {
		switch n := n.(type) {
		case *parser.HTMLTemplate:
			w.nodes(n.Children, false)
		case *parser.TemplateFileGoExpression:
			// The Go sections of a .templ file (imports, vars, funcs).
			w.reportGo(n.Expression, false)
		}
	}
	return w.out, w.fences
}
