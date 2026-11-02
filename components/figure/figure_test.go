package figure

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), templ.Raw("<svg>MEDIA</svg>"))
	if err := Figure(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestFigure_Caption(t *testing.T) {
	out := render(t, Props{Caption: "Sword"})
	for _, want := range []string{"<figure", "<figcaption", "Sword", "</figcaption>", "<svg>MEDIA</svg>", "text-muted-foreground", "items-center"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if m, c := strings.Index(out, "MEDIA"), strings.Index(out, "<figcaption"); !(m < c) {
		t.Errorf("caption should follow the media: %s", out)
	}
	empty := render(t, Props{})
	if strings.Contains(empty, "figcaption") {
		t.Errorf("empty Caption must render no figcaption: %s", empty)
	}
	if !strings.Contains(empty, "<figure") || !strings.Contains(empty, "MEDIA") {
		t.Errorf("figure and children still render without a caption: %s", empty)
	}
}

func TestFigure_Align(t *testing.T) {
	start := render(t, Props{Align: AlignStart})
	if !strings.Contains(start, "items-start") || strings.Contains(start, "items-center") {
		t.Errorf("AlignStart should use items-start only: %s", start)
	}
	center := render(t, Props{})
	if !strings.Contains(center, "items-center") {
		t.Errorf("default Align should center: %s", center)
	}
}

func TestFigure_IDClassAttributes(t *testing.T) {
	out := render(t, Props{ID: "fig-1", Class: "mt-2", Attributes: templ.Attributes{"data-x": "y"}})
	for _, want := range []string{`id="fig-1"`, "mt-2", `data-x="y"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}
