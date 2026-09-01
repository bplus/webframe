package dashboardgrid

// clamp restricts v to the range [lo, hi]. The dashboard grid template uses it
// to keep user-supplied column/row spans within the grid's actual bounds so a
// bad Props value degrades to an edge cell instead of broken CSS.
//
// @pure
// @invariant lo <= hi — callers pass literal grid bounds; with lo > hi the
// result is whichever bound triggers first, which is meaningless
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
