package statcomparison

import (
	"strings"
	"testing"
)

// With LowerIsBetter a fall is good news: down is green and up is red. The
// arrow still follows the direction the number moved.
func TestStatComparison_LowerIsBetter(t *testing.T) {
	cases := []struct {
		dir   Direction
		class string
		path  string
	}{
		{DirectionDown, "text-success", "M8 13V3M8 13l4-4M8 13L4 9"},
		{DirectionUp, "text-destructive", "M8 3v10M8 3l4 4M8 3L4 7"},
		{DirectionFlat, "text-muted-foreground", "M2 8h12"},
	}
	for _, tc := range cases {
		t.Run(string(tc.dir), func(t *testing.T) {
			if got := ToneClass(tc.dir, true); got != tc.class {
				t.Errorf("ToneClass(%q, true) = %q, want %q", tc.dir, got, tc.class)
			}
			out := render(t, StatComparison(Props{Label: "Open bugs", Value: "4", Delta: "-3", Direction: tc.dir, LowerIsBetter: true}))
			for _, want := range []string{tc.class, tc.path} {
				if !strings.Contains(out, want) {
					t.Errorf("missing %q in %s", want, out)
				}
			}
		})
	}
}
