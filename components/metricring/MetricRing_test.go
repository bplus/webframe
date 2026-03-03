package metricring

import (
	"math"
	"testing"
)

// --- circumference() tests ---

func TestCircumference(t *testing.T) {
	tests := []struct {
		name   string
		radius int
		want   int
	}{
		{"radius 34 (default 80px ring)", 34, int(math.Round(2 * math.Pi * 34))},
		{"radius 0", 0, 0},
		{"radius 1", 1, int(math.Round(2 * math.Pi * 1))},
		{"radius 100", 100, int(math.Round(2 * math.Pi * 100))},
		{"radius 50", 50, int(math.Round(2 * math.Pi * 50))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := circumference(tt.radius)
			if got != tt.want {
				t.Errorf("circumference(%d) = %d, want %d", tt.radius, got, tt.want)
			}
		})
	}
}

func TestCircumference_PiPrecision(t *testing.T) {
	// Verify the result is consistent with 2*Pi*r rounded
	r := 34
	expected := 2 * math.Pi * float64(r)
	got := circumference(r)
	diff := math.Abs(expected - float64(got))
	if diff > 0.5 {
		t.Errorf("circumference(%d) = %d, expected ~%.2f, diff %.2f exceeds 0.5", r, got, expected, diff)
	}
}

// --- resolveColor() / resolveTrackColor() tests ---

func TestResolveColor_Empty(t *testing.T) {
	p := Props{}
	got := p.resolveColor()
	if got != "var(--color-primary)" {
		t.Errorf("resolveColor() empty = %q, want default", got)
	}
}

func TestResolveColor_Provided(t *testing.T) {
	p := Props{Color: "#ff0000"}
	got := p.resolveColor()
	if got != "#ff0000" {
		t.Errorf("resolveColor() = %q, want #ff0000", got)
	}
}

func TestResolveColor_Sanitized(t *testing.T) {
	// Malicious input should be sanitized to fallback
	p := Props{Color: "red;background:url(evil)"}
	got := p.resolveColor()
	if got != "#888" {
		t.Errorf("resolveColor() with injection = %q, want #888 (sanitized)", got)
	}
}

func TestResolveTrackColor_Empty(t *testing.T) {
	p := Props{}
	got := p.resolveTrackColor()
	if got != "var(--color-muted)" {
		t.Errorf("resolveTrackColor() empty = %q, want default", got)
	}
}

func TestResolveTrackColor_Provided(t *testing.T) {
	p := Props{TrackColor: "rgb(200, 200, 200)"}
	got := p.resolveTrackColor()
	if got != "rgb(200, 200, 200)" {
		t.Errorf("resolveTrackColor() = %q, want rgb(200, 200, 200)", got)
	}
}

func TestResolveTrackColor_Sanitized(t *testing.T) {
	p := Props{TrackColor: `"onmouseover=alert(1)`}
	got := p.resolveTrackColor()
	if got != "#888" {
		t.Errorf("resolveTrackColor() with injection = %q, want #888 (sanitized)", got)
	}
}

// --- clamp() tests ---

func TestClamp_BelowMin(t *testing.T) {
	got := clamp(-5, 0, 100)
	if got != 0 {
		t.Errorf("clamp(-5, 0, 100) = %d, want 0", got)
	}
}

func TestClamp_AboveMax(t *testing.T) {
	got := clamp(150, 0, 100)
	if got != 100 {
		t.Errorf("clamp(150, 0, 100) = %d, want 100", got)
	}
}

func TestClamp_InRange(t *testing.T) {
	got := clamp(50, 0, 100)
	if got != 50 {
		t.Errorf("clamp(50, 0, 100) = %d, want 50", got)
	}
}

func TestClamp_AtBoundaries(t *testing.T) {
	if got := clamp(0, 0, 100); got != 0 {
		t.Errorf("clamp(0, 0, 100) = %d, want 0", got)
	}
	if got := clamp(100, 0, 100); got != 100 {
		t.Errorf("clamp(100, 0, 100) = %d, want 100", got)
	}
}
