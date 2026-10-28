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
// "page" or "block". Populated by the fence handling added in a later task.
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
var skipDirs = map[string]bool{"third_party": true, "node_modules": true}

// Lint checks every path. Directories are walked recursively for *.templ
// files (skipping third_party, node_modules and dot-directories); files are
// taken as given, whatever their extension. The error is non-nil only when a
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
		info, err := os.Stat(p)
		if err != nil {
			return Report{}, fmt.Errorf("wflint: %w", err)
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
			if strings.HasSuffix(d.Name(), ".templ") {
				add(path)
			}
			return nil
		})
		if err != nil {
			return Report{}, fmt.Errorf("wflint: walking %s: %w", p, err)
		}
	}

	var rep Report
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return Report{}, fmt.Errorf("wflint: %w", err)
		}
		rep.Violations = append(rep.Violations, lintSource(f, string(src))...)
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
	return rep, nil
}

// lintSource parses one templ source and returns its violations.
func lintSource(file, src string) []Violation {
	tf, err := parser.ParseString(src)
	if err != nil {
		return []Violation{{
			File:    file,
			Line:    1,
			Col:     1,
			Rule:    "parse-error",
			Message: err.Error(),
		}}
	}
	w := &walker{file: file}
	for _, n := range tf.Nodes {
		if t, ok := n.(*parser.HTMLTemplate); ok {
			w.nodes(t.Children, false)
		}
	}
	return w.out
}
