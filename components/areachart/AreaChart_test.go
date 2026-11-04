package areachart

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/bplus/webframe/utils"
)

// --- globalMinMax ---

func TestGlobalMinMax_EmptySeries(t *testing.T) {
	minV, maxV := globalMinMax(nil)
	if minV != 0 || maxV != 1 {
		t.Errorf("expected (0, 1) for nil series, got (%.2f, %.2f)", minV, maxV)
	}
	minV, maxV = globalMinMax([]Series{})
	if minV != 0 || maxV != 1 {
		t.Errorf("expected (0, 1) for empty series, got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_SeriesWithNoData(t *testing.T) {
	series := []Series{
		{Label: "A", Data: []DataPoint{}},
		{Label: "B", Data: nil},
	}
	minV, maxV := globalMinMax(series)
	if minV != 0 || maxV != 1 {
		t.Errorf("expected (0, 1) for series with no data points, got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_SinglePoint(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: 5}}},
	}
	minV, maxV := globalMinMax(series)
	// When min == max, function returns (min, min+1)
	if minV != 5 || maxV != 6 {
		t.Errorf("expected (5, 6) for single point, got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_AllNegative(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: -10}, {Value: -5}, {Value: -1}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != -10 || maxV != -1 {
		t.Errorf("expected (-10, -1), got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_AllPositive(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: 1}, {Value: 5}, {Value: 10}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != 1 || maxV != 10 {
		t.Errorf("expected (1, 10), got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_MixedPositiveNegative(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: -5}, {Value: 0}, {Value: 5}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != -5 || maxV != 5 {
		t.Errorf("expected (-5, 5), got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_AllSameValues(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: 3}, {Value: 3}, {Value: 3}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != 3 || maxV != 4 {
		t.Errorf("expected (3, 4) when all same, got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_AllZero(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: 0}, {Value: 0}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != 0 || maxV != 1 {
		t.Errorf("expected (0, 1) when all zero, got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_MultipleSeries(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: 0}, {Value: 10}}},
		{Data: []DataPoint{{Value: -5}, {Value: 20}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != -5 || maxV != 20 {
		t.Errorf("expected (-5, 20) across series, got (%.2f, %.2f)", minV, maxV)
	}
}

func TestGlobalMinMax_SpecialFloats(t *testing.T) {
	series := []Series{
		{Data: []DataPoint{{Value: math.SmallestNonzeroFloat64}, {Value: math.MaxFloat64}}},
	}
	minV, maxV := globalMinMax(series)
	if minV != math.SmallestNonzeroFloat64 {
		t.Errorf("expected min=SmallestNonzeroFloat64, got %v", minV)
	}
	if maxV != math.MaxFloat64 {
		t.Errorf("expected max=MaxFloat64, got %v", maxV)
	}
}

// --- buildAreaPoints ---

func TestBuildAreaPoints_EmptyData(t *testing.T) {
	polyline, polygon := buildAreaPoints(nil, 400, 200, 10, 10, 10, 20, 0, 100)
	if polyline != "" || polygon != "" {
		t.Errorf("expected empty strings for nil data, got polyline=%q polygon=%q", polyline, polygon)
	}
	polyline, polygon = buildAreaPoints([]DataPoint{}, 400, 200, 10, 10, 10, 20, 0, 100)
	if polyline != "" || polygon != "" {
		t.Errorf("expected empty strings for empty data, got polyline=%q polygon=%q", polyline, polygon)
	}
}

func TestBuildAreaPoints_SinglePoint(t *testing.T) {
	data := []DataPoint{{Label: "A", Value: 50}}
	polyline, polygon := buildAreaPoints(data, 400, 200, 10, 10, 10, 20, 0, 100)
	if polyline == "" {
		t.Fatal("expected non-empty polyline for single point")
	}
	if polygon == "" {
		t.Fatal("expected non-empty polygon for single point")
	}
	// Polyline should be a single point
	if strings.Contains(polyline, " ") {
		t.Errorf("expected single point in polyline (no spaces), got %q", polyline)
	}
	// Polygon should extend the polyline with bottom corners
	if !strings.HasPrefix(polygon, polyline) {
		t.Errorf("polygon should start with polyline content")
	}
	if len(polygon) <= len(polyline) {
		t.Errorf("polygon should be longer than polyline (includes corners)")
	}
}

func TestBuildAreaPoints_BasicData(t *testing.T) {
	data := []DataPoint{
		{Label: "Jan", Value: 10},
		{Label: "Feb", Value: 50},
		{Label: "Mar", Value: 30},
	}
	polyline, polygon := buildAreaPoints(data, 400, 200, 10, 10, 10, 20, 10, 50)
	if polyline == "" {
		t.Fatal("expected non-empty polyline")
	}
	parts := strings.Split(polyline, " ")
	if len(parts) != 3 {
		t.Errorf("expected 3 points in polyline, got %d", len(parts))
	}
	// First x at padL (10)
	firstX := extractX(t, parts[0])
	if firstX != 10.0 {
		t.Errorf("expected first x=10 (padL), got %.2f", firstX)
	}
	// Last x at w - padR (390)
	lastX := extractX(t, parts[2])
	if lastX != 390.0 {
		t.Errorf("expected last x=390 (w-padR), got %.2f", lastX)
	}
	// Value 10 (min) should have highest y, value 50 (max) should have lowest y
	y0 := extractY(t, parts[0]) // value 10 (min)
	y1 := extractY(t, parts[1]) // value 50 (max)
	if y0 <= y1 {
		t.Errorf("min value should have higher y than max: y0=%.2f, y1=%.2f", y0, y1)
	}
	// Polygon should be longer (has closing corners)
	if len(polygon) <= len(polyline) {
		t.Errorf("polygon should include closing corners")
	}
}

func TestBuildAreaPoints_EqualMinMax(t *testing.T) {
	// When minV == maxV, range is forced to 1 to avoid division by zero
	data := []DataPoint{
		{Label: "A", Value: 5},
		{Label: "B", Value: 5},
	}
	polyline, polygon := buildAreaPoints(data, 400, 200, 10, 10, 10, 20, 5, 5)
	if polyline == "" || polygon == "" {
		t.Fatal("expected non-empty output even with equal min/max")
	}
}

func TestBuildAreaPoints_YBounds(t *testing.T) {
	// All y values should be within [padT, padT + plotH]
	data := []DataPoint{
		{Label: "A", Value: 0},
		{Label: "B", Value: 50},
		{Label: "C", Value: 100},
	}
	padT, padB := 10, 20
	h := 200
	polyline, _ := buildAreaPoints(data, 400, h, 10, 10, padT, padB, 0, 100)
	parts := strings.Split(polyline, " ")
	plotH := float64(h - padT - padB)
	for _, p := range parts {
		y := extractY(t, p)
		if y < float64(padT)-0.01 || y > float64(padT)+plotH+0.01 {
			t.Errorf("y=%.2f outside expected range [%d, %.0f]", y, padT, float64(padT)+plotH)
		}
	}
}

// --- areaDefaultColors ---

func TestAreaDefaultColors_Length(t *testing.T) {
	if len(areaDefaultColors) == 0 {
		t.Fatal("areaDefaultColors should not be empty")
	}
}

// Default colors are theme tokens that survive SanitizeColor unchanged.
func TestAreaDefaultColors_ThemeTokens(t *testing.T) {
	for i, c := range areaDefaultColors {
		if !strings.HasPrefix(c, "var(--color-") {
			t.Errorf("areaDefaultColors[%d] = %q, expected a var(--color-...) theme token", i, c)
		}
		if got := utils.SanitizeColor(c); got != c {
			t.Errorf("areaDefaultColors[%d] = %q is rewritten to %q by SanitizeColor", i, c, got)
		}
	}
}

// --- helpers ---

func extractX(t *testing.T, point string) float64 {
	t.Helper()
	comma := strings.Index(point, ",")
	if comma < 0 {
		t.Fatalf("malformed point (no comma): %q", point)
	}
	x, err := strconv.ParseFloat(point[:comma], 64)
	if err != nil {
		t.Fatalf("cannot parse x from %q: %v", point, err)
	}
	return x
}

func extractY(t *testing.T, point string) float64 {
	t.Helper()
	comma := strings.Index(point, ",")
	if comma < 0 {
		t.Fatalf("malformed point (no comma): %q", point)
	}
	y, err := strconv.ParseFloat(point[comma+1:], 64)
	if err != nil {
		t.Fatalf("cannot parse y from %q: %v", point, err)
	}
	return y
}
