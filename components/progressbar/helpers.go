package progressbar

// clampPercent converts a value/max pair into an integer percentage clamped to
// [0, 100], used by the progress bar template for the fill width. A
// non-positive max is treated as 100 so a zero-value Props (or a caller that
// only sets Value) still renders sensibly instead of dividing by zero.
// Integer division truncates: value=1, max=3 yields 33, not 34.
//
// @pure
func clampPercent(value, max int) int {
	if max <= 0 {
		max = 100
	}
	pct := (value * 100) / max
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}
