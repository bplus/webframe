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
		ScrollSm: "max-h-[min(240px,calc(100dvh-2rem))]",
		ScrollMd: "max-h-[min(400px,calc(100dvh-2rem))]",
		ScrollLg: "max-h-[min(600px,calc(100dvh-2rem))]",
	}
	cases := map[ScrollHeight]string{ScrollSm: all[ScrollSm], ScrollMd: all[ScrollMd], ScrollLg: all[ScrollLg], "unknown": all[ScrollMd]}
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
	if out := renderScroll(t, ScrollProps{}, ""); strings.Contains(out, "aria-label") {
		t.Errorf("no aria-label without Label: %s", out)
	}
}
