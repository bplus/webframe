package panel

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestPanel_Section(t *testing.T) {
	out := render(t, Panel(Props{Title: "Members"}))
	for _, want := range []string{"<section", "<h3", "Members"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "<details") {
		t.Errorf("unexpected <details in %s", out)
	}
}

func TestPanel_CollapsibleOpen(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Collapsible: true, Open: true}))
	for _, want := range []string{"<details open", "<summary"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestPanel_CollapsibleClosed(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Collapsible: true}))
	if !strings.Contains(out, "<details") {
		t.Errorf("missing <details in %s", out)
	}
	if strings.Contains(out, "<details open") {
		t.Errorf("unexpected open in %s", out)
	}
}

func TestPanel_Flush(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Flush: true}))
	if !strings.Contains(out, `class="p-0"`) {
		t.Errorf("missing class=p-0 in %s", out)
	}
	if strings.Contains(out, `class="p-4"`) {
		t.Errorf("unexpected class=p-4 in %s", out)
	}
}

func TestPanel_DangerVariant(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Variant: VariantDanger}))
	if !strings.Contains(out, "border-destructive") {
		t.Errorf("missing border-destructive in %s", out)
	}
}

func TestPanel_ActionsWrapperCanShrink(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Actions: []action.Action{{Label: "A"}}}))
	if strings.Contains(out, `<div class="shrink-0">`) {
		t.Errorf("actions wrapper must not be shrink-0: %s", out)
	}
	if !strings.Contains(out, `<div class="min-w-0 max-w-full">`) {
		t.Errorf("actions wrapper missing min-w-0 max-w-full: %s", out)
	}
}

func TestPanelLevelDefaultsToH3(t *testing.T) {
	out := render(t, Panel(Props{Title: "T"}))
	if !strings.Contains(out, `<h3 class="text-base font-semibold leading-none">`) {
		t.Errorf("want default h3 in %s", out)
	}
}

func TestPanelLevel2RendersH2(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Level: 2}))
	if !strings.Contains(out, `<h2 class="text-lg font-semibold leading-none">`) || !strings.Contains(out, "</h2>") || strings.Contains(out, "<h3") {
		t.Errorf("want h2 in %s", out)
	}
}

func TestPanelLevelClamps(t *testing.T) {
	if out := render(t, Panel(Props{Title: "T", Level: 9})); !strings.Contains(out, `<h4 class="text-sm font-semibold leading-none">`) || !strings.Contains(out, "</h4>") {
		t.Errorf("level 9 want h4 in %s", out)
	}
	if out := render(t, Panel(Props{Title: "T", Level: 1})); !strings.Contains(out, "<h2 ") || strings.Contains(out, "<h1") {
		t.Errorf("level 1 want h2 in %s", out)
	}
}

// A collapsible panel shows which state it is in: a decorative chevron at
// the start of the summary that turns when the panel's own details is open.
func TestPanel_CollapsibleMark(t *testing.T) {
	for _, open := range []bool{false, true} {
		out := render(t, Panel(Props{Title: "T", Collapsible: true, Open: open}))
		summary := out[strings.Index(out, "<summary"):strings.Index(out, "</summary>")]
		for _, want := range []string{"data-wf-panel-mark", `aria-hidden="true"`, "<svg", "[details[open]&gt;summary_&amp;]:rotate-90", "T</h3>"} {
			if !strings.Contains(summary, want) {
				t.Errorf("open=%v: missing %q in summary %s", open, want, summary)
			}
		}
		if strings.Index(summary, "data-wf-panel-mark") > strings.Index(summary, "<h3") {
			t.Errorf("open=%v: the mark must come before the title: %s", open, summary)
		}
		if !strings.Contains(summary, "focus-visible:ring-[3px]") {
			t.Errorf("open=%v: the summary needs a visible focus ring: %s", open, summary)
		}
	}
}

// Only a collapsible panel with a header has a mark.
func TestPanel_NoMarkWithoutSummary(t *testing.T) {
	for name, p := range map[string]Props{
		"plain":                {Title: "T", Subtitle: "S"},
		"collapsible headless": {Collapsible: true},
	} {
		if out := render(t, Panel(p)); strings.Contains(out, "data-wf-panel-mark") || strings.Contains(out, "<summary") {
			t.Errorf("%s: unexpected mark or summary in %s", name, out)
		}
	}
}
