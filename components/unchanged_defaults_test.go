package components_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/dialog"
	"github.com/bplus/webframe/components/panel"
	"github.com/bplus/webframe/components/slideover"
	"github.com/bplus/webframe/compose"
)

// The files under <package>/testdata/unchanged-*.html hold the HTML that
// dialog, slideover and panel rendered before they got their size, height,
// pinned-region and marker options. They pin the promise that a caller who
// sets none of the new fields gets the markup it got before: a change to one
// of these files is a change for every existing caller and must be meant.
//
// Regenerate with: go test ./components -run TestUnchangedDefaults -update-unchanged
var updateUnchanged = flag.Bool("update-unchanged", false, "rewrite the testdata/unchanged-*.html files from the current output")

func unchangedDialog() templ.Component {
	content := func(p dialog.ContentProps) templ.Component {
		return compose.With(dialog.Content(p),
			compose.With(dialog.Header(),
				compose.With(dialog.Title(), compose.Text("Title")),
				compose.With(dialog.Description(), compose.Text("Description")),
			),
			compose.With(dialog.Footer(),
				compose.With(dialog.Close(), compose.Text("Cancel")),
			),
		)
	}
	return compose.Join(
		compose.With(dialog.Dialog(dialog.Props{ID: "closed"}),
			compose.With(dialog.Trigger(), compose.Text("Open")),
			content(dialog.ContentProps{}),
		),
		compose.With(dialog.Dialog(dialog.Props{ID: "open", Open: true, DisableClickAway: true, DisableESC: true}),
			content(dialog.ContentProps{HideCloseButton: true, DisableAutoFocus: true}),
		),
		content(dialog.ContentProps{ID: "standalone", Open: true}),
	)
}

func unchangedSlideover() templ.Component {
	panelOf := func(p slideover.PanelProps) templ.Component {
		return compose.With(slideover.Panel(p),
			compose.With(slideover.Body(), compose.Text("Body")),
			compose.With(slideover.Footer(),
				compose.With(slideover.Close(), compose.Text("Done")),
			),
		)
	}
	parts := []templ.Component{
		compose.With(slideover.SlideOver(slideover.Props{ID: "root", Open: true}),
			compose.With(slideover.Trigger(), compose.Text("Open")),
			panelOf(slideover.PanelProps{Title: "Root"}),
		),
	}
	for _, w := range []slideover.Width{"", slideover.WidthSm, slideover.WidthMd, slideover.WidthLg, slideover.WidthXl, slideover.WidthFull, "unknown"} {
		parts = append(parts, panelOf(slideover.PanelProps{ID: "w-" + string(w), Title: "Width", Width: w}))
	}
	parts = append(parts, panelOf(slideover.PanelProps{ID: "left", Side: slideover.SideLeft, HideCloseButton: true, DisableAutoFocus: true}))
	for _, h := range []slideover.Height{"", slideover.HeightSm, slideover.HeightMd, slideover.HeightLg} {
		parts = append(parts, panelOf(slideover.PanelProps{ID: "h-" + string(h), Title: "Sheet", Side: slideover.SideBottom, Height: h}))
	}
	return compose.Join(parts...)
}

func unchangedPanel() templ.Component {
	body := compose.Text("Body")
	acts := []action.Action{{Label: "Edit", Href: "/edit"}}
	var parts []templ.Component
	for _, p := range []panel.Props{
		{},
		{ID: "titled", Title: "Title", Subtitle: "Subtitle", Actions: acts},
		{Title: "Muted", Variant: panel.VariantMuted, Level: 2},
		{Title: "Danger", Variant: panel.VariantDanger, Level: 4, Flush: true},
		{ID: "closed", Title: "Closed", Subtitle: "Subtitle", Actions: acts, Collapsible: true},
		{Title: "Open", Collapsible: true, Open: true, Level: 2},
		{Title: "Open h4", Collapsible: true, Open: true, Level: 4, Flush: true},
		{Collapsible: true},
	} {
		parts = append(parts, compose.With(panel.Panel(p), body))
	}
	return compose.Join(parts...)
}

func TestUnchangedDefaults(t *testing.T) {
	cases := []struct {
		pkg  string
		comp templ.Component
	}{
		{"dialog", unchangedDialog()},
		{"slideover", unchangedSlideover()},
		{"panel", unchangedPanel()},
	}
	for _, tc := range cases {
		t.Run(tc.pkg, func(t *testing.T) {
			// One element per line, so a difference reads as a small diff.
			got := strings.ReplaceAll(renderHTML(t, tc.comp), "><", ">\n<") + "\n"
			path := filepath.Join(tc.pkg, "testdata", "unchanged-defaults.html")
			if *updateUnchanged {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v (run with -update-unchanged to create it)", path, err)
			}
			if got != string(want) {
				t.Errorf("%s: output for callers that set no new field has changed.\n--- want\n%s\n--- got\n%s", tc.pkg, want, got)
			}
		})
	}
}
