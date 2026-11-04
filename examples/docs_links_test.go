package examples

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// lineLink matches a relative Markdown link to source lines: (path#L12) or
// (path#L12-L30). Links with a scheme (http:, https:) never match, because
// the path may not contain a colon.
var lineLink = regexp.MustCompile(`\]\(([^():#\s]+)#L(\d+)(?:-L(\d+))?\)`)

// TestDocsLineLinks checks every link to exact source lines in docs/*.md
// and docs/tutorial/*.md: the file must exist and the line range must lie
// inside it. An edit above a linked line moves the code under the link;
// this test is what notices.
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
	checked := 0
	for _, page := range pages {
		md, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range lineLink.FindAllSubmatch(md, -1) {
			checked++
			target := string(m[1])
			first, _ := strconv.Atoi(string(m[2]))
			last := first
			if len(m[3]) > 0 {
				last, _ = strconv.Atoi(string(m[3]))
			}
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
				t.Errorf("%s: link to %s#L%d-L%d is outside the file (%d lines)", page, target, first, last, lines)
			}
		}
	}
	if checked == 0 {
		t.Log("no line links found in docs")
	}
}
