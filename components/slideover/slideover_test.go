package slideover

import (
	"bytes"
	"context"
	"io"
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

// panelClass returns the class attribute of the panel div (the element
// carrying data-tui-slideover-panel), so assertions do not match the
// backdrop or the header by accident.
func panelClass(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, "data-tui-slideover-panel")
	if i < 0 {
		t.Fatalf("no panel in %s", out)
	}
	start := strings.LastIndex(out[:i], `class="`)
	if start < 0 {
		t.Fatalf("panel has no class in %s", out)
	}
	start += len(`class="`)
	end := strings.Index(out[start:], `"`)
	return out[start : start+end]
}

func hasClass(classes, want string) bool {
	for _, c := range strings.Fields(classes) {
		if c == want {
			return true
		}
	}
	return false
}

func wantClasses(t *testing.T, classes string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !hasClass(classes, w) {
			t.Errorf("missing class %q in %q", w, classes)
		}
	}
}

func rejectClasses(t *testing.T, classes string, rejects ...string) {
	t.Helper()
	for _, r := range rejects {
		if hasClass(classes, r) {
			t.Errorf("unexpected class %q in %q", r, classes)
		}
	}
}

func TestPanel_SideBottomGeometry(t *testing.T) {
	out := render(t, Panel(PanelProps{Side: SideBottom, Title: "Sheet"}))
	cls := panelClass(t, out)
	wantClasses(t, cls,
		"fixed", "inset-x-0", "bottom-0", "w-full", "rounded-t-xl", "border-t", "border-border",
		"max-h-[60vh]", "flex", "flex-col",
		"data-[tui-slideover-open=false]:translate-y-full",
		"data-[tui-slideover-open=true]:translate-y-0",
	)
	rejectClasses(t, cls,
		"top-0", "h-full", "right-0", "left-0", "border-l", "border-r",
		"w-[480px]", "max-w-[calc(100%-2rem)]",
		"data-[tui-slideover-open=false]:translate-x-full",
		"data-[tui-slideover-open=true]:translate-x-0",
	)
}

func TestPanel_SideBottomIgnoresWidth(t *testing.T) {
	for _, w := range []Width{WidthSm, WidthLg, WidthXl} {
		cls := panelClass(t, render(t, Panel(PanelProps{Side: SideBottom, Width: w})))
		wantClasses(t, cls, "w-full")
		rejectClasses(t, cls, widthClass(w), "max-w-[calc(100%-2rem)]")
	}
}

func TestPanel_HeightTiers(t *testing.T) {
	cases := []struct {
		h    Height
		want string
	}{
		{"", "max-h-[60vh]"},
		{HeightSm, "max-h-[40vh]"},
		{HeightMd, "max-h-[60vh]"},
		{HeightLg, "max-h-[85vh]"},
		{Height("bogus"), "max-h-[60vh]"},
	}
	for _, c := range cases {
		cls := panelClass(t, render(t, Panel(PanelProps{Side: SideBottom, Height: c.h})))
		wantClasses(t, cls, c.want)
		for _, other := range []string{"max-h-[40vh]", "max-h-[60vh]", "max-h-[85vh]"} {
			if other != c.want {
				rejectClasses(t, cls, other)
			}
		}
	}
}

func TestPanel_HeightIgnoredForSides(t *testing.T) {
	for _, s := range []Side{SideLeft, SideRight} {
		cls := panelClass(t, render(t, Panel(PanelProps{Side: s, Height: HeightSm})))
		rejectClasses(t, cls, "max-h-[40vh]", "max-h-[60vh]", "max-h-[85vh]")
		wantClasses(t, cls, "h-full")
	}
}

func TestPanel_DragHandleOnlyForBottom(t *testing.T) {
	bottom := render(t, Panel(PanelProps{Side: SideBottom}))
	for _, want := range []string{"data-tui-slideover-handle", `aria-hidden="true"`, "bg-muted-foreground/30", "rounded-full", "w-10", "h-1.5"} {
		if !strings.Contains(bottom, want) {
			t.Errorf("bottom sheet missing %q in %s", want, bottom)
		}
	}
	// The handle precedes the header.
	if h, hd := strings.Index(bottom, "data-tui-slideover-handle"), strings.Index(bottom, "justify-between"); h > hd {
		t.Errorf("drag handle must come before the header: %s", bottom)
	}
	for _, s := range []Side{"", SideLeft, SideRight} {
		out := render(t, Panel(PanelProps{Side: s}))
		if strings.Contains(out, "data-tui-slideover-handle") {
			t.Errorf("side %q must not render a drag handle: %s", s, out)
		}
	}
}

func TestPanel_RightDefaultUnchanged(t *testing.T) {
	cls := panelClass(t, render(t, Panel()))
	wantClasses(t, cls,
		"fixed", "top-0", "h-full", "right-0", "w-[480px]", "border-l", "border-border",
		"max-w-[calc(100%-2rem)]",
		"data-[tui-slideover-open=false]:translate-x-full",
		"data-[tui-slideover-open=true]:translate-x-0",
	)
	rejectClasses(t, cls, "bottom-0", "inset-x-0", "rounded-t-xl", "border-t",
		"data-[tui-slideover-open=false]:translate-y-full")
}

func TestPanel_LeftUnchanged(t *testing.T) {
	cls := panelClass(t, render(t, Panel(PanelProps{Side: SideLeft, Width: WidthSm})))
	wantClasses(t, cls, "left-0", "w-80", "border-r", "h-full",
		"data-[tui-slideover-open=false]:-translate-x-full",
		"data-[tui-slideover-open=true]:translate-x-0")
	rejectClasses(t, cls, "right-0", "border-l", "bottom-0")
}

func TestPanel_SidesFullWidthBelowSm(t *testing.T) {
	for _, s := range []Side{SideLeft, SideRight} {
		for _, w := range []Width{WidthSm, WidthMd, WidthLg, WidthXl, WidthFull} {
			cls := panelClass(t, render(t, Panel(PanelProps{Side: s, Width: w})))
			wantClasses(t, cls, "max-sm:w-full", "max-sm:max-w-full", widthClass(w))
		}
	}
	// The sheet is already full width; the phone override is not needed there.
	rejectClasses(t, panelClass(t, render(t, Panel(PanelProps{Side: SideBottom}))), "max-sm:w-full")
}

func TestPanel_ZIndexAboveToasts(t *testing.T) {
	for _, s := range []Side{"", SideLeft, SideRight, SideBottom} {
		out := render(t, Panel(PanelProps{Side: s}))
		wantClasses(t, panelClass(t, out), "z-[60]")
		rejectClasses(t, panelClass(t, out), "z-50")
		// Backdrop is the first div.
		backdropEnd := strings.Index(out, "data-tui-slideover-backdrop")
		backdrop := out[:backdropEnd]
		if !strings.Contains(backdrop, "z-[55]") {
			t.Errorf("side %q backdrop missing z-[55]: %s", s, backdrop)
		}
		if strings.Contains(backdrop, "z-50") {
			t.Errorf("side %q backdrop still z-50: %s", s, backdrop)
		}
	}
}

func TestPanel_ClosedStateAttributes(t *testing.T) {
	out := render(t, Panel(PanelProps{Side: SideBottom}))
	if strings.Count(out, `data-tui-slideover-open="false"`) != 2 {
		t.Errorf("backdrop and panel must both start closed: %s", out)
	}
	if strings.Count(out, `data-tui-slideover-hidden="true"`) != 2 {
		t.Errorf("backdrop and panel must both start hidden: %s", out)
	}
}

func TestPanel_OpenFromSlideOverContext(t *testing.T) {
	c := SlideOver(Props{ID: "sheet", Open: true})
	wrapped := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, Panel(PanelProps{Side: SideBottom})), w)
	})
	out := render(t, wrapped)
	if strings.Count(out, `data-tui-slideover-open="true"`) != 2 {
		t.Errorf("open sheet must mark backdrop and panel open: %s", out)
	}
	if !strings.Contains(out, `data-slideover-instance="sheet"`) {
		t.Errorf("panel must inherit the wrapper instance: %s", out)
	}
}
