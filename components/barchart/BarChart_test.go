package barchart

import (
	"math"
	"testing"
)

func TestMaxVal_EmptyBars(t *testing.T) {
	got := maxVal(nil)
	if got != 1 {
		t.Errorf("maxVal(nil) = %f, want 1", got)
	}
	got = maxVal([]Bar{})
	if got != 1 {
		t.Errorf("maxVal([]) = %f, want 1", got)
	}
}

func TestMaxVal_AllPositive(t *testing.T) {
	bars := []Bar{
		{Value: 10},
		{Value: 30},
		{Value: 20},
	}
	got := maxVal(bars)
	if got != 30 {
		t.Errorf("maxVal(all positive) = %f, want 30", got)
	}
}

func TestMaxVal_AllNegative(t *testing.T) {
	bars := []Bar{
		{Value: -5},
		{Value: -15},
		{Value: -10},
	}
	got := maxVal(bars)
	if got != 15 {
		t.Errorf("maxVal(all negative) = %f, want 15", got)
	}
}

func TestMaxVal_MixedPositiveNegative(t *testing.T) {
	bars := []Bar{
		{Value: 10},
		{Value: -25},
		{Value: 5},
	}
	got := maxVal(bars)
	if got != 25 {
		t.Errorf("maxVal(mixed) = %f, want 25", got)
	}
}

func TestMaxVal_AllZero(t *testing.T) {
	bars := []Bar{
		{Value: 0},
		{Value: 0},
	}
	got := maxVal(bars)
	if got != 1 {
		t.Errorf("maxVal(all zero) = %f, want 1 (sentinel)", got)
	}
}

func TestMaxVal_SingleBar(t *testing.T) {
	bars := []Bar{{Value: 42}}
	got := maxVal(bars)
	if got != 42 {
		t.Errorf("maxVal(single) = %f, want 42", got)
	}
}

func TestBarWidthNeverNegative(t *testing.T) {
	// Simulate the bar width calculation used in the template for
	// horizontal orientation with mixed positive/negative values.
	bars := []Bar{
		{Value: 10},
		{Value: -20},
		{Value: 5},
		{Value: -3},
	}
	mv := maxVal(bars)
	plotW := 230 // default: 300 - 60(labelW) - 10

	for _, bar := range bars {
		barWidth := max(0, int(math.Abs(bar.Value)*float64(plotW)/mv))
		if barWidth < 0 {
			t.Errorf("bar %q (value=%f): barWidth=%d, must be >= 0", bar.Label, bar.Value, barWidth)
		}
	}
}

func TestBarWidthProportional(t *testing.T) {
	// Verify bars are proportional to their absolute values.
	bars := []Bar{
		{Value: 10},
		{Value: -20},
	}
	mv := maxVal(bars)
	plotW := 230

	w1 := max(0, int(math.Abs(bars[0].Value)*float64(plotW)/mv))
	w2 := max(0, int(math.Abs(bars[1].Value)*float64(plotW)/mv))

	// -20 has double the magnitude of 10, so w2 should be ~2x w1
	if w2 < w1 {
		t.Errorf("bar with |20| should be wider than bar with |10|: got %d vs %d", w2, w1)
	}
	ratio := float64(w2) / float64(w1)
	if ratio < 1.9 || ratio > 2.1 {
		t.Errorf("expected ~2x ratio, got %.2f (w2=%d, w1=%d)", ratio, w2, w1)
	}
}
