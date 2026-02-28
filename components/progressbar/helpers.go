package progressbar

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
