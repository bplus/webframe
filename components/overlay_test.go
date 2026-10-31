package components_test

import (
	"bytes"
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/dialog"
	"github.com/bplus/webframe/components/slideover"
)

// The overlay ladder (batch 3, kept by batch 4's static test): a slideover
// backdrop at 55 under its panel at 60, a dialog (backdrop and content) at
// 65 so a confirm dialog opened from inside a sheet is on top of it, and
// the toast container at 70 so a toast is never covered by any of them.
// The slideover and dialog values are Tailwind z-[N] classes in the
// rendered markup; the toast value is a z-index rule on
// .wf-toast-container in static/css/input.css.

var zClass = regexp.MustCompile(`\bz-\[(\d+)\]`)

// renderHTML renders c with no children.
func renderHTML(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// openTag returns the opening tag (from "<div" to ">") of the element whose
// attributes contain marker, which must occur exactly once in html.
func openTag(t *testing.T, html, marker string) string {
	t.Helper()
	if strings.Count(html, marker) != 1 {
		t.Fatalf("marker %q occurs %d times in:\n%s", marker, strings.Count(html, marker), html)
	}
	at := strings.Index(html, marker)
	start := strings.LastIndex(html[:at], "<div")
	if start < 0 {
		t.Fatalf("no <div before %q in:\n%s", marker, html)
	}
	end := strings.Index(html[at:], ">")
	if end < 0 {
		t.Fatalf("unterminated tag for %q in:\n%s", marker, html)
	}
	return html[start : at+end+1]
}

// zIndexClass returns the one z-[N] value in tag.
func zIndexClass(t *testing.T, tag string) int {
	t.Helper()
	m := zClass.FindAllStringSubmatch(tag, -1)
	if len(m) != 1 {
		t.Fatalf("want exactly one z-[N] class, got %d in %s", len(m), tag)
	}
	n, err := strconv.Atoi(m[0][1])
	if err != nil {
		t.Fatalf("z-[N] parse: %v", err)
	}
	return n
}

// toastZIndex reads the z-index of the first .wf-toast-container rule in
// static/css/input.css (the base rule; the phone media query override sets
// no z-index).
func toastZIndex(t *testing.T) int {
	t.Helper()
	css, err := os.ReadFile("../static/css/input.css")
	if err != nil {
		t.Fatalf("read input.css: %v", err)
	}
	src := string(css)
	at := strings.Index(src, ".wf-toast-container {")
	if at < 0 {
		t.Fatalf(".wf-toast-container rule not found in input.css")
	}
	block := src[at:]
	if end := strings.Index(block, "}"); end >= 0 {
		block = block[:end]
	}
	m := regexp.MustCompile(`z-index:\s*(\d+)\s*;`).FindStringSubmatch(block)
	if m == nil {
		t.Fatalf("no z-index in .wf-toast-container rule:\n%s", block)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("z-index parse: %v", err)
	}
	return n
}

func TestOverlayLadder(t *testing.T) {
	sheet := renderHTML(t, slideover.Panel(slideover.PanelProps{ID: "sheet", Title: "Sheet"}))
	sheetBackdrop := zIndexClass(t, openTag(t, sheet, "data-tui-slideover-backdrop"))
	sheetPanel := zIndexClass(t, openTag(t, sheet, "data-tui-slideover-panel"))

	dlg := renderHTML(t, dialog.Content(dialog.ContentProps{ID: "dlg"}))
	dialogBackdrop := zIndexClass(t, openTag(t, dlg, "data-tui-dialog-backdrop"))
	dialogContent := zIndexClass(t, openTag(t, dlg, "data-tui-dialog-content"))

	toast := toastZIndex(t)

	// Exact values: the ladder is documented as 55 / 60 / 65 / 70.
	want := map[string][2]int{
		"slideover backdrop": {sheetBackdrop, 55},
		"slideover panel":    {sheetPanel, 60},
		"dialog backdrop":    {dialogBackdrop, 65},
		"dialog content":     {dialogContent, 65},
		"toast container":    {toast, 70},
	}
	for name, v := range want {
		if v[0] != v[1] {
			t.Errorf("%s z-index = %d, want %d", name, v[0], v[1])
		}
	}

	// Order: backdrop < panel < dialog < toast.
	if !(sheetBackdrop < sheetPanel) {
		t.Errorf("slideover backdrop (%d) must be below its panel (%d)", sheetBackdrop, sheetPanel)
	}
	if !(sheetPanel < dialogBackdrop && sheetPanel < dialogContent) {
		t.Errorf("slideover panel (%d) must be below the dialog (%d/%d)", sheetPanel, dialogBackdrop, dialogContent)
	}
	if !(dialogBackdrop <= dialogContent) {
		t.Errorf("dialog backdrop (%d) must not be above its content (%d)", dialogBackdrop, dialogContent)
	}
	if !(dialogContent < toast) {
		t.Errorf("dialog (%d) must be below the toast container (%d)", dialogContent, toast)
	}
}
