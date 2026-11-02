package components_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/aspectratio"
	"github.com/bplus/webframe/components/avatar"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/components/card"
	"github.com/bplus/webframe/components/customblock"
	"github.com/bplus/webframe/components/custompage"
	"github.com/bplus/webframe/components/dashboardgrid"
	"github.com/bplus/webframe/components/detailpage"
	"github.com/bplus/webframe/components/dialog"
	"github.com/bplus/webframe/components/dropdown"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/figure"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/hovercard"
	"github.com/bplus/webframe/components/label"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/panel"
	"github.com/bplus/webframe/components/popover"
	"github.com/bplus/webframe/components/selectbox"
	"github.com/bplus/webframe/components/separator"
	"github.com/bplus/webframe/components/slideover"
	"github.com/bplus/webframe/components/table"
	"github.com/bplus/webframe/components/tabs"
	"github.com/bplus/webframe/components/toggle"
	"github.com/bplus/webframe/components/toolbar"
	"github.com/bplus/webframe/compose"
)

// childCases has one row per component package whose templates render
// { children... }: a container from that package, filled from plain Go.
var childCases = map[string]func() templ.Component{
	"appshell":      func() templ.Component { return appshell.Shell() },
	"aspectratio":   func() templ.Component { return aspectratio.AspectRatio() },
	"avatar":        func() templ.Component { return avatar.Avatar() },
	"badge":         func() templ.Component { return badge.Badge() },
	"button":        func() templ.Component { return button.Button() },
	"card":          func() templ.Component { return card.Card() },
	"customblock":   func() templ.Component { return customblock.CustomBlock() },
	"custompage":    func() templ.Component { return custompage.CustomPage() },
	"dashboardgrid": func() templ.Component { return dashboardgrid.Grid() },
	"detailpage":    func() templ.Component { return detailpage.Page(detailpage.Props{}) },
	"dialog":        func() templ.Component { return dialog.Dialog() },
	// Dropdown itself renders only its children; Trigger wraps them.
	"dropdown":   func() templ.Component { return dropdown.Trigger() },
	"emptystate": func() templ.Component { return emptystate.EmptyState() },
	"figure":     func() templ.Component { return figure.Figure() },
	"form":       func() templ.Component { return form.Form() },
	"hovercard":  func() templ.Component { return hovercard.Hover(hovercard.Props{ID: "h"}) },
	"label":      func() templ.Component { return label.Label() },
	"layout":     func() templ.Component { return layout.Stack() },
	"panel":      func() templ.Component { return panel.Panel() },
	"popover":    func() templ.Component { return popover.Trigger() },
	"selectbox":  func() templ.Component { return selectbox.SelectBox() },
	"separator":  func() templ.Component { return separator.Separator() },
	"slideover":  func() templ.Component { return slideover.SlideOver() },
	"table":      func() templ.Component { return table.Table() },
	"tabs":       func() templ.Component { return tabs.Tabs() },
	"toggle":     func() templ.Component { return toggle.Toggle() },
	"toolbar":    func() templ.Component { return toolbar.Toolbar() },
}

var childrenRe = regexp.MustCompile(`\{\s*children\.\.\.\s*\}`)

// TestChildCasesCoverAllContainers fails when a component package renders
// { children... } but has no row in childCases, or a row names a package
// that no longer does.
func TestChildCasesCoverAllContainers(t *testing.T) {
	files, err := filepath.Glob("*/*.templ")
	if err != nil {
		t.Fatal(err)
	}
	containers := map[string]bool{}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if childrenRe.Match(src) {
			containers[filepath.Dir(f)] = true
		}
	}
	if len(containers) == 0 {
		t.Fatal("found no templates rendering { children... }; is the test running in components/?")
	}
	for pkg := range containers {
		if _, ok := childCases[pkg]; !ok {
			t.Errorf("components/%s renders { children... } but has no row in childCases", pkg)
		}
	}
	for pkg := range childCases {
		if !containers[pkg] {
			t.Errorf("childCases has a row for %s, which renders no { children... }", pkg)
		}
	}
}

// TestComposeWithFillsEveryContainer renders every container from plain
// Go with compose.With and checks the child text comes out inside it,
// escaped.
func TestComposeWithFillsEveryContainer(t *testing.T) {
	const marker = "MARKER<b>"
	const escaped = "MARKER&lt;b&gt;"
	for pkg, ctor := range childCases {
		t.Run(pkg, func(t *testing.T) {
			var buf bytes.Buffer
			if err := compose.With(ctor(), compose.Text(marker)).Render(context.Background(), &buf); err != nil {
				t.Fatalf("Render: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, escaped) {
				t.Fatalf("output does not contain %q:\n%s", escaped, out)
			}
			if strings.Contains(out, marker) {
				t.Fatalf("output contains the unescaped marker %q:\n%s", marker, out)
			}
			if strings.Count(out, escaped) != 1 {
				t.Fatalf("marker rendered %d times, want 1:\n%s", strings.Count(out, escaped), out)
			}
			if i := strings.Index(out, escaped); i == 0 || i+len(escaped) == len(out) {
				t.Fatalf("marker is not inside the container's markup:\n%s", out)
			}
		})
	}
}
