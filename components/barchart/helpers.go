package barchart

import (
	"fmt"
	"math"
	"sort"
)

// labelCharW is the approximate width in px of a single label character at the
// 11px font size used for horizontal-orientation axis labels. Used both to size
// the adaptive label gutter and to decide when a label needs truncation.
const labelCharW = 6.5

// capBars enforces an optional cap on the number of bars. When maxBars <= 0
// (the default) or the input already fits, the bars are returned untouched in
// their original order — this preserves the historical behavior. When there are
// more bars than maxBars, the largest maxBars bars are kept (stable-sorted
// descending by raw Value, not |Value| — a large-magnitude negative sorts last)
// and everything else is folded into a single aggregated "Other (K)" bar whose
// value is the sum of the folded remainder and K is the number folded. The
// caller's slice is never mutated.
//
// @pure - sorts a copy; no mutation of the input, no I/O
// @complexity O(n log n) time, O(n) space, n = len(bars)
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

// longestLabelRunes returns the rune length of the widest bar label. Rune
// length (not byte length) is used so multi-byte UTF-8 labels don't over-size
// the gutter. Returns 0 for an empty slice or all-empty labels.
//
// @pure
// @complexity O(total label bytes)
func longestLabelRunes(bars []Bar) int {
	m := 0
	for _, b := range bars {
		if n := len([]rune(b.Label)); n > m {
			m = n
		}
	}
	return m
}

// computeLabelGutter sizes the left label gutter for horizontal bar charts so
// that real labels render fully instead of being clipped off the left edge of
// the viewBox. It returns the gutter width and the (possibly widened) SVG width.
//
// Rules:
//   - When labels are not shown, the historical 60px gutter and original width
//     are returned unchanged (no regression for label-less charts).
//   - The gutter is sized to the widest label (chars * labelCharW + padding),
//     with a 60px floor so short-label charts are pixel-identical to before.
//   - The gutter is capped at 45% of the SVG width so bars always have room. If
//     the desired gutter exceeds that ceiling, the SVG is widened (up to 2x the
//     original width) to accommodate it; anything still too long is clamped and
//     will be truncated with an ellipsis + <title> tooltip by fitLabel.
//
// @pure
// @complexity O(n) over bars (dominated by longestLabelRunes)
// @invariant outWidth <= 2 * width always; gutter <= 0.45 * outWidth whenever
// showLabels is true (the !showLabels early return keeps the fixed 60px gutter,
// which can exceed 45% of a width below 134px)
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

// fitLabel truncates a label to fit within a gutter of gutterPx pixels, adding a
// trailing ellipsis when it does. It returns the display string and whether it
// was truncated (so the caller can attach a <title> tooltip with the full text).
// A gutter too narrow for even one character still yields a bare "…" rather
// than an empty string, so the row is never silently unlabeled.
//
// @pure
// @complexity O(len(label))
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
