package dialog

import (
	"bytes"
	"context"
	"io"
	"regexp"
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

var contentTag = regexp.MustCompile(`<div class="([^"]*)" data-tui-dialog-content[^>]*>`)

// contentClasses returns the class list of the dialog panel element.
func contentClasses(t *testing.T, html string) []string {
	t.Helper()
	m := contentTag.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no data-tui-dialog-content element in %s", html)
	}
	return strings.Fields(m[1])
}

func has(classes []string, want string) bool {
	for _, c := range classes {
		if c == want {
			return true
		}
	}
	return false
}

func TestContent_WidthTiers(t *testing.T) {
	all := []string{
		"sm:max-w-lg",
		"sm:max-w-[min(640px,calc(100%-2rem))]",
		"sm:max-w-[min(800px,calc(100%-2rem))]",
		"sm:max-w-[calc(100%-2rem)]",
	}
	cases := []struct {
		width Width
		want  string
	}{
		{WidthDefault, all[0]},
		{WidthLg, all[1]},
		{WidthXl, all[2]},
		{WidthFull, all[3]},
		{Width("unknown"), all[0]},
	}
	for _, tc := range cases {
		classes := contentClasses(t, render(t, Content(ContentProps{ID: "d", Width: tc.width})))
		for _, c := range all {
			if got := has(classes, c); got != (c == tc.want) {
				t.Errorf("Width %q: class %q present = %v, want %v", tc.width, c, got, c == tc.want)
			}
		}
		// Below sm every tier keeps the 1rem margin.
		for _, c := range []string{"w-full", "max-w-[calc(100%-2rem)]"} {
			if !has(classes, c) {
				t.Errorf("Width %q: missing %q", tc.width, c)
			}
		}
	}
}

// A Content with no Width and no Body keeps the width it had before the
// tiers, and the attributes dialog.min.js opens and closes it by.
func TestContent_ZeroValueKeepsWidthAndBehaviour(t *testing.T) {
	zero := render(t, Content(ContentProps{ID: "d"}))
	if zero != render(t, Content(ContentProps{ID: "d", Width: WidthDefault})) {
		t.Error("WidthDefault is not the zero value")
	}
	classes := contentClasses(t, zero)
	for _, c := range []string{"w-full", "max-w-[calc(100%-2rem)]", "sm:max-w-lg"} {
		if !has(classes, c) {
			t.Errorf("missing width class %q in %v", c, classes)
		}
	}
	for _, want := range []string{
		`data-tui-dialog-backdrop data-dialog-instance="d" data-tui-dialog-open="false" data-tui-dialog-hidden="true"`,
		`data-tui-dialog-content data-dialog-instance="d" data-tui-dialog-open="false" data-tui-dialog-hidden="true"`,
		`data-tui-dialog-close="d"`,
		"data-[tui-dialog-open=false]:opacity-0",
		"data-[tui-dialog-open=true]:opacity-100",
		"data-[tui-dialog-hidden=true]:!hidden",
	} {
		if !strings.Contains(zero, want) {
			t.Errorf("missing %q in %s", want, zero)
		}
	}
	open := render(t, Content(ContentProps{ID: "d", Open: true}))
	if !strings.Contains(open, `data-tui-dialog-content data-dialog-instance="d" data-tui-dialog-open="true">`) {
		t.Errorf("an open Content must not be hidden: %s", open)
	}
	if strings.Contains(zero, "data-tui-dialog-body") {
		t.Errorf("Content renders no Body of its own: %s", zero)
	}
}

// Content is a column that is never taller than the screen, so a Body can
// scroll inside it and a wide child cannot widen it.
func TestContent_HeightLimitAndColumn(t *testing.T) {
	classes := contentClasses(t, render(t, Content(ContentProps{ID: "d"})))
	for _, c := range []string{"flex", "flex-col", "max-h-[calc(100dvh-2rem)]", "overflow-y-auto"} {
		if !has(classes, c) {
			t.Errorf("missing %q in %v", c, classes)
		}
	}
	if has(classes, "grid") {
		t.Errorf("the panel must not be a grid whose column grows with its content: %v", classes)
	}
}

func TestBody(t *testing.T) {
	out := render(t, Body(BodyProps{ID: "list"}))
	for _, want := range []string{
		`id="list"`, "min-h-48", "flex-1", "overflow-y-auto",
		"data-tui-dialog-body", `role="region"`, `aria-label="Dialog content"`, `tabindex="0"`,
		"focus-visible:ring-[3px]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	out = render(t, Body(BodyProps{Label: "Spells"}))
	if !strings.Contains(out, `aria-label="Spells"`) {
		t.Errorf("Label must be the aria-label: %s", out)
	}
}

// The Body sits inside the panel between the parts before and after it.
func TestContent_BodyOrder(t *testing.T) {
	inner := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, part := range []templ.Component{Header(), Body(), Footer()} {
			if err := part.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
	out := render(t, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Content(ContentProps{ID: "d"}).Render(templ.WithChildren(ctx, inner), w)
	}))
	panel := strings.Index(out, "data-tui-dialog-content")
	header := strings.Index(out, "text-center sm:text-left")
	body := strings.Index(out, "data-tui-dialog-body")
	footer := strings.Index(out, "flex-col-reverse")
	if !(panel >= 0 && panel < header && header < body && body < footer) {
		t.Errorf("want panel < header < body < footer, got %d %d %d %d in %s", panel, header, body, footer, out)
	}
}
