package diffview_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/diffview"
)

func render(t *testing.T, p diffview.Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := diffview.DiffView(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// rows splits rendered HTML into <tr>...</tr> chunks.
func rows(html string) []string {
	parts := strings.Split(html, "<tr")
	return parts[1:]
}

const twoFileDiff = `diff --git a/cmd/old.go b/cmd/new.go
similarity index 85%
rename from cmd/old.go
rename to cmd/new.go
index 1111111..2222222 100644
--- a/cmd/old.go
+++ b/cmd/new.go
@@ -1,4 +1,4 @@ package cmd
 package cmd

-func old() {}
+func renamed() {}
 // tail
diff --git a/README.md b/README.md
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/README.md
@@ -0,0 +1,2 @@
+# Title
+<script>alert(1)</script>
\ No newline at end of file
`

func TestParseTwoFilesWithRename(t *testing.T) {
	files, err := diffview.Parse(twoFileDiff)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}

	f0 := files[0]
	if f0.Status != diffview.StatusRenamed || f0.OldPath != "cmd/old.go" || f0.Path != "cmd/new.go" {
		t.Errorf("file 0: got status %q old %q new %q", f0.Status, f0.OldPath, f0.Path)
	}
	if len(f0.Hunks) != 1 || f0.Hunks[0].Header != "@@ -1,4 +1,4 @@ package cmd" {
		t.Fatalf("file 0 hunks: %+v", f0.Hunks)
	}
	want := []diffview.Line{
		{Kind: diffview.KindContext, OldNo: 1, NewNo: 1, Text: "package cmd"},
		{Kind: diffview.KindContext, OldNo: 2, NewNo: 2, Text: ""},
		{Kind: diffview.KindDel, OldNo: 3, Text: "func old() {}"},
		{Kind: diffview.KindAdd, NewNo: 3, Text: "func renamed() {}"},
		{Kind: diffview.KindContext, OldNo: 4, NewNo: 4, Text: "// tail"},
	}
	got := f0.Hunks[0].Lines
	if len(got) != len(want) {
		t.Fatalf("file 0 lines: got %d want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %+v want %+v", i, got[i], want[i])
		}
	}

	f1 := files[1]
	if f1.Status != diffview.StatusAdded || f1.Path != "README.md" || f1.OldPath != "" {
		t.Errorf("file 1: got status %q old %q new %q", f1.Status, f1.OldPath, f1.Path)
	}
	if len(f1.Hunks) != 1 || len(f1.Hunks[0].Lines) != 2 {
		t.Fatalf("file 1 hunks: %+v", f1.Hunks)
	}
	if l := f1.Hunks[0].Lines[1]; l.Kind != diffview.KindAdd || l.NewNo != 2 || l.Text != "<script>alert(1)</script>" {
		t.Errorf("file 1 line 1: %+v", l)
	}
}

func TestParsePlainUnifiedAndDeleted(t *testing.T) {
	src := "--- a/gone.txt\t2026-10-31\n+++ /dev/null\n@@ -1 +0,0 @@\n-bye\n--- a/x.txt\n+++ b/x.txt\n@@ -1 +1 @@\n-a\n+b\n"
	files, err := diffview.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}
	if files[0].Status != diffview.StatusDeleted || files[0].Path != "gone.txt" {
		t.Errorf("file 0: %+v", files[0])
	}
	if files[1].Status != diffview.StatusModified || files[1].Path != "x.txt" || files[1].OldPath != "" {
		t.Errorf("file 1: %+v", files[1])
	}
	if l := files[1].Hunks[0].Lines; len(l) != 2 || l[0].OldNo != 1 || l[1].NewNo != 1 {
		t.Errorf("file 1 lines: %+v", l)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := diffview.Parse("@@ -1 +1 @@\n-a\n+b\n"); err == nil {
		t.Errorf("hunk before any file should fail")
	}
	if _, err := diffview.Parse("--- a/x\n+++ b/x\n@@ nonsense @@\n"); err == nil {
		t.Errorf("malformed hunk header should fail")
	}
	files, err := diffview.Parse("")
	if err != nil || len(files) != 0 {
		t.Errorf("empty input: %v %+v", err, files)
	}
}

func TestInlineRowKindsAndNumbers(t *testing.T) {
	files, err := diffview.Parse(twoFileDiff)
	if err != nil {
		t.Fatal(err)
	}
	html := render(t, diffview.Props{ID: "d", Files: files})
	if !strings.Contains(html, `data-mode="inline"`) {
		t.Errorf("default mode is inline")
	}
	if strings.Contains(html, `data-layout="side-by-side"`) {
		t.Errorf("inline mode must not render the side-by-side table")
	}
	if strings.Contains(html, "md:hidden") {
		t.Errorf("inline mode must not hide its table at md")
	}
	for _, want := range []string{
		`data-status="renamed"`, ">Renamed<", ">cmd/old.go<", "cmd/new.go", ">renamed to<",
		`data-status="added"`, ">Added<", ">README.md<",
		`id="d-file-0"`, `id="d-file-1"`,
		">@@ -1,4 +1,4 @@ package cmd<",
		"overflow-x-auto", "whitespace-pre", "w-8",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
	rs := rows(html)
	if len(rs) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rs))
	}
	del := rs[2]
	if !strings.Contains(del, `data-kind="del"`) || !strings.Contains(del, "bg-destructive/10") || !strings.Contains(del, ">-<") {
		t.Errorf("deletion row: %s", del)
	}
	if !strings.Contains(del, ">3</td>") || strings.Contains(del, ">func old() {}</td>") == false {
		t.Errorf("deletion row numbers/text: %s", del)
	}
	// Deletion has no new number: two gutter cells, the second empty.
	if strings.Count(del, `text-muted-foreground"></td>`) != 1 {
		t.Errorf("deletion should have exactly one empty gutter: %s", del)
	}
	add := rs[3]
	if !strings.Contains(add, `data-kind="add"`) || !strings.Contains(add, "bg-success/10") || !strings.Contains(add, ">+<") || !strings.Contains(add, ">3</td>") {
		t.Errorf("addition row: %s", add)
	}
	ctx := rs[0]
	if !strings.Contains(ctx, `data-kind="context"`) || strings.Contains(ctx, "bg-success") || strings.Contains(ctx, "bg-destructive") {
		t.Errorf("context row: %s", ctx)
	}
	if strings.Count(ctx, ">1</td>") != 2 {
		t.Errorf("context row shows both numbers: %s", ctx)
	}
}

func TestEscapesScriptLine(t *testing.T) {
	files, err := diffview.Parse(twoFileDiff)
	if err != nil {
		t.Fatal(err)
	}
	html := render(t, diffview.Props{Files: files, Mode: diffview.ModeSideBySide})
	if strings.Contains(html, "<script>") {
		t.Fatalf("script tag leaked unescaped:\n%s", html)
	}
	if strings.Count(html, "&lt;script&gt;alert(1)&lt;/script&gt;") != 2 {
		t.Errorf("expected the escaped line once per rendering (inline + side-by-side):\n%s", html)
	}
}

func TestSideBySidePairing(t *testing.T) {
	f := diffview.File{Path: "p.go", Hunks: []diffview.Hunk{{Header: "@@ -1,4 +1,4 @@", Lines: []diffview.Line{
		{Kind: diffview.KindContext, OldNo: 1, NewNo: 1, Text: "same"},
		{Kind: diffview.KindDel, OldNo: 2, Text: "gone-a"},
		{Kind: diffview.KindDel, OldNo: 3, Text: "gone-b"},
		{Kind: diffview.KindAdd, NewNo: 2, Text: "new-a"},
		{Kind: diffview.KindContext, OldNo: 4, NewNo: 3, Text: "tail"},
		{Kind: diffview.KindAdd, NewNo: 4, Text: "appended"},
	}}}}
	html := render(t, diffview.Props{Files: []diffview.File{f}, Mode: diffview.ModeSideBySide})
	if !strings.Contains(html, `data-mode="side-by-side"`) {
		t.Errorf("mode attribute")
	}
	// Both renderings are present: inline hidden from md up, side-by-side shown from md up.
	if !strings.Contains(html, `class="overflow-x-auto md:hidden"`) || !strings.Contains(html, `class="hidden overflow-x-auto md:block"`) {
		t.Errorf("expected the two breakpoint-gated renderings in:\n%s", html)
	}
	side := html[strings.Index(html, `data-layout="side-by-side"`):]
	rs := rows(side)
	if len(rs) != 5 {
		t.Fatalf("expected 5 paired rows, got %d:\n%s", len(rs), side)
	}
	type want struct{ old, new string }
	wants := []want{
		{">same</td>", ">same</td>"},
		{">gone-a</td>", ">new-a</td>"},
		{">gone-b</td>", `data-side="new"></td>`},
		{">tail</td>", ">tail</td>"},
		{`data-side="old"></td>`, ">appended</td>"},
	}
	for i, w := range wants {
		if !strings.Contains(rs[i], w.old) || !strings.Contains(rs[i], w.new) {
			t.Errorf("row %d: want old %q new %q in:\n%s", i, w.old, w.new, rs[i])
		}
	}
	if !strings.Contains(rs[1], "bg-destructive/10") || !strings.Contains(rs[1], "bg-success/10") {
		t.Errorf("paired change row tints both sides: %s", rs[1])
	}
	if !strings.Contains(rs[2], "bg-muted/40") {
		t.Errorf("unpaired deletion leaves a muted gap: %s", rs[2])
	}
	// Numbers sit with their side.
	if !strings.Contains(rs[1], ">2</td>") || !strings.Contains(rs[4], ">4</td>") {
		t.Errorf("gutter numbers: %s / %s", rs[1], rs[4])
	}
}

func TestCollapsed(t *testing.T) {
	files := []diffview.File{
		{Path: "a", Hunks: []diffview.Hunk{{Header: "@@ -1 +1 @@", Lines: []diffview.Line{{Kind: diffview.KindAdd, NewNo: 1, Text: "x"}}}}},
		{Path: "b", Collapsed: true},
	}
	html := render(t, diffview.Props{Files: files})
	if strings.Count(html, "<details") != 2 {
		t.Fatalf("expected two file sections:\n%s", html)
	}
	if strings.Count(html, ` open>`) != 1 {
		t.Errorf("only the first file should be open:\n%s", html)
	}
	all := render(t, diffview.Props{Files: files, Collapsed: true})
	if strings.Contains(all, ` open>`) {
		t.Errorf("Collapsed closes every file:\n%s", all)
	}
}

func TestZeroValue(t *testing.T) {
	html := render(t, diffview.Props{})
	if !strings.HasPrefix(html, "<div") || strings.Contains(html, "<details") {
		t.Errorf("zero value renders an empty root:\n%s", html)
	}
}
