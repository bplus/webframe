package piechart

import (
	"fmt"
	"sort"
)

// capSlices enforces an optional cap on the number of pie slices. When
// maxSlices <= 0 (the default) or the input already fits, the slices are
// returned untouched in their original order. When there are more slices than
// maxSlices, the largest maxSlices slices are kept (sorted descending) and the
// rest are folded into a single aggregated "Other (K)" slice whose value is the
// sum of the folded remainder. The folded slice has no explicit color, so it
// falls back to the default palette. The caller's slice is never mutated.
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
