package piechart

import "testing"

func TestCapSlices_ZeroMaxUnlimited(t *testing.T) {
	slices := []Slice{
		{Label: "a", Value: 2},
		{Label: "b", Value: 9},
		{Label: "c", Value: 5},
	}
	got := capSlices(slices, 0)
	if len(got) != 3 {
		t.Fatalf("MaxSlices=0 should leave slices untouched, got %d", len(got))
	}
	if got[0].Label != "a" || got[1].Label != "b" || got[2].Label != "c" {
		t.Errorf("MaxSlices=0 must preserve order, got %q,%q,%q", got[0].Label, got[1].Label, got[2].Label)
	}
}

func TestCapSlices_UnderLimitUnchanged(t *testing.T) {
	slices := []Slice{{Label: "a", Value: 2}, {Label: "b", Value: 9}}
	got := capSlices(slices, 12)
	if len(got) != 2 || got[0].Label != "a" || got[1].Label != "b" {
		t.Errorf("len<=MaxSlices should be untouched in order, got %+v", got)
	}
}

func TestCapSlices_CapsAndAggregates(t *testing.T) {
	slices := make([]Slice, 0, 20)
	for v := 1; v <= 20; v++ {
		slices = append(slices, Slice{Label: "s", Value: float64(v)})
	}
	got := capSlices(slices, 5)
	if len(got) != 6 {
		t.Fatalf("expected 6 slices (5 kept + Other), got %d", len(got))
	}
	if got[0].Value != 20 {
		t.Errorf("top slice should be largest (20), got %v", got[0].Value)
	}
	for i := 1; i < 5; i++ {
		if got[i].Value > got[i-1].Value {
			t.Errorf("kept slices not sorted desc at %d", i)
		}
	}
	other := got[5]
	if other.Label != "Other (15)" {
		t.Errorf("expected 'Other (15)', got %q", other.Label)
	}
	// sum 1..15 == 120
	if other.Value != 120 {
		t.Errorf("Other should sum folded remainder (120), got %v", other.Value)
	}
}

func TestCapSlices_DoesNotMutateInput(t *testing.T) {
	slices := []Slice{{Label: "a", Value: 1}, {Label: "b", Value: 3}, {Label: "c", Value: 2}}
	_ = capSlices(slices, 1)
	if slices[0].Label != "a" || slices[1].Label != "b" || slices[2].Label != "c" {
		t.Errorf("capSlices must not mutate caller's slice, got %+v", slices)
	}
}
