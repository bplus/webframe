package layout

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderTwo(t *testing.T, p TwoColumnProps) string {
	t.Helper()
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), templ.Raw("MAINTEXT"))
	if err := TwoColumn(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestTwoColumn_AsideAfterMainByDefault(t *testing.T) {
	out := renderTwo(t, TwoColumnProps{Aside: templ.Raw("ASIDETEXT")})
	m, a := strings.Index(out, "MAINTEXT"), strings.Index(out, "ASIDETEXT")
	if m < 0 || a < 0 || a < m {
		t.Errorf("aside should follow main: %s", out)
	}
}

func TestTwoColumn_SideLeft(t *testing.T) {
	out := renderTwo(t, TwoColumnProps{Aside: templ.Raw("ASIDETEXT"), Side: SideLeft})
	if !strings.Contains(out, "order-first") {
		t.Errorf("missing order-first in %s", out)
	}
	if strings.Index(out, "ASIDETEXT") < strings.Index(out, "MAINTEXT") {
		t.Errorf("aside should still follow main in the DOM: %s", out)
	}
	if def := renderTwo(t, TwoColumnProps{Aside: templ.Raw("ASIDETEXT")}); strings.Contains(def, "order-first") {
		t.Errorf("default render has order-first: %s", def)
	}
}

func TestTwoColumn_Sticky(t *testing.T) {
	out := renderTwo(t, TwoColumnProps{Aside: templ.Raw("ASIDETEXT"), Sticky: true})
	if !strings.Contains(out, "md:sticky") {
		t.Errorf("missing md:sticky in %s", out)
	}
}

func TestAsideWidthClass(t *testing.T) {
	tests := []struct {
		w    AsideWidth
		want string
	}{
		{AsideS, "md:w-64"},
		{AsideM, "md:w-80"},
		{AsideL, "md:w-96"},
	}
	for _, tt := range tests {
		if got := asideWidthClass(tt.w); got != tt.want {
			t.Errorf("asideWidthClass(%q) = %q, want %q", tt.w, got, tt.want)
		}
	}
}

func TestThreeColumn_RendersAllThree(t *testing.T) {
	var buf bytes.Buffer
	err := ThreeColumn(ThreeColumnProps{Left: templ.Raw("LEFTTEXT"), Right: templ.Raw("RIGHTTEXT")}).
		Render(templ.WithChildren(context.Background(), templ.Raw("CENTERTEXT")), &buf)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	l, c, r := strings.Index(out, "LEFTTEXT"), strings.Index(out, "CENTERTEXT"), strings.Index(out, "RIGHTTEXT")
	if l < 0 || c < 0 || r < 0 || !(l < c && c < r) {
		t.Errorf("want left<center<right: %s", out)
	}
	if !strings.Contains(out, "details-content") {
		t.Errorf("missing details-content fallback in %s", out)
	}
	if n := strings.Count(out, `<summary class="md:hidden"`); n != 2 {
		t.Errorf("summary count = %d, want 2 in %s", n, out)
	}
}

func renderThree(t *testing.T, p ThreeColumnProps) string {
	t.Helper()
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), templ.Raw("CENTERTEXT"))
	if err := ThreeColumn(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestThreeColumn_Labels(t *testing.T) {
	out := renderThree(t, ThreeColumnProps{
		Left: templ.Raw("LEFTTEXT"), Right: templ.Raw("RIGHTTEXT"),
		LeftLabel: "Nav", RightLabel: "Inspector",
	})
	for _, want := range []string{`<summary class="md:hidden">Nav</summary>`, `<summary class="md:hidden">Inspector</summary>`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	for _, bad := range []string{">Left<", ">Right<"} {
		if strings.Contains(out, bad) {
			t.Errorf("default label %q leaked when a label was given: %s", bad, out)
		}
	}
	def := renderThree(t, ThreeColumnProps{Left: templ.Raw("L"), Right: templ.Raw("R")})
	for _, want := range []string{`<summary class="md:hidden">Left</summary>`, `<summary class="md:hidden">Right</summary>`} {
		if !strings.Contains(def, want) {
			t.Errorf("missing default %q in %s", want, def)
		}
	}
}
