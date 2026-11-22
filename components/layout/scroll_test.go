package layout

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderScroll(t *testing.T, p ScrollProps, child string) string {
	t.Helper()
	kid := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, child)
		return err
	})
	var buf bytes.Buffer
	if err := Scroll(p).Render(templ.WithChildren(context.Background(), kid), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestScroll_HeightTiers(t *testing.T) {
	all := map[ScrollHeight]string{
		ScrollS: "max-h-[min(240px,calc(100dvh-2rem))]",
		ScrollM: "max-h-[min(400px,calc(100dvh-2rem))]",
		ScrollL: "max-h-[min(600px,calc(100dvh-2rem))]",
	}
	cases := map[ScrollHeight]string{ScrollS: all[ScrollS], ScrollM: all[ScrollM], ScrollL: all[ScrollL], "unknown": all[ScrollM]}
	for h, want := range cases {
		out := renderScroll(t, ScrollProps{MaxHeight: h}, "")
		for _, c := range all {
			if got := strings.Contains(out, c); got != (c == want) {
				t.Errorf("MaxHeight %q: class %q present = %v, want %v in %s", h, c, got, c == want, out)
			}
		}
	}
}

func TestScroll_Region(t *testing.T) {
	out := renderScroll(t, ScrollProps{ID: "rows", Label: "Rows"}, "CHILD")
	for _, want := range []string{`id="rows"`, "overflow-y-auto", `role="region"`, `aria-label="Rows"`, `tabindex="0"`, "focus-visible:ring-[3px]", ">CHILD</div>"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	for _, want := range []string{"-mx-1", "px-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q (room for the children's focus rings) in %s", want, out)
		}
	}
	// Without a Label the box is not a region, but it still takes the focus.
	out = renderScroll(t, ScrollProps{}, "")
	for _, not := range []string{"aria-label", "role="} {
		if strings.Contains(out, not) {
			t.Errorf("unexpected %q without Label: %s", not, out)
		}
	}
	if !strings.Contains(out, `tabindex="0"`) {
		t.Errorf("missing tabindex without Label: %s", out)
	}
}
