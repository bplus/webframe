package lifecyclestepper

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderCompact(t *testing.T, steps []Step) string {
	t.Helper()
	var buf bytes.Buffer
	if err := LifecycleStepper(Props{Steps: steps}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render LifecycleStepper: %v", err)
	}
	return buf.String()
}

func renderLarge(t *testing.T, steps []Step) string {
	t.Helper()
	var buf bytes.Buffer
	if err := LifecycleStepper(Props{Steps: steps, Large: true}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render LifecycleStepper (Large): %v", err)
	}
	return buf.String()
}

// allStates is one step per state, in an order that also exercises
// the connector rule: done->active (grey), active->attn (grey),
// attn->failed (grey), failed->future (grey), future->skipped (grey).
// A separate two-step case below covers the done->done (green) path.
func allStates() []Step {
	return []Step{
		{Label: "Submitted", State: StateDone},
		{Label: "Started", State: StateActive},
		{Label: "Build", State: StateAttn},
		{Label: "Review", State: StateFailed},
		{Label: "Deploy", State: StateFuture},
		{Label: "Reflect", State: StateSkipped},
	}
}

func TestRender_BothSizes_AllStates(t *testing.T) {
	for name, render := range map[string]func(*testing.T, []Step) string{
		"compact": renderCompact,
		"large":   renderLarge,
	} {
		t.Run(name, func(t *testing.T) {
			html := render(t, allStates())
			for _, want := range []string{"Submitted", "Started", "Build", "Review", "Deploy", "Reflect"} {
				if !strings.Contains(html, want) {
					t.Errorf("expected label %q in output; html:\n%s", want, html)
				}
			}
		})
	}
}

// Done state renders the exact check-mark SVG path from the design
// spec, in the success foreground, on a --color-success circle.
func TestRender_DoneState_ChecksSVGPath(t *testing.T) {
	html := renderCompact(t, []Step{{Label: "Submitted", State: StateDone}})
	if !strings.Contains(html, "M2.5 6.5L5 9L9.5 3.5") {
		t.Errorf("expected done check path in output; html:\n%s", html)
	}
	if !strings.Contains(html, "background-color:var(--color-success);border:none") {
		t.Errorf("expected done circle color var(--color-success) in output; html:\n%s", html)
	}
}

// Skipped state renders a dashed border (never solid) using the
// future/skipped border color.
func TestRender_SkippedState_DashedBorder(t *testing.T) {
	html := renderCompact(t, []Step{
		{Label: "Submitted", State: StateDone},
		{Label: "Reflect", State: StateSkipped},
	})
	if !strings.Contains(html, "1.5px dashed var(--color-border)") {
		t.Errorf("expected skipped node to have a dashed --color-border border; html:\n%s", html)
	}
	if strings.Contains(html, "1.5px solid var(--color-border)") {
		t.Errorf("skipped node must not also render a solid border; html:\n%s", html)
	}
}

// Future state (the zero State value too) renders a solid border and
// the 1-based index as its glyph, not the skipped en dash.
func TestRender_FutureState_SolidBorderAndIndex(t *testing.T) {
	html := renderCompact(t, []Step{
		{Label: "Submitted", State: StateDone},
		{Label: "Started", State: StateActive},
		{Label: "Deploy", State: StateFuture},
	})
	if !strings.Contains(html, "1.5px solid var(--color-border)") {
		t.Errorf("expected future node to have a solid --color-border border; html:\n%s", html)
	}
	// Deploy is the 3rd step (1-based index 3).
	if !strings.Contains(html, ">3</span>") {
		t.Errorf("expected future node glyph to be its 1-based index (3); html:\n%s", html)
	}
}

// hx-get (and hx-target/hx-swap) must appear only on nodes whose
// PopoverURL is set; a node without one stays a plain, non-hx div.
func TestRender_PopoverURL_GatesHxAttributes(t *testing.T) {
	html := renderCompact(t, []Step{
		{Label: "Submitted", State: StateDone, PopoverURL: "/stages/1", PopoverTarget: "#detail"},
		{Label: "Started", State: StateActive},
	})
	if !strings.Contains(html, `hx-get="/stages/1"`) {
		t.Errorf("expected hx-get for the step with PopoverURL set; html:\n%s", html)
	}
	if !strings.Contains(html, `hx-target="#detail"`) {
		t.Errorf("expected hx-target for the step with PopoverTarget set; html:\n%s", html)
	}
	if !strings.Contains(html, `hx-swap="innerHTML"`) {
		t.Errorf("expected hx-swap=\"innerHTML\" alongside hx-get; html:\n%s", html)
	}
	if strings.Contains(html, "<button") == false {
		t.Errorf("expected a <button> root element for the clickable step; html:\n%s", html)
	}
	// The second step has no PopoverURL: it must not carry hx-get anywhere
	// scoped to it. We approximate by counting hx-get occurrences: exactly one.
	if got := strings.Count(html, "hx-get="); got != 1 {
		t.Errorf("expected exactly 1 hx-get attribute (only the step with PopoverURL), got %d; html:\n%s", got, html)
	}
}

// A step with no PopoverURL renders as a plain <div>, not a <button>,
// and carries no hx-get attribute at all.
func TestRender_NoPopoverURL_PlainDiv(t *testing.T) {
	html := renderCompact(t, []Step{{Label: "Submitted", State: StateDone}})
	if strings.Contains(html, "hx-get=") {
		t.Errorf("expected no hx-get without PopoverURL; html:\n%s", html)
	}
	if strings.Contains(html, "<button") {
		t.Errorf("expected a plain <div> node without PopoverURL, not a <button>; html:\n%s", html)
	}
}

// Title, when set, renders as a plain title attribute regardless of
// PopoverURL.
func TestRender_Title_RendersHoverAttribute(t *testing.T) {
	html := renderCompact(t, []Step{{Label: "Submitted", State: StateDone, Title: "Ran at 10:02"}})
	if !strings.Contains(html, `title="Ran at 10:02"`) {
		t.Errorf("expected title attribute; html:\n%s", html)
	}
}

// The connector rule: only a done->done pair is green (--color-success); a
// done->skipped pair (or any pair with a non-done neighbor) must
// render the grey off-color, never green — a fully-green bar can only
// mean every stage actually ran.
func TestRender_ConnectorColor_DoneDoneGreen(t *testing.T) {
	html := renderCompact(t, []Step{
		{Label: "Submitted", State: StateDone},
		{Label: "Started", State: StateDone},
	})
	if !strings.Contains(html, "background-color:var(--color-success);display:block;flex-shrink:0") {
		t.Errorf("expected a green connector between two done nodes; html:\n%s", html)
	}
}

func TestRender_ConnectorColor_DoneSkippedGrey(t *testing.T) {
	html := renderCompact(t, []Step{
		{Label: "Submitted", State: StateDone},
		{Label: "Reflect", State: StateSkipped},
	})
	if strings.Contains(html, "background-color:var(--color-success);display:block;flex-shrink:0") {
		t.Errorf("connector adjacent to a skipped node must not be green; html:\n%s", html)
	}
	// Grey connector uses the --color-border theme token (#DBDAD5 in
	// static/css/input.css), not the hard-coded done-green.
	if !strings.Contains(html, "background-color:var(--color-border);display:block;flex-shrink:0") {
		t.Errorf("expected connector to use the var(--color-border) grey token; html:\n%s", html)
	}
}

// A single step renders no connector at all.
func TestRender_SingleStep_NoConnector(t *testing.T) {
	html := renderCompact(t, []Step{{Label: "Submitted", State: StateDone}})
	if strings.Contains(html, "display:block") {
		t.Errorf("a single step must not render a connector line; html:\n%s", html)
	}
}

// Sizes differ: compact uses 22px dots / 12.5px labels / 26px
// connectors, large uses 26px / 13.5px / 34px.
func TestRender_SizesDiffer(t *testing.T) {
	steps := []Step{
		{Label: "Submitted", State: StateDone},
		{Label: "Started", State: StateActive},
	}
	compact := renderCompact(t, steps)
	large := renderLarge(t, steps)

	if !strings.Contains(compact, "width:22px;height:22px") {
		t.Errorf("expected compact dot to be 22px; html:\n%s", compact)
	}
	if !strings.Contains(compact, "font-size:12.5px") {
		t.Errorf("expected compact label to be 12.5px; html:\n%s", compact)
	}
	if !strings.Contains(compact, "width:26px;height:1.5px") {
		t.Errorf("expected compact connector to be 26px wide; html:\n%s", compact)
	}

	if !strings.Contains(large, "width:26px;height:26px") {
		t.Errorf("expected large dot to be 26px; html:\n%s", large)
	}
	if !strings.Contains(large, "font-size:13.5px") {
		t.Errorf("expected large label to be 13.5px; html:\n%s", large)
	}
	if !strings.Contains(large, "width:34px;height:1.5px") {
		t.Errorf("expected large connector to be 34px wide; html:\n%s", large)
	}
}

func TestRender_EmptySteps(t *testing.T) {
	html := renderCompact(t, nil)
	if strings.Contains(html, "<button") || strings.Contains(html, "display:block") {
		t.Errorf("empty steps should render an empty row; html:\n%s", html)
	}
}

// Props.ID, Props.Class, and Props.Attributes all land on the root
// element: ID as a plain id attribute, Class merged alongside the
// component's base classes via TwMerge, and Attributes spread
// verbatim.
func TestRender_Props_IDClassAttributes(t *testing.T) {
	var buf bytes.Buffer
	props := Props{
		ID:         "x",
		Class:      "mt-2",
		Attributes: templ.Attributes{"data-foo": "bar"},
	}
	if err := LifecycleStepper(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render LifecycleStepper: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `id="x"`) {
		t.Errorf("expected id=\"x\" on the root element; html:\n%s", html)
	}
	if !strings.Contains(html, "gap:12px 8px") {
		t.Errorf("expected the row's 8px column gap (canvas: steps and connectors are 8px apart); html:\n%s", html)
	}
	if !strings.Contains(html, "inline-flex") || !strings.Contains(html, "mt-2") {
		t.Errorf("expected the base class and mt-2 merged onto the root element; html:\n%s", html)
	}
	if !strings.Contains(html, `data-foo="bar"`) {
		t.Errorf("expected data-foo=\"bar\" spread onto the root element; html:\n%s", html)
	}
}
