package piechart

import (
	"fmt"
	"sort"
)

// capSlices enforces an optional cap on the number of pie slices. When the
// input exceeds maxSlices (> 0), the largest maxSlices slices are kept,
// sorted descending, and the rest fold into one "Other (K)" slice. Otherwise
// the input is returned unchanged. The caller's slice is never mutated.
func capSlices(slices []Slice, maxSlices int) []Slice {
	if maxSlices <= 0 || len(slices) <= maxSlices {
		return slices
	}
	sorted := make([]Slice, len(slices))
	copy(sorted, slices)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})
	var rest float64
	for _, s := range sorted[maxSlices:] {
		rest += s.Value
	}
	out := make([]Slice, 0, maxSlices+1)
	out = append(out, sorted[:maxSlices]...)
	out = append(out, Slice{Label: fmt.Sprintf("Other (%d)", len(sorted)-maxSlices), Value: rest})
	return out
}
