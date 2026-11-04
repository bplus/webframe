// Package examples holds the tutorial programs, one directory per step.
// The test here keeps each tutorial page in docs/tutorial showing exactly the
// code in its example.
package examples

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var fileBlock = regexp.MustCompile("(?s)<!-- file: (\\S+) -->\\n```go\\n(.*?)```")

func TestTutorialCodeMatchesExamples(t *testing.T) {
	pages, err := filepath.Glob("../docs/tutorial/*.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no tutorial pages found in docs/tutorial")
	}
	for _, page := range pages {
		if filepath.Base(page) == "README.md" {
			continue // the index, not a step
		}
		md, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		blocks := fileBlock.FindAllSubmatch(md, -1)
		if len(blocks) == 0 {
			t.Errorf("%s shows no example file (want a <!-- file: path --> marker before a go block)", page)
		}
		for _, b := range blocks {
			want, err := os.ReadFile(filepath.Join("..", string(b[1])))
			if err != nil {
				t.Errorf("%s: %v", page, err)
				continue
			}
			if string(b[2]) != string(want) {
				t.Errorf("%s: the code shown for %s differs from the file; paste the file into the page again", page, b[1])
			}
		}
	}
}
