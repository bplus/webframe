package barchart

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := BarChart(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// Problem 1: a label wider than the old 60px gutter must render in full, and the
// SVG (viewBox) must widen to make room rather than clipping the label.
func TestRender_LongLabelRendersFullAndWidens(t *testing.T) {
	html := render(t, Props{
		Bars:       []Bar{{Label: "acme-billing/concept", Value: 10}},
		ShowLabels: true,
		Width:      300,
	})
	if !strings.Contains(html, "acme-billing/concept") {
		t.Errorf("full label should render, not clipped; html:\n%s", html)
	}
	// viewBox must have widened past the original 300px.
	if strings.Contains(html, `viewBox="0 0 300 `) {
		t.Errorf("SVG should have widened past 300px to fit the label; html:\n%s", html)
	}
}

// Problem 1: an overlong label is truncated with an ellipsis and its full text is
// preserved in a <title> tooltip.
func TestRender_OverlongLabelTruncatedWithTooltip(t *testing.T) {
	full := strings.Repeat("verylongtoken ", 8) // ~112 chars, exceeds even the 2x-widened gutter
	full = strings.TrimSpace(full)
	html := render(t, Props{
		Bars:       []Bar{{Label: full, Value: 10}},
		ShowLabels: true,
		Width:      300,
	})
	if !strings.Contains(html, "…") {
		t.Errorf("overlong label should be truncated with an ellipsis")
	}
	if !strings.Contains(html, "<title>") || !strings.Contains(html, full) {
		t.Errorf("full label should be preserved in a <title> tooltip; html:\n%s", html)
	}
}

// Problem 2: MaxBars caps the rendered bars and folds the rest into "Other (K)".
func TestRender_MaxBarsCapsAndAggregates(t *testing.T) {
	bars := make([]Bar, 0, 20)
	for i := 1; i <= 20; i++ {
		bars = append(bars, Bar{Label: fmt.Sprintf("src%02d", i), Value: float64(i)})
	}
	html := render(t, Props{Bars: bars, ShowLabels: true, MaxBars: 5})

	if !strings.Contains(html, "Other (15)") {
		t.Errorf("expected aggregated 'Other (15)' bar in rendered output")
	}
	// A folded small bar must not render its own label.
	if strings.Contains(html, "src01") {
		t.Errorf("src01 should have been folded into Other, but its label rendered")
	}
	// The largest bar survives.
	if !strings.Contains(html, "src20") {
		t.Errorf("largest bar src20 should still render")
	}
}

// A fixed-width/height SVG forced the whole page to scroll
// sideways at a narrow viewport even after its dashboardgrid.Slot had
// already collapsed to one column. max-width:100%/height:auto lets the
// SVG shrink to fit its container while viewBox keeps bars/labels drawn
// at their original scale.
func TestRender_SVGIsResponsive(t *testing.T) {
	html := render(t, Props{Bars: []Bar{{Label: "a", Value: 10}}, Width: 640, Height: 190})
	if !strings.Contains(html, `style="max-width:100%;height:auto;"`) {
		t.Errorf("expected a responsive max-width/height:auto style on the <svg>; html:\n%s", html)
	}
}

// Backward-compat: MaxBars=0 leaves all bars untouched (no Other bucket).
func TestRender_MaxBarsZeroUnlimited(t *testing.T) {
	bars := make([]Bar, 0, 20)
	for i := 1; i <= 20; i++ {
		bars = append(bars, Bar{Label: fmt.Sprintf("src%02d", i), Value: float64(i)})
	}
	html := render(t, Props{Bars: bars, ShowLabels: true, MaxBars: 0})
	if strings.Contains(html, "Other (") {
		t.Errorf("MaxBars=0 must not introduce an Other bucket")
	}
	if !strings.Contains(html, "src01") {
		t.Errorf("MaxBars=0 must keep every bar, src01 missing")
	}
}
