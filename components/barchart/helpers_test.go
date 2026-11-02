package barchart

import "testing"

// --- capBars ---

func TestCapBars_ZeroMaxUnlimited(t *testing.T) {
	// MaxBars == 0 means unlimited: bars returned untouched, original order.
	bars := []Bar{
		{Label: "a", Value: 2},
		{Label: "b", Value: 9},
		{Label: "c", Value: 5},
	}
	got := capBars(bars, 0)
	if len(got) != 3 {
		t.Fatalf("MaxBars=0 should leave bars untouched, got %d bars", len(got))
	}
	// Order preserved (no sort when not capping).
	if got[0].Label != "a" || got[1].Label != "b" || got[2].Label != "c" {
		t.Errorf("MaxBars=0 must preserve order, got %q,%q,%q", got[0].Label, got[1].Label, got[2].Label)
	}
}

func TestCapBars_UnderLimitUnchanged(t *testing.T) {
	bars := []Bar{
		{Label: "a", Value: 2},
		{Label: "b", Value: 9},
	}
	got := capBars(bars, 12)
	if len(got) != 2 {
		t.Fatalf("len<=MaxBars should be untouched, got %d", len(got))
	}
	if got[0].Label != "a" || got[1].Label != "b" {
		t.Errorf("under-limit order must be preserved, got %q,%q", got[0].Label, got[1].Label)
	}
}

func TestCapBars_CapsAndAggregates(t *testing.T) {
	// 54 bars: values 1..54 in ascending input order (to prove we sort desc).
	bars := make([]Bar, 0, 54)
	for v := 1; v <= 54; v++ {
		bars = append(bars, Bar{Label: "src", Value: float64(v)})
	}
	got := capBars(bars, 12)

	if len(got) != 13 {
		t.Fatalf("expected 13 bars (12 kept + Other), got %d", len(got))
	}
	if got[0].Value != 54 {
		t.Errorf("top bar should be largest (54), got %v", got[0].Value)
	}
	if got[11].Value != 43 {
		t.Errorf("12th kept bar should be 43, got %v", got[11].Value)
	}
	// Descending order across kept bars.
	for i := 1; i < 12; i++ {
		if got[i].Value > got[i-1].Value {
			t.Errorf("kept bars not sorted desc at %d: %v > %v", i, got[i].Value, got[i-1].Value)
		}
	}
	other := got[12]
	if other.Label != "Other (42)" {
		t.Errorf("expected aggregated label 'Other (42)', got %q", other.Label)
	}
	// sum 1..42 == 903
	if other.Value != 903 {
		t.Errorf("Other should sum the folded remainder (903), got %v", other.Value)
	}
}

func TestCapBars_DoesNotMutateInput(t *testing.T) {
	bars := []Bar{
		{Label: "a", Value: 1},
		{Label: "b", Value: 3},
		{Label: "c", Value: 2},
	}
	_ = capBars(bars, 1)
	if bars[0].Label != "a" || bars[1].Label != "b" || bars[2].Label != "c" {
		t.Errorf("capBars must not mutate the caller's slice, got %q,%q,%q", bars[0].Label, bars[1].Label, bars[2].Label)
	}
}

// --- computeLabelGutter ---

func TestComputeLabelGutter_ShortLabelsKeepFloor(t *testing.T) {
	// Short labels keep the 60px minimum gutter and the requested width.
	bars := []Bar{{Label: "A"}, {Label: "BB"}, {Label: "CCC"}}
	gutter, w := computeLabelGutter(bars, 300, true)
	if gutter != 60 {
		t.Errorf("short labels should keep 60px gutter floor, got %d", gutter)
	}
	if w != 300 {
		t.Errorf("short labels should not widen SVG, got width %d", w)
	}
}

func TestComputeLabelGutter_NoLabelsUnchanged(t *testing.T) {
	// Without labels, the gutter is a fixed 60px and the width is unchanged.
	bars := []Bar{{Label: "a very long label indeed that would overflow"}}
	gutter, w := computeLabelGutter(bars, 300, false)
	if gutter != 60 {
		t.Errorf("!ShowLabels should keep 60px gutter, got %d", gutter)
	}
	if w != 300 {
		t.Errorf("!ShowLabels should not widen SVG, got %d", w)
	}
}

func TestComputeLabelGutter_LongLabelWidensGutter(t *testing.T) {
	// A label wider than 60px widens the gutter.
	bars := []Bar{{Label: "acme-billing/concept"}} // 20 chars
	gutter, w := computeLabelGutter(bars, 300, true)
	if gutter <= 60 {
		t.Errorf("long label must widen gutter beyond 60px, got %d", gutter)
	}
	// Gutter must never exceed 45%% of the (possibly widened) width, so bars still show.
	if float64(gutter) > 0.45*float64(w)+0.5 {
		t.Errorf("gutter %d exceeds 45%% of width %d", gutter, w)
	}
	// Plot area must remain positive.
	if w-gutter-10 <= 0 {
		t.Errorf("no room left for bars: width=%d gutter=%d", w, gutter)
	}
}

func TestComputeLabelGutter_PathologicalLabelClampsWidth(t *testing.T) {
	// An absurdly long label must not blow up the SVG without bound; width is
	// capped and the gutter still respects the 45%% ceiling.
	long := ""
	for i := 0; i < 120; i++ {
		long += "x"
	}
	bars := []Bar{{Label: long}}
	gutter, w := computeLabelGutter(bars, 300, true)
	if w > 2*300 {
		t.Errorf("width should be capped at 2x original, got %d", w)
	}
	if float64(gutter) > 0.45*float64(w)+0.5 {
		t.Errorf("gutter %d exceeds 45%% of width %d", gutter, w)
	}
}

// --- fitLabel ---

func TestFitLabel_FitsUntouched(t *testing.T) {
	disp, truncated := fitLabel("short", 60)
	if truncated {
		t.Errorf("label that fits should not be truncated")
	}
	if disp != "short" {
		t.Errorf("fitting label should be returned verbatim, got %q", disp)
	}
}

func TestFitLabel_TruncatesWithEllipsis(t *testing.T) {
	long := "acmewidgetlongoverflowinglabel"
	disp, truncated := fitLabel(long, 40) // small gutter forces truncation
	if !truncated {
		t.Errorf("overlong label should be truncated")
	}
	if len([]rune(disp)) >= len([]rune(long)) {
		t.Errorf("truncated label should be shorter than original")
	}
	if []rune(disp)[len([]rune(disp))-1] != '…' {
		t.Errorf("truncated label should end with ellipsis, got %q", disp)
	}
}
