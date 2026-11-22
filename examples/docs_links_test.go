package examples

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// lineLink matches a relative Markdown link to source lines: (path#L12) or
// (path#L12-L30). Links with a scheme (http:, https:) never match, because
// the path may not contain a colon.
var lineLink = regexp.MustCompile(`\]\(([^():#\s]+)#L(\d+)(?:-L(\d+))?\)`)

// namedLink finds the code span that names what a line link points at. The
// docs write such links in two ways:
//
//	`Name` ([file.go:12](...))     the span right before the link, joined by
//	`Name` is at [file.go:12](...) " (", " is at ", " are at ", " in " or ", "
//	[`Name`, file.go:12](...)      the span inside the link text
//
// The match must end where the link's "[" starts, so a span earlier in the
// sentence never counts.
var namedLink = regexp.MustCompile("(?:`([^`]+)`(?:\\s*\\(|\\s+(?:is|are) at\\s+|\\s+in\\s+|,\\s*)\\[|\\[`([^`]+)`)$")

// goName is a Go identifier, optionally qualified (App.Middleware,
// liststate.WithSort) and optionally followed by call arguments (Mount(mux)).
var goName = regexp.MustCompile(`^[A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*(?:\(.*\))?$`)

// linkedName returns the last component of the Go identifier that names the
// link starting at md[start] (the "[" of the link), or "" when none does.
func linkedName(md []byte, start int) string {
	head := md[:start+1]
	if bytes.HasPrefix(md[start:], []byte("[`")) {
		if end := bytes.IndexByte(md[start+2:], '`'); end >= 0 {
			head = md[:start+2+end+1]
		}
	}
	from := max(0, len(head)-200)
	m := namedLink.FindSubmatch(head[from:])
	if m == nil {
		return ""
	}
	name := string(m[1]) + string(m[2])
	if !goName.MatchString(name) {
		return ""
	}
	if i := strings.IndexByte(name, '('); i >= 0 {
		name = name[:i]
	}
	return name[strings.LastIndexByte(name, '.')+1:]
}

// TestDocsLineLinks checks every link to exact source lines in docs/*.md
// and docs/tutorial/*.md: the file must exist and the line range must lie
// inside it. When the link is named by a Go identifier (see namedLink), the
// identifier's last part must also appear as a whole word inside the linked
// lines. An edit above a linked line moves the code under the link; this
// test is what notices.
func TestDocsLineLinks(t *testing.T) {
	var pages []string
	for _, pattern := range []string{"../docs/*.md", "../docs/tutorial/*.md"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, m...)
	}
	if len(pages) == 0 {
		t.Fatal("no Markdown pages found in docs")
	}
	checked, named := 0, 0
	for _, page := range pages {
		md, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, idx := range lineLink.FindAllSubmatchIndex(md, -1) {
			checked++
			target := string(md[idx[2]:idx[3]])
			first, _ := strconv.Atoi(string(md[idx[4]:idx[5]]))
			last := first
			if idx[6] >= 0 {
				last, _ = strconv.Atoi(string(md[idx[6]:idx[7]]))
			}
			name := linkedName(md, bytes.LastIndexByte(md[:idx[0]], '['))
			docLine := bytes.Count(md[:idx[0]], []byte("\n")) + 1
			src, err := os.ReadFile(filepath.Join(filepath.Dir(page), target))
			if err != nil {
				t.Errorf("%s: link to %s#L%d: %v", page, target, first, err)
				continue
			}
			lines := bytes.Count(src, []byte("\n"))
			if len(src) > 0 && src[len(src)-1] != '\n' {
				lines++
			}
			if first < 1 || last < first || last > lines {
				t.Errorf("%s:%d: link to %s#L%d-L%d is outside the file (%d lines)", page, docLine, target, first, last, lines)
				continue
			}
			if name == "" {
				continue
			}
			named++
			if msg := checkName(target, src, name, first, last); msg != "" {
				t.Errorf("%s:%d: link to %s#L%d-L%d is named %q, but %s", page, docLine, target, first, last, name, msg)
			}
		}
	}
	if checked == 0 {
		t.Log("no line links found in docs")
	}
	t.Logf("%d line links checked, %d of them named by a Go identifier", checked, named)
}

// checkName reports, as a phrase, why lines first..last of src do not show
// name, or "" when they do. The name must appear as a whole word on a line
// that is not a // comment, so a doc comment that mentions it is not
// enough. In a Go file that declares name at top level (a func, method,
// type, var or const), the range must also hold the first line of one of
// those declarations, and must be that one line or reach the declaration's
// last line, so a function that grew longer under its link is noticed too.
func checkName(target string, src []byte, name string, first, last int) string {
	word := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	lines := bytes.Split(src, []byte("\n"))[first-1 : last]
	found := false
	for _, l := range lines {
		if !bytes.HasPrefix(bytes.TrimSpace(l), []byte("//")) && word.Match(l) {
			found = true
			break
		}
	}
	if !found {
		return "no code line in that range contains it"
	}
	if filepath.Ext(target) != ".go" {
		return ""
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, target, src, 0)
	if err != nil {
		return "the file does not parse: " + err.Error()
	}
	decls := declarations(f, name)
	var where []string
	for _, d := range decls {
		start, end := fset.Position(d.Pos()).Line, fset.Position(d.End()).Line
		where = append(where, strconv.Itoa(start)+"-"+strconv.Itoa(end))
		if start < first || start > last {
			continue
		}
		if first != last && end > last {
			return "its declaration runs from line " + strconv.Itoa(start) + " to " + strconv.Itoa(end)
		}
		return ""
	}
	if len(decls) > 0 {
		return "it is declared at lines " + strings.Join(where, ", ")
	}
	return ""
}

// declarations returns the top-level declarations of name in f. A var or
// const counts as its one spec, a func or method as the whole function.
func declarations(f *ast.File, name string) []ast.Node {
	var out []ast.Node
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == name {
				out = append(out, d)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if spec.Name.Name == name {
						out = append(out, spec)
					}
				case *ast.ValueSpec:
					for _, n := range spec.Names {
						if n.Name == name {
							out = append(out, spec)
						}
					}
				}
			}
		}
	}
	return out
}
