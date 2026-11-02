package barchart

import (
	"fmt"
	"math"
	"sort"
)

// labelCharW is the approximate width in px of one character of an 11px
// horizontal axis label.
const labelCharW = 6.5

// capBars keeps the maxBars largest bars (by raw Value, so a large negative
// sorts last) and folds the rest into one "Other (K)" bar holding their sum.
// It returns bars unchanged when maxBars <= 0 or they already fit, and never
// mutates the input.
func capBars(bars []Bar, maxBars int) []Bar {
	if maxBars <= 0 || len(bars) <= maxBars {
		return bars
	}
	sorted := make([]Bar, len(bars))
	copy(sorted, bars)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	var rest float64
	for _, b := range sorted[maxBars:] {
		rest += b.Value
	}
	out := make([]Bar, 0, maxBars+1)
	out = append(out, sorted[:maxBars]...)
	out = append(out, Bar{Label: fmt.Sprintf("Other (%d)", len(sorted)-maxBars), Value: rest})
	return out
}

// longestLabelRunes returns the length in runes of the longest bar label.
func longestLabelRunes(bars []Bar) int {
	m := 0
	for _, b := range bars {
		if n := len([]rune(b.Label)); n > m {
			m = n
		}
	}
	return m
}

// computeLabelGutter returns the left label gutter for a horizontal chart and
// the SVG width to use. Without labels the gutter is a fixed 60px. With
// labels it fits the widest label (minimum 60px) but stays within 45% of the
// SVG width, widening the SVG up to 2x to make room; labels that still don't
// fit are truncated by fitLabel.
func computeLabelGutter(bars []Bar, width int, showLabels bool) (gutter, outWidth int) {
	const floor = 60
	if !showLabels {
		return floor, width
	}
	desired := int(math.Ceil(float64(longestLabelRunes(bars))*labelCharW)) + 12
	if desired < floor {
		desired = floor
	}
	outWidth = width
	maxGutter := int(0.45 * float64(outWidth))
	if desired > maxGutter {
		// Widen the SVG so the gutter fits within the 45% ceiling, capped at 2x.
		wantW := int(math.Ceil(float64(desired) / 0.45))
		if cap := 2 * width; wantW > cap {
			wantW = cap
		}
		outWidth = wantW
		maxGutter = int(0.45 * float64(outWidth))
		if desired > maxGutter {
			desired = maxGutter
		}
	}
	return desired, outWidth
}

// fitLabel truncates label with a trailing ellipsis to fit gutterPx and
// reports whether it did. A gutter too narrow for one character still yields
// "…", so a row is never unlabeled.
func fitLabel(label string, gutterPx int) (string, bool) {
	runes := []rune(label)
	// Leave ~8px of padding inside the gutter for the text anchor offset.
	maxChars := int((float64(gutterPx) - 8) / labelCharW)
	if maxChars < 1 {
		maxChars = 1
	}
	if len(runes) <= maxChars {
		return label, false
	}
	if maxChars < 2 {
		return "…", true
	}
	return string(runes[:maxChars-1]) + "…", true
}
