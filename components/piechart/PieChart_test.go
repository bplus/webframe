package piechart

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/bplus/webframe/utils"
)

// --- total() tests ---

func TestTotal_AllPositive(t *testing.T) {
	slices := []Slice{
		{Value: 10},
		{Value: 20},
		{Value: 30},
	}
	got := total(slices)
	if got != 60 {
		t.Errorf("total(all positive) = %f, want 60", got)
	}
}

func TestTotal_MixedWithNegatives(t *testing.T) {
	slices := []Slice{
		{Value: 10},
		{Value: -5},
		{Value: 20},
	}
	got := total(slices)
	// Should only sum positives: 10 + 20 = 30
	if got != 30 {
		t.Errorf("total(mixed) = %f, want 30 (only positive values)", got)
	}
}

func TestTotal_AllNegative(t *testing.T) {
	slices := []Slice{
		{Value: -10},
		{Value: -5},
	}
	got := total(slices)
	// No positive values, sum is 0, sentinel returns 1
	if got != 1 {
		t.Errorf("total(all negative) = %f, want 1 (sentinel)", got)
	}
}

func TestTotal_AllZero(t *testing.T) {
	slices := []Slice{
		{Value: 0},
		{Value: 0},
	}
	got := total(slices)
	if got != 1 {
		t.Errorf("total(all zero) = %f, want 1 (sentinel)", got)
	}
}

func TestTotal_Empty(t *testing.T) {
	got := total(nil)
	if got != 1 {
		t.Errorf("total(nil) = %f, want 1 (sentinel)", got)
	}
}

// --- computeArcs() tests ---

func TestComputeArcs_SingleSlice(t *testing.T) {
	slices := []Slice{{Value: 100, Color: "#f00"}}
	arcs := computeArcs(slices)
	if len(arcs) != 1 {
		t.Fatalf("expected 1 arc, got %d", len(arcs))
	}
	sweep := arcs[0].endAngle - arcs[0].startAngle
	if math.Abs(sweep-2*math.Pi) > 0.01 {
		t.Errorf("single slice sweep = %f, want ~%f (full circle)", sweep, 2*math.Pi)
	}
}

func TestComputeArcs_TwoEqualSlices(t *testing.T) {
	slices := []Slice{
		{Value: 50, Color: "#f00"},
		{Value: 50, Color: "#0f0"},
	}
	arcs := computeArcs(slices)
	if len(arcs) != 2 {
		t.Fatalf("expected 2 arcs, got %d", len(arcs))
	}
	for i, arc := range arcs {
		sweep := arc.endAngle - arc.startAngle
		if math.Abs(sweep-math.Pi) > 0.01 {
			t.Errorf("arc[%d] sweep = %f, want ~%f (half circle)", i, sweep, math.Pi)
		}
	}
}

func TestComputeArcs_ZeroValueSlice(t *testing.T) {
	slices := []Slice{
		{Value: 100, Color: "#f00"},
		{Value: 0, Color: "#0f0"},
	}
	arcs := computeArcs(slices)
	if len(arcs) != 2 {
		t.Fatalf("expected 2 arcs, got %d", len(arcs))
	}
	// Zero-value slice should have zero sweep
	sweep := arcs[1].endAngle - arcs[1].startAngle
	if sweep != 0 {
		t.Errorf("zero-value arc sweep = %f, want 0", sweep)
	}
}

func TestComputeArcs_NegativeValueSlice(t *testing.T) {
	slices := []Slice{
		{Value: 100, Color: "#f00"},
		{Value: -50, Color: "#0f0"},
	}
	arcs := computeArcs(slices)
	if len(arcs) != 2 {
		t.Fatalf("expected 2 arcs, got %d", len(arcs))
	}
	// Negative-value slice should have zero sweep
	sweep := arcs[1].endAngle - arcs[1].startAngle
	if sweep != 0 {
		t.Errorf("negative-value arc sweep = %f, want 0", sweep)
	}
	// Positive slice should be the full circle
	positiveSweep := arcs[0].endAngle - arcs[0].startAngle
	if math.Abs(positiveSweep-2*math.Pi) > 0.01 {
		t.Errorf("positive arc sweep = %f, want ~%f (full circle since negative is excluded)", positiveSweep, 2*math.Pi)
	}
}

func TestComputeArcs_DefaultColors(t *testing.T) {
	slices := []Slice{
		{Value: 50}, // no Color — should get defaultColors[0]
	}
	arcs := computeArcs(slices)
	if arcs[0].color != defaultColors[0] {
		t.Errorf("arc color = %q, want defaultColors[0] = %q", arcs[0].color, defaultColors[0])
	}
}

// --- resolveInnerSize() tests ---

func TestResolveInnerSize_Nil(t *testing.T) {
	p := Props{}
	got := p.resolveInnerSize()
	if got != 50 {
		t.Errorf("resolveInnerSize(nil) = %d, want 50 (default donut)", got)
	}
}

func TestResolveInnerSize_Zero(t *testing.T) {
	zero := 0
	p := Props{InnerSize: &zero}
	got := p.resolveInnerSize()
	if got != 0 {
		t.Errorf("resolveInnerSize(0) = %d, want 0 (full pie)", got)
	}
}

func TestResolveInnerSize_Positive(t *testing.T) {
	val := 30
	p := Props{InnerSize: &val}
	got := p.resolveInnerSize()
	if got != 30 {
		t.Errorf("resolveInnerSize(30) = %d, want 30", got)
	}
}

// --- arcPath() tests ---

func TestArcPath_LargeArcFlag(t *testing.T) {
	// Sweep > Pi should produce largeArc=1
	cx, cy, outerR, innerR := 80, 80, 76, 50
	start := -math.Pi / 2
	end := start + math.Pi + 0.5 // > 180 degrees

	path := arcPath(cx, cy, outerR, innerR, start, end)
	// largeArc=1 means the "1" appears in the A command
	// Format: A rx ry rotation largeArc sweep x y
	if !strings.Contains(path, " 1 1 ") {
		t.Errorf("expected large arc flag=1 for sweep > Pi, got path: %s", path)
	}
}

func TestArcPath_SmallArc(t *testing.T) {
	// Sweep < Pi should produce largeArc=0
	cx, cy, outerR, innerR := 80, 80, 76, 50
	start := -math.Pi / 2
	end := start + math.Pi/4 // 45 degrees

	path := arcPath(cx, cy, outerR, innerR, start, end)
	// Should have "0 1" (largeArc=0, sweep=1) in the arc command
	if !strings.Contains(path, " 0 1 ") {
		t.Errorf("expected large arc flag=0 for sweep < Pi, got path: %s", path)
	}
}

func TestArcPath_FullCircle(t *testing.T) {
	// Full circle (2*Pi) should produce two arcs (not a degenerate path)
	cx, cy, outerR, innerR := 80, 80, 76, 50
	start := -math.Pi / 2
	end := start + 2*math.Pi

	path := arcPath(cx, cy, outerR, innerR, start, end)
	// Full circle is split into two arcs, so we should see at least 2 "A" commands
	arcCount := strings.Count(path, " A ")
	// The path starts with M, and each half has an A and an inner A = 4 total
	if arcCount < 2 {
		t.Errorf("full circle should have >= 2 arc commands, got %d in path: %s", arcCount, path)
	}
}

func TestArcPath_NoInnerRadius(t *testing.T) {
	// innerR=0 should produce a pie wedge (path ends with L cx cy Z)
	cx, cy, outerR := 80, 80, 76
	start := -math.Pi / 2
	end := start + math.Pi/2

	path := arcPath(cx, cy, outerR, 0, start, end)
	expected := "L 80 80 Z"
	if !strings.Contains(path, expected) {
		t.Errorf("pie wedge should end with %q, got path: %s", expected, path)
	}
}

// Default colors are theme tokens that survive SanitizeColor unchanged and
// never repeat within the palette.
func TestDefaultColors_ThemeTokens(t *testing.T) {
	seen := map[string]bool{}
	for i, c := range defaultColors {
		if !strings.Contains(c, "var(--color-") || strings.Contains(c, "#") {
			t.Errorf("defaultColors[%d] = %q, expected a var(--color-...) theme token", i, c)
		}
		if got := utils.SanitizeColor(c); got != c {
			t.Errorf("defaultColors[%d] = %q is rewritten to %q by SanitizeColor", i, c, got)
		}
		if seen[c] {
			t.Errorf("defaultColors[%d] = %q repeats an earlier color", i, c)
		}
		seen[c] = true
	}
}

// A pie with no slice colors draws no hard-coded hex, in arcs or legend.
func TestRender_DefaultColorsHaveNoHex(t *testing.T) {
	slices := make([]Slice, 0, 10)
	for i := 1; i <= 10; i++ {
		slices = append(slices, Slice{Label: fmt.Sprintf("s%d", i), Value: float64(i)})
	}
	var buf bytes.Buffer
	if err := PieChart(Props{Slices: slices, ShowLegend: true}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	if strings.Contains(html, `fill="#`) || strings.Contains(html, "background-color: #") {
		t.Errorf("default pie colors should be theme tokens, got hex in:\n%s", html)
	}
	if !strings.Contains(html, `fill="var(--color-primary)"`) {
		t.Errorf("first slice should be filled with var(--color-primary)")
	}
}
