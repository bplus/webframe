// Package components_test contains a cross-package render smoke test: every
// component package's primary constructor, called with zero-value Props,
// must render without panicking, without erroring, and without leaking
// template-generation artifacts (e.g. the templ.NopComponent text
// bug) into its HTML output.
package components_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/areachart"
	"github.com/bplus/webframe/components/aspectratio"
	"github.com/bplus/webframe/components/avatar"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/barchart"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/components/card"
	"github.com/bplus/webframe/components/checkbox"
	"github.com/bplus/webframe/components/copybutton"
	"github.com/bplus/webframe/components/dashboardgrid"
	"github.com/bplus/webframe/components/dashboardheader"
	"github.com/bplus/webframe/components/dialog"
	"github.com/bplus/webframe/components/dropdown"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/gameicon"
	"github.com/bplus/webframe/components/graphwidget"
	"github.com/bplus/webframe/components/hovercard"
	"github.com/bplus/webframe/components/icon"
	"github.com/bplus/webframe/components/input"
	"github.com/bplus/webframe/components/label"
	"github.com/bplus/webframe/components/lifecyclestepper"
	"github.com/bplus/webframe/components/metricring"
	"github.com/bplus/webframe/components/nodepill"
	"github.com/bplus/webframe/components/piechart"
	"github.com/bplus/webframe/components/popover"
	"github.com/bplus/webframe/components/progressbar"
	"github.com/bplus/webframe/components/radio"
	"github.com/bplus/webframe/components/selectbox"
	"github.com/bplus/webframe/components/separator"
	"github.com/bplus/webframe/components/slideover"
	"github.com/bplus/webframe/components/sparkline"
	"github.com/bplus/webframe/components/statcomparison"
	"github.com/bplus/webframe/components/table"
	"github.com/bplus/webframe/components/tabs"
	"github.com/bplus/webframe/components/textarea"
	"github.com/bplus/webframe/components/trafficstream"
)

// bannedSubstrings are markers of template-generation artifacts leaking into
// rendered HTML: a stray templ.NopComponent/templ.Component reference, a Go
// fmt error verb, or an unresolved templ `{{ }}` directive.
var bannedSubstrings = []string{"NopComponent", "templ.", "%!", "{{"}

// componentCase is one row of the render smoke test: a component package's
// primary constructor, called with zero-value Props.
type componentCase struct {
	// name is the components/ subdirectory this row covers.
	name string

	constructor func() templ.Component

	// expectEmpty documents a package whose zero-value Props legitimately
	// renders no output, so the non-empty-output assertion is skipped.
	expectEmpty bool
	// expectEmptyReason explains why, so this isn't a silent skip.
	expectEmptyReason string

	// knownIssue points at a tracked bug that makes this row's output
	// contain a banned substring today. While the issue is open the row is
	// skipped (not failed) with a pointer to it. Once the bug is fixed the
	// banned-substring check stops triggering and this case starts failing
	// loudly again (with a message to remove knownIssue) — the row "flips"
	// back to a real assertion instead of silently passing forever.
	knownIssue string
}

var componentCases = []componentCase{
	{name: "appshell", constructor: func() templ.Component { return appshell.Shell() }},
	{name: "areachart", constructor: func() templ.Component { return areachart.AreaChart() }},
	{name: "aspectratio", constructor: func() templ.Component { return aspectratio.AspectRatio() }},
	{name: "avatar", constructor: func() templ.Component { return avatar.Avatar() }},
	{name: "badge", constructor: func() templ.Component { return badge.Badge() }},
	{name: "barchart", constructor: func() templ.Component { return barchart.BarChart() }},
	{name: "button", constructor: func() templ.Component { return button.Button() }},
	{name: "card", constructor: func() templ.Component { return card.Card() }},
	{name: "checkbox", constructor: func() templ.Component { return checkbox.Checkbox() }},
	{name: "copybutton", constructor: func() templ.Component { return copybutton.CopyButton() }},
	{name: "dashboardgrid", constructor: func() templ.Component { return dashboardgrid.Grid() }},
	{name: "dashboardheader", constructor: func() templ.Component { return dashboardheader.Header() }},
	{name: "dialog", constructor: func() templ.Component { return dialog.Dialog() }},
	{
		name:              "dropdown",
		constructor:       func() templ.Component { return dropdown.Dropdown() },
		expectEmpty:       true,
		expectEmptyReason: "Dropdown() only stashes a context value and renders `{ children... }`; with no children supplied, templ.GetChildren falls back to templ.NopComponent (a real, silent empty component — not the leaked-text bug), so zero-value output is legitimately zero bytes.",
	},
	{name: "emptystate", constructor: func() templ.Component { return emptystate.EmptyState() }},
	{name: "filterbar", constructor: func() templ.Component { return filterbar.FilterBar() }},
	{name: "form", constructor: func() templ.Component { return form.Item() }},
	{name: "gameicon", constructor: func() templ.Component { return gameicon.GameIcon(gameicon.Props{}) }},
	{name: "graphwidget", constructor: func() templ.Component { return graphwidget.GraphWidget() }},
	{name: "hovercard", constructor: func() templ.Component { return hovercard.Hover(hovercard.Props{}) }},
	// icon.Icon(name) is a factory, not a Props-taking constructor; icon_defs.go
	// predefines one var per Lucide icon (e.g. AArrowDown = Icon("a-arrow-down")),
	// so calling one of those with zero Props exercises the same rendering path.
	{name: "icon", constructor: func() templ.Component { return icon.AArrowDown() }},
	{name: "input", constructor: func() templ.Component { return input.Input() }},
	{name: "label", constructor: func() templ.Component { return label.Label() }},
	{name: "lifecyclestepper", constructor: func() templ.Component { return lifecyclestepper.LifecycleStepper() }},
	{name: "metricring", constructor: func() templ.Component { return metricring.MetricRing() }},
	{name: "nodepill", constructor: func() templ.Component { return nodepill.NodePill() }},
	{name: "piechart", constructor: func() templ.Component { return piechart.PieChart() }},
	// popover has no top-level Popover(); Content is the substantive floating
	// panel and the best single representative of the package.
	{name: "popover", constructor: func() templ.Component { return popover.Content() }},
	{name: "progressbar", constructor: func() templ.Component { return progressbar.ProgressBar() }},
	{name: "action", constructor: func() templ.Component { return action.Button(action.Action{Label: "Go"}) }},
	{name: "radio", constructor: func() templ.Component { return radio.Radio() }},
	{name: "selectbox", constructor: func() templ.Component { return selectbox.SelectBox() }},
	{name: "separator", constructor: func() templ.Component { return separator.Separator() }},
	{name: "slideover", constructor: func() templ.Component { return slideover.SlideOver() }},
	{name: "sparkline", constructor: func() templ.Component { return sparkline.Sparkline() }},
	{name: "statcomparison", constructor: func() templ.Component { return statcomparison.StatComparison() }},
	{name: "table", constructor: func() templ.Component { return table.Table() }},
	// tabs.Content() exercises the empty-Value rendering path covered by the
	// package-level regression test.
	{name: "tabs", constructor: func() templ.Component { return tabs.Content() }},
	{name: "textarea", constructor: func() templ.Component { return textarea.Textarea() }},
	{name: "trafficstream", constructor: func() templ.Component { return trafficstream.TrafficStream() }},
}

// TestComponentCasesCoverAllPackages ensures every directory under
// components/ has exactly one row above, so a newly added component package
// can't be forgotten by this suite.
func TestComponentCasesCoverAllPackages(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading components directory: %v", err)
	}

	var dirCount int
	for _, e := range entries {
		if e.IsDir() {
			dirCount++
		}
	}

	if dirCount != len(componentCases) {
		t.Fatalf("components/ has %d subdirectories but componentCases has %d entries — a component package was added or removed without updating this table", dirCount, len(componentCases))
	}
}

func TestComponentRenderSmoke(t *testing.T) {
	for _, tc := range componentCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tc.constructor().Render(context.Background(), &buf); err != nil {
				t.Fatalf("Render returned error: %v", err)
			}
			out := buf.String()

			var bad string
			for _, s := range bannedSubstrings {
				if strings.Contains(out, s) {
					bad = s
					break
				}
			}

			if bad != "" {
				if tc.knownIssue != "" {
					t.Skipf("known issue (%s); output contains banned substring %q — remove the knownIssue annotation once fixed", tc.knownIssue, bad)
				}
				t.Fatalf("output contains banned substring %q:\n%s", bad, out)
			}
			if tc.knownIssue != "" {
				t.Fatalf("knownIssue %q set but output no longer contains a banned substring — the underlying bug appears fixed; remove the knownIssue annotation for %q", tc.knownIssue, tc.name)
			}

			if tc.expectEmpty {
				if out != "" {
					t.Fatalf("expectEmpty is set (%s) but output is non-empty (%d bytes) — remove expectEmpty", tc.expectEmptyReason, len(out))
				}
				return
			}
			if out == "" {
				t.Fatalf("output is empty; if this is expected for zero-value Props, set expectEmpty and document why")
			}
		})
	}
}
