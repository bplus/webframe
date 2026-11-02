// Package components_test contains a cross-package render smoke test: every
// component package's primary constructor, called with zero-value Props,
// must render without panicking, without erroring, and without leaking
// template-generation artifacts (such as the literal text of a
// templ.NopComponent) into its HTML output.
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
	"github.com/bplus/webframe/components/authpage"
	"github.com/bplus/webframe/components/autocomplete"
	"github.com/bplus/webframe/components/avatar"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/banner"
	"github.com/bplus/webframe/components/barchart"
	"github.com/bplus/webframe/components/breadcrumbs"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/components/card"
	"github.com/bplus/webframe/components/cardgrid"
	"github.com/bplus/webframe/components/checkbox"
	"github.com/bplus/webframe/components/codeblock"
	"github.com/bplus/webframe/components/commandpalette"
	"github.com/bplus/webframe/components/confirmdialog"
	"github.com/bplus/webframe/components/copybutton"
	"github.com/bplus/webframe/components/customblock"
	"github.com/bplus/webframe/components/custompage"
	"github.com/bplus/webframe/components/dashboardgrid"
	"github.com/bplus/webframe/components/dashboardheader"
	"github.com/bplus/webframe/components/dashboardpage"
	"github.com/bplus/webframe/components/datatable"
	"github.com/bplus/webframe/components/datetimepicker"
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/detailpage"
	"github.com/bplus/webframe/components/dialog"
	"github.com/bplus/webframe/components/diffview"
	"github.com/bplus/webframe/components/dropdown"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/errorpage"
	"github.com/bplus/webframe/components/figure"
	"github.com/bplus/webframe/components/filedrop"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/flash"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/formpage"
	"github.com/bplus/webframe/components/graphwidget"
	"github.com/bplus/webframe/components/hovercard"
	"github.com/bplus/webframe/components/icon"
	"github.com/bplus/webframe/components/image"
	"github.com/bplus/webframe/components/input"
	"github.com/bplus/webframe/components/jsonviewer"
	"github.com/bplus/webframe/components/keyvalueeditor"
	"github.com/bplus/webframe/components/label"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/lifecyclestepper"
	"github.com/bplus/webframe/components/listpage"
	"github.com/bplus/webframe/components/logviewer"
	"github.com/bplus/webframe/components/metricring"
	"github.com/bplus/webframe/components/nodepill"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/pagination"
	"github.com/bplus/webframe/components/panel"
	"github.com/bplus/webframe/components/piechart"
	"github.com/bplus/webframe/components/popover"
	"github.com/bplus/webframe/components/progressbar"
	"github.com/bplus/webframe/components/prose"
	"github.com/bplus/webframe/components/radio"
	"github.com/bplus/webframe/components/rating"
	"github.com/bplus/webframe/components/selectbox"
	"github.com/bplus/webframe/components/separator"
	"github.com/bplus/webframe/components/settingspage"
	"github.com/bplus/webframe/components/skeleton"
	"github.com/bplus/webframe/components/slideover"
	"github.com/bplus/webframe/components/sparkline"
	"github.com/bplus/webframe/components/spinner"
	"github.com/bplus/webframe/components/splitpane"
	"github.com/bplus/webframe/components/statcomparison"
	"github.com/bplus/webframe/components/stattile"
	"github.com/bplus/webframe/components/swatch"
	"github.com/bplus/webframe/components/table"
	"github.com/bplus/webframe/components/tabs"
	"github.com/bplus/webframe/components/tagsinput"
	"github.com/bplus/webframe/components/textarea"
	"github.com/bplus/webframe/components/timeline"
	"github.com/bplus/webframe/components/toast"
	"github.com/bplus/webframe/components/toggle"
	"github.com/bplus/webframe/components/toolbar"
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

	// knownIssue names a tracked bug that makes this row's output contain a
	// banned substring. The row is skipped while the bug exists, and fails
	// (asking for knownIssue to be removed) once the output is clean.
	knownIssue string
}

var componentCases = []componentCase{
	{name: "appshell", constructor: func() templ.Component { return appshell.Shell() }},
	{name: "areachart", constructor: func() templ.Component { return areachart.AreaChart() }},
	{name: "aspectratio", constructor: func() templ.Component { return aspectratio.AspectRatio() }},
	{name: "authpage", constructor: func() templ.Component { return authpage.AuthPage(authpage.Props{}) }},
	{name: "autocomplete", constructor: func() templ.Component { return autocomplete.Autocomplete(autocomplete.Props{ID: "x", HxGet: "/q"}) }},
	{name: "avatar", constructor: func() templ.Component { return avatar.Avatar() }},
	{name: "badge", constructor: func() templ.Component { return badge.Badge() }},
	{
		name:              "banner",
		constructor:       func() templ.Component { return banner.Banner() },
		expectEmpty:       true,
		expectEmptyReason: "a banner with no title, body or details renders nothing on purpose",
	},
	{name: "barchart", constructor: func() templ.Component { return barchart.BarChart() }},
	{name: "breadcrumbs", constructor: func() templ.Component { return breadcrumbs.Breadcrumbs() }},
	{name: "button", constructor: func() templ.Component { return button.Button() }},
	{name: "card", constructor: func() templ.Component { return card.Card() }},
	{name: "codeblock", constructor: func() templ.Component { return codeblock.CodeBlock() }},
	{name: "commandpalette", constructor: func() templ.Component { return commandpalette.Palette(commandpalette.Props{}) }},
	{name: "cardgrid", constructor: func() templ.Component { return cardgrid.CardGrid(cardgrid.Props{}) }},
	{name: "checkbox", constructor: func() templ.Component { return checkbox.Checkbox() }},
	{name: "confirmdialog", constructor: func() templ.Component { return confirmdialog.ConfirmDialog(confirmdialog.Props{}) }},
	{name: "copybutton", constructor: func() templ.Component { return copybutton.CopyButton() }},
	{name: "customblock", constructor: func() templ.Component { return customblock.CustomBlock() }},
	{name: "custompage", constructor: func() templ.Component { return custompage.CustomPage() }},
	{name: "dashboardgrid", constructor: func() templ.Component { return dashboardgrid.Grid() }},
	{name: "dashboardheader", constructor: func() templ.Component { return dashboardheader.Header() }},
	{name: "dashboardpage", constructor: func() templ.Component { return dashboardpage.Page(dashboardpage.Props{}) }},
	{name: "datatable", constructor: func() templ.Component { return datatable.DataTable() }},
	{name: "datetimepicker", constructor: func() templ.Component { return datetimepicker.DateTimePicker() }},
	{name: "descriptionlist", constructor: func() templ.Component { return descriptionlist.DescriptionList() }},
	{name: "detailpage", constructor: func() templ.Component { return detailpage.Page(detailpage.Props{}) }},
	{name: "dialog", constructor: func() templ.Component { return dialog.Dialog() }},
	{name: "diffview", constructor: func() templ.Component { return diffview.DiffView(diffview.Props{}) }},
	{
		name:              "dropdown",
		constructor:       func() templ.Component { return dropdown.Dropdown() },
		expectEmpty:       true,
		expectEmptyReason: "Dropdown() only stashes a context value and renders `{ children... }`; with no children supplied, templ.GetChildren falls back to templ.NopComponent (a real, silent empty component — not the leaked-text bug), so zero-value output is legitimately zero bytes.",
	},
	{name: "emptystate", constructor: func() templ.Component { return emptystate.EmptyState() }},
	{name: "errorpage", constructor: func() templ.Component { return errorpage.ErrorPage() }},
	{name: "figure", constructor: func() templ.Component { return figure.Figure() }},
	{name: "filedrop", constructor: func() templ.Component { return filedrop.FileDrop() }},
	{name: "filterbar", constructor: func() templ.Component { return filterbar.FilterBar() }},
	{name: "flash", constructor: func() templ.Component { return flash.Flash(flash.Message{}) }},
	{name: "form", constructor: func() templ.Component { return form.Item() }},
	{name: "formfield", constructor: func() templ.Component { return formfield.FormField(formfield.Field{Name: "x"}) }},
	{name: "formpage", constructor: func() templ.Component { return formpage.Page(formpage.Props{}) }},
	{name: "graphwidget", constructor: func() templ.Component { return graphwidget.GraphWidget() }},
	{name: "hovercard", constructor: func() templ.Component { return hovercard.Hover(hovercard.Props{}) }},
	// icon.Icon(name) is a factory, not a Props-taking constructor; icon_defs.go
	// predefines one var per Lucide icon (e.g. AArrowDown = Icon("a-arrow-down")),
	// so calling one of those with zero Props exercises the same rendering path.
	{name: "icon", constructor: func() templ.Component { return icon.AArrowDown() }},
	{name: "image", constructor: func() templ.Component { return image.Image() }},
	{name: "jsonviewer", constructor: func() templ.Component { return jsonviewer.JSONViewer(jsonviewer.Props{}) }},
	{name: "keyvalueeditor", constructor: func() templ.Component { return keyvalueeditor.KeyValueEditor() }},
	{name: "input", constructor: func() templ.Component { return input.Input() }},
	{name: "label", constructor: func() templ.Component { return label.Label() }},
	{name: "layout", constructor: func() templ.Component { return layout.Stack() }},
	{name: "lifecyclestepper", constructor: func() templ.Component { return lifecyclestepper.LifecycleStepper() }},
	{name: "listpage", constructor: func() templ.Component { return listpage.Page(listpage.Props{}) }},
	{name: "logviewer", constructor: func() templ.Component { return logviewer.LogViewer() }},
	{name: "metricring", constructor: func() templ.Component { return metricring.MetricRing() }},
	{name: "nodepill", constructor: func() templ.Component { return nodepill.NodePill() }},
	{name: "pageheader", constructor: func() templ.Component { return pageheader.PageHeader() }},
	{name: "pagination", constructor: func() templ.Component { return pagination.Pagination() }},
	{name: "panel", constructor: func() templ.Component { return panel.Panel() }},
	{name: "piechart", constructor: func() templ.Component { return piechart.PieChart() }},
	// popover has no top-level Popover(); Content is the substantive floating
	// panel and the best single representative of the package.
	{name: "popover", constructor: func() templ.Component { return popover.Content() }},
	{name: "progressbar", constructor: func() templ.Component { return progressbar.ProgressBar() }},
	{name: "prose", constructor: func() templ.Component { return prose.Prose(prose.Props{Text: "x"}) }},
	{name: "action", constructor: func() templ.Component { return action.Button(action.Action{Label: "Go"}) }},
	{name: "radio", constructor: func() templ.Component { return radio.Radio() }},
	{name: "rating", constructor: func() templ.Component { return rating.Rating() }},
	{name: "selectbox", constructor: func() templ.Component { return selectbox.SelectBox() }},
	{name: "separator", constructor: func() templ.Component { return separator.Separator() }},
	{name: "settingspage", constructor: func() templ.Component { return settingspage.Page(settingspage.Props{}) }},
	{name: "skeleton", constructor: func() templ.Component { return skeleton.Lines(3) }},
	{name: "slideover", constructor: func() templ.Component { return slideover.SlideOver() }},
	{name: "spinner", constructor: func() templ.Component { return spinner.Spinner() }},
	{name: "splitpane", constructor: func() templ.Component { return splitpane.SplitPane(splitpane.Props{}) }},
	{name: "sparkline", constructor: func() templ.Component { return sparkline.Sparkline() }},
	{name: "statcomparison", constructor: func() templ.Component { return statcomparison.StatComparison() }},
	{name: "stattile", constructor: func() templ.Component { return stattile.StatTile() }},
	{name: "swatch", constructor: func() templ.Component { return swatch.Swatch() }},
	{name: "table", constructor: func() templ.Component { return table.Table() }},
	// tabs.Content() exercises the empty-Value rendering path covered by the
	// package-level regression test.
	{name: "tabs", constructor: func() templ.Component { return tabs.Content() }},
	{name: "tagsinput", constructor: func() templ.Component { return tagsinput.TagsInput() }},
	{name: "textarea", constructor: func() templ.Component { return textarea.Textarea() }},
	{name: "timeline", constructor: func() templ.Component { return timeline.Timeline() }},
	{
		name:              "toast",
		constructor:       func() templ.Component { return toast.Toast(toast.Props{}) },
		expectEmpty:       true,
		expectEmptyReason: "a toast with no title and no body renders nothing, like banner",
	},
	{name: "toggle", constructor: func() templ.Component { return toggle.Toggle() }},
	{name: "toolbar", constructor: func() templ.Component { return toolbar.Toolbar() }},
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
