package sparkline

import (
	"strconv"
	"strings"
	"testing"
	"webframe/utils"
)

// --- buildPolylinePoints ---

func TestBuildPolylinePoints_Empty(t *testing.T) {
	got := buildPolylinePoints(nil, 80, 24)
	if got != "" {
		t.Errorf("expected empty string for nil data, got %q", got)
	}
	got = buildPolylinePoints([]float64{}, 80, 24)
	if got != "" {
		t.Errorf("expected empty string for empty slice, got %q", got)
	}
}

func TestBuildPolylinePoints_SinglePoint(t *testing.T) {
	got := buildPolylinePoints([]float64{5}, 80, 24)
	if got == "" {
		t.Fatal("expected non-empty string for single point")
	}
	// Single point: should be exactly one coordinate pair (no spaces)
	if strings.Contains(got, " ") {
		t.Errorf("expected single point with no spaces, got %q", got)
	}
	// x should be at padding
	x := extractX(t, got)
	if x != float64(sparklinePadding) {
		t.Errorf("expected x at padding=%d, got %.2f", sparklinePadding, x)
	}
}

func TestBuildPolylinePoints_MultiplePoints(t *testing.T) {
	data := []float64{0, 5, 10}
	got := buildPolylinePoints(data, 80, 24)
	if got == "" {
		t.Fatal("expected non-empty points string")
	}
	parts := strings.Split(got, " ")
	if len(parts) != 3 {
		t.Errorf("expected 3 points, got %d: %q", len(parts), got)
	}
	// First x at padding, last x at w - padding
	firstX := extractX(t, parts[0])
	lastX := extractX(t, parts[2])
	if firstX != float64(sparklinePadding) {
		t.Errorf("expected first x=%.0f, got %.2f", float64(sparklinePadding), firstX)
	}
	if lastX != float64(80-sparklinePadding) {
		t.Errorf("expected last x=%.0f, got %.2f", float64(80-sparklinePadding), lastX)
	}
}

func TestBuildPolylinePoints_AllSameValues(t *testing.T) {
	data := []float64{7, 7, 7, 7}
	got := buildPolylinePoints(data, 80, 24)
	if got == "" {
		t.Fatal("expected non-empty points string")
	}
	parts := strings.Split(got, " ")
	if len(parts) != 4 {
		t.Fatalf("expected 4 points, got %d", len(parts))
	}
	// All y values should be identical (flat line)
	firstY := extractY(t, parts[0])
	for i := 1; i < len(parts); i++ {
		y := extractY(t, parts[i])
		if y != firstY {
			t.Errorf("expected all y values equal for constant data, point %d has y=%.2f, want %.2f", i, y, firstY)
		}
	}
}

func TestBuildPolylinePoints_AllZeroValues(t *testing.T) {
	data := []float64{0, 0, 0}
	got := buildPolylinePoints(data, 80, 24)
	if got == "" {
		t.Fatal("expected non-empty points string")
	}
	parts := strings.Split(got, " ")
	if len(parts) != 3 {
		t.Errorf("expected 3 points, got %d", len(parts))
	}
}

func TestBuildPolylinePoints_NegativeValues(t *testing.T) {
	data := []float64{-10, -5, 0, 5, 10}
	got := buildPolylinePoints(data, 100, 50)
	if got == "" {
		t.Fatal("expected non-empty points string")
	}
	parts := strings.Split(got, " ")
	if len(parts) != 5 {
		t.Errorf("expected 5 points, got %d", len(parts))
	}
	// Min value (-10) should have highest y (bottom), max (10) lowest y (top)
	firstY := extractY(t, parts[0])
	lastY := extractY(t, parts[4])
	if firstY <= lastY {
		t.Errorf("min value should have higher y than max: first_y=%.2f last_y=%.2f", firstY, lastY)
	}
}

func TestBuildPolylinePoints_MonotonicallyIncreasing(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}
	got := buildPolylinePoints(data, 100, 50)
	parts := strings.Split(got, " ")
	if len(parts) != 5 {
		t.Fatalf("expected 5 points, got %d", len(parts))
	}
	// Increasing data should produce decreasing y (SVG y-axis is inverted)
	prevY := extractY(t, parts[0])
	for i := 1; i < len(parts); i++ {
		y := extractY(t, parts[i])
		if y > prevY {
			t.Errorf("expected decreasing y for increasing data at point %d: y=%.2f > prev=%.2f", i, y, prevY)
		}
		prevY = y
	}
}

// --- sparklinePadding constant ---

func TestSparklinePaddingConstant(t *testing.T) {
	if sparklinePadding != 2 {
		t.Errorf("expected sparklinePadding=2, got %d", sparklinePadding)
	}
}

func TestBuildPolylinePoints_RespectsPadding(t *testing.T) {
	// With w=80, h=24, padding=2:
	// x range should be [2, 78], y range should be [2, 22]
	data := []float64{0, 10}
	got := buildPolylinePoints(data, 80, 24)
	parts := strings.Split(got, " ")
	if len(parts) != 2 {
		t.Fatalf("expected 2 points, got %d", len(parts))
	}

	firstX := extractX(t, parts[0])
	lastX := extractX(t, parts[1])
	if firstX != float64(sparklinePadding) {
		t.Errorf("expected first x=%d, got %.2f", sparklinePadding, firstX)
	}
	expectedLastX := float64(80 - sparklinePadding)
	if lastX != expectedLastX {
		t.Errorf("expected last x=%.0f, got %.2f", expectedLastX, lastX)
	}

	// y values should be within [padding, h - padding]
	for _, p := range parts {
		y := extractY(t, p)
		if y < float64(sparklinePadding) || y > float64(24-sparklinePadding) {
			t.Errorf("y=%.2f outside padded range [%d, %d]", y, sparklinePadding, 24-sparklinePadding)
		}
	}
}

// --- utils.SanitizeColor ---

func TestSanitizeColor_ValidHex(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"#fff", "#fff"},
		{"#aabbcc", "#aabbcc"},
		{"#3b82f6", "#3b82f6"},
		{"#000000", "#000000"},
	}
	for _, tc := range cases {
		got := utils.SanitizeColor(tc.input)
		if got != tc.want {
			t.Errorf("SanitizeColor(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeColor_ValidNames(t *testing.T) {
	cases := []string{"red", "blue", "currentColor", "transparent"}
	for _, c := range cases {
		got := utils.SanitizeColor(c)
		if got != c {
			t.Errorf("SanitizeColor(%q) = %q, want passthrough", c, got)
		}
	}
}

func TestSanitizeColor_InvalidStrings(t *testing.T) {
	// The regex allows alphanumeric, #, (), commas, dots, spaces, %, _, -
	// It blocks semicolons, quotes, and other injection characters
	cases := []string{
		"red; background: url(evil)", // semicolon blocked
		`"red"`,                      // quotes blocked
		"red'",                       // single quote blocked
		"red\nblue",                  // newline blocked
	}
	for _, c := range cases {
		got := utils.SanitizeColor(c)
		if got != "#888" {
			t.Errorf("SanitizeColor(%q) = %q, want fallback #888", c, got)
		}
	}
}

func TestSanitizeColor_Empty(t *testing.T) {
	got := utils.SanitizeColor("")
	if got != "#888" {
		t.Errorf("SanitizeColor(%q) = %q, want fallback #888", "", got)
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
