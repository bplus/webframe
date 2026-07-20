package piechart

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
	if err := PieChart(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// MaxSlices caps the legend/arcs and folds the remainder into "Other (K)".
func TestRender_MaxSlicesCapsAndAggregates(t *testing.T) {
	slices := make([]Slice, 0, 20)
	for i := 1; i <= 20; i++ {
		slices = append(slices, Slice{Label: fmt.Sprintf("s%02d", i), Value: float64(i)})
	}
	html := render(t, Props{Slices: slices, ShowLegend: true, MaxSlices: 5})
	if !strings.Contains(html, "Other (15)") {
		t.Errorf("expected aggregated 'Other (15)' slice in legend")
	}
	if strings.Contains(html, "s01 ") {
		t.Errorf("s01 should have folded into Other, but rendered")
	}
	if !strings.Contains(html, "s20") {
		t.Errorf("largest slice s20 should still render")
	}
}

// Backward-compat: MaxSlices=0 leaves all slices untouched.
func TestRender_MaxSlicesZeroUnlimited(t *testing.T) {
	slices := make([]Slice, 0, 20)
	for i := 1; i <= 20; i++ {
		slices = append(slices, Slice{Label: fmt.Sprintf("s%02d", i), Value: float64(i)})
	}
	html := render(t, Props{Slices: slices, ShowLegend: true, MaxSlices: 0})
	if strings.Contains(html, "Other (") {
		t.Errorf("MaxSlices=0 must not introduce an Other slice")
	}
	if !strings.Contains(html, "s01") {
		t.Errorf("MaxSlices=0 must keep every slice, s01 missing")
	}
}
