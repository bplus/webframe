package areachart

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func renderChart(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := AreaChart(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// The SVG scales down to its container like barchart's, so the default
// 400px width does not push a 390px phone page sideways.
func TestAreaChart_SVGIsResponsive(t *testing.T) {
	html := renderChart(t, Props{Series: []Series{{Label: "a", Data: []DataPoint{{Value: 1}, {Value: 2}}}}})
	if !strings.Contains(html, `style="max-width:100%;height:auto;"`) {
		t.Errorf("expected max-width:100%%;height:auto on the <svg>; html:\n%s", html)
	}
	if !strings.Contains(html, "max-w-full") {
		t.Errorf("expected the wrapper to be capped at its container (max-w-full); html:\n%s", html)
	}
}

// Series without a Color take theme tokens, so they follow light and dark.
func TestAreaChart_DefaultColorsAreThemeTokens(t *testing.T) {
	var series []Series
	for i := 0; i < 6; i++ {
		series = append(series, Series{Label: "s", Data: []DataPoint{{Value: 1}, {Value: 2}}})
	}
	html := renderChart(t, Props{Series: series, ShowLegend: true})
	if strings.Contains(html, `stroke="#`) || strings.Contains(html, "background-color: #") {
		t.Errorf("default series colors must be theme tokens, found a hex color; html:\n%s", html)
	}
	if !strings.Contains(html, `stroke="var(--color-primary)"`) {
		t.Errorf("first series should default to var(--color-primary); html:\n%s", html)
	}
}
