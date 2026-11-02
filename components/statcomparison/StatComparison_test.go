package statcomparison

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestStatComparison_Basics(t *testing.T) {
	out := render(t, StatComparison(Props{ID: "nodes", Label: "Total Nodes", Value: "1,234", PrevValue: "1,100"}))
	for _, want := range []string{`id="nodes"`, "Total Nodes", "1,234", "from 1,100"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	for _, unwanted := range []string{"text-success", "text-destructive", "<svg"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("no Delta should render no delta row, found %q in %s", unwanted, out)
		}
	}
}

// Delta colors come from theme tokens (success, destructive, muted-foreground),
// never Tailwind default palette classes.
func TestStatComparison_DeltaColors(t *testing.T) {
	cases := []struct {
		name      string
		direction Direction
		class     string
		path      string
	}{
		{"up", DirectionUp, "text-success", "M8 3v10M8 3l4 4M8 3L4 7"},
		{"down", DirectionDown, "text-destructive", "M8 13V3M8 13l4-4M8 13L4 9"},
		{"flat", DirectionFlat, "text-muted-foreground", "M2 8h12"},
		{"zero", Direction(""), "text-muted-foreground", "M2 8h12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := render(t, StatComparison(Props{Value: "10", Delta: "+1", Direction: tc.direction}))
			for _, want := range []string{"+1", tc.class, tc.path} {
				if !strings.Contains(out, want) {
					t.Errorf("missing %q in %s", want, out)
				}
			}
			for _, unwanted := range []string{"text-green-500", "text-red-500"} {
				if strings.Contains(out, unwanted) {
					t.Errorf("default-palette class %q leaked into %s", unwanted, out)
				}
			}
		})
	}
}

// ToneClass and Arrow are the shared direction→color and direction→icon
// mapping (stattile uses them); the arrow is decorative (aria-hidden).
func TestToneClassAndArrow(t *testing.T) {
	cases := []struct {
		dir   Direction
		class string
		path  string
	}{
		{DirectionUp, "text-success", "M8 3v10M8 3l4 4M8 3L4 7"},
		{DirectionDown, "text-destructive", "M8 13V3M8 13l4-4M8 13L4 9"},
		{DirectionFlat, "text-muted-foreground", "M2 8h12"},
		{Direction(""), "text-muted-foreground", "M2 8h12"},
		{Direction("sideways"), "text-muted-foreground", "M2 8h12"},
	}
	for _, tc := range cases {
		t.Run(string(tc.dir), func(t *testing.T) {
			if got := ToneClass(tc.dir); got != tc.class {
				t.Errorf("ToneClass(%q) = %q, want %q", tc.dir, got, tc.class)
			}
			out := render(t, Arrow(tc.dir))
			for _, want := range []string{"<svg", `aria-hidden="true"`, `stroke="currentColor"`, tc.path} {
				if !strings.Contains(out, want) {
					t.Errorf("Arrow(%q): missing %q in %s", tc.dir, want, out)
				}
			}
			if strings.Count(out, "<svg") != 1 {
				t.Errorf("Arrow(%q) should render exactly one svg: %s", tc.dir, out)
			}
		})
	}
}
