package spinner

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

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestSpinner_Defaults(t *testing.T) {
	out := render(t, Spinner())
	mustContain(t, out, `role="status"`, "animate-spin", "h-5", "w-5", "text-muted-foreground", `stroke="currentColor"`, `fill="currentColor"`, `<span class="sr-only">Loading</span>`, `aria-hidden="true"`)
	if strings.Contains(out, "htmx-indicator") {
		t.Errorf("default spinner must not be an htmx indicator: %s", out)
	}
	if !strings.HasPrefix(out, "<span") {
		t.Errorf("root should be a span: %s", out)
	}
}

func TestSpinner_Sizes(t *testing.T) {
	// utils.TwMerge emits classes in no fixed order, so assert each class alone.
	for size, want := range map[Size][]string{SizeSm: {"h-4", "w-4"}, SizeMd: {"h-5", "w-5"}, SizeLg: {"h-8", "w-8"}} {
		out := render(t, Spinner(Props{Size: size}))
		for _, w := range want {
			if !strings.Contains(out, w) {
				t.Errorf("Size %q: missing %q in %s", size, w, out)
			}
		}
	}
}

func TestSpinner_IndicatorAndAttrs(t *testing.T) {
	out := render(t, Spinner(Props{ID: "busy", Class: "text-primary", Indicator: true, Attributes: templ.Attributes{"data-x": "1"}}))
	mustContain(t, out, `id="busy"`, "htmx-indicator", "text-primary", `data-x="1"`)
}

func TestSpinner_LabelEscaped(t *testing.T) {
	out := render(t, Spinner(Props{Label: `Saving <b>"x"</b>`}))
	if strings.Contains(out, "<b>") {
		t.Errorf("label not escaped: %s", out)
	}
	mustContain(t, out, "Saving &lt;b&gt;")
}

func TestInline(t *testing.T) {
	out := render(t, Inline("Saving <x>"))
	mustContain(t, out, `role="status"`, "h-4", "w-4", "animate-spin", "gap-2", "Saving &lt;x&gt;")
	if strings.Contains(out, "sr-only") {
		t.Errorf("Inline shows its label; no sr-only copy expected: %s", out)
	}
}
