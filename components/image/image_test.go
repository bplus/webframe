package image

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, props ...Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Image(props...).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestImage_Defaults(t *testing.T) {
	out := render(t, Props{Src: "/static/img/a.png"})
	for _, want := range []string{`loading="lazy"`, `decoding="async"`, `alt=""`, `max-w-full`, `h-auto`, `src="/static/img/a.png"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in: %s", want, out)
		}
	}
	for _, bad := range []string{`width=`, `height=`, `object-cover`, `rounded-md`} {
		if strings.Contains(out, bad) {
			t.Errorf("unexpected %s in: %s", bad, out)
		}
	}
	// Zero-value call: alt is still present, src is omitted rather than src="".
	zero := render(t)
	if !strings.Contains(zero, `alt=""`) {
		t.Errorf("zero-value render must still emit alt: %s", zero)
	}
	if strings.Contains(zero, `src=`) {
		t.Errorf("empty Src must omit the src attribute: %s", zero)
	}
}

func TestImage_Size(t *testing.T) {
	out := render(t, Props{Src: "/a.png", Alt: "A", Width: 400, Height: 300})
	for _, want := range []string{`width="400"`, `height="300"`, `alt="A"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in: %s", want, out)
		}
	}
}

func TestImage_Eager(t *testing.T) {
	out := render(t, Props{Src: "/a.png", Eager: true})
	if strings.Contains(out, `loading="lazy"`) {
		t.Errorf("Eager must drop loading=lazy: %s", out)
	}
	if !strings.Contains(out, `decoding="async"`) {
		t.Errorf("Eager keeps decoding=async: %s", out)
	}
}

func TestImage_CoverRounded(t *testing.T) {
	out := render(t, Props{Src: "/a.png", Width: 200, Height: 200, Fit: FitCover, Rounded: true})
	for _, want := range []string{`object-cover`, `rounded-md`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in: %s", want, out)
		}
	}
	if strings.Contains(out, `h-auto`) {
		t.Errorf("cover fit must let the height attribute govern the box, not h-auto: %s", out)
	}
}

func TestImage_SrcSanitized(t *testing.T) {
	out := render(t, Props{Src: "javascript:alert(1)"})
	if !strings.Contains(out, `about:invalid`) {
		t.Errorf("javascript: src must be neutralised to about:invalid: %s", out)
	}
	if strings.Contains(out, `javascript:`) {
		t.Errorf("javascript: leaked into src: %s", out)
	}
}

func TestImage_IDClassAttributes(t *testing.T) {
	out := render(t, Props{ID: "hero", Class: "border", Attributes: templ.Attributes{"data-x": "1"}, Src: "/a.png"})
	for _, want := range []string{`id="hero"`, `border`, `data-x="1"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in: %s", want, out)
		}
	}
}
