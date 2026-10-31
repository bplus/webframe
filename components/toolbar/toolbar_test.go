package toolbar

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
)

func render(t *testing.T, p Props, children templ.Component) string {
	t.Helper()
	ctx := context.Background()
	if children != nil {
		ctx = templ.WithChildren(ctx, children)
	}
	var buf bytes.Buffer
	if err := Toolbar(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// indexOf returns the index of needle in s, failing the test when absent.
func indexOf(t *testing.T, s, needle string) int {
	t.Helper()
	i := strings.Index(s, needle)
	if i < 0 {
		t.Fatalf("missing %q in %s", needle, s)
	}
	return i
}

func TestToolbar_RoleToolbar(t *testing.T) {
	out := render(t, Props{ID: "tb", Class: "mt-4", Attributes: templ.Attributes{"data-x": "1"}}, nil)
	for _, want := range []string{`role="toolbar"`, `id="tb"`, `mt-4`, `data-x="1"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestToolbar_LeftRightOrder(t *testing.T) {
	out := render(t, Props{
		Left:  []action.Action{{Label: "LEFTACT", Href: "/l"}},
		Right: []action.Action{{Label: "RIGHTACT", HxPost: "/r"}},
	}, templ.Raw("MIDDLECHILD"))
	// The full (desktop) rendering comes first; take the first occurrence of each.
	l := indexOf(t, out, "LEFTACT")
	m := indexOf(t, out, "MIDDLECHILD")
	r := indexOf(t, out, "RIGHTACT")
	if !(l < m && m < r) {
		t.Errorf("order want left < middle < right, got %d %d %d in %s", l, m, r, out)
	}
	if !strings.Contains(out, `hx-post="/r"`) {
		t.Errorf("missing hx-post in %s", out)
	}
}

func TestToolbar_OverflowMenu(t *testing.T) {
	out := render(t, Props{
		Overflow: 1,
		Right: []action.Action{
			{Label: "FIRST", Href: "/1"},
			{Label: "SECOND", HxPost: "/2", HxTarget: "#results", Confirm: &action.Confirm{Message: "Sure?"}},
			{Label: "THIRD", HxDelete: "/3"},
		},
	}, nil)
	phoneStart := indexOf(t, out, "md:hidden")
	phone := out[phoneStart:]
	more := indexOf(t, phone, `aria-label="More actions"`)
	first := indexOf(t, phone, "FIRST")
	second := indexOf(t, phone, "SECOND")
	third := indexOf(t, phone, "THIRD")
	if !(first < more) {
		t.Errorf("FIRST should be inline before the menu: first=%d more=%d\n%s", first, more, phone)
	}
	if !(more < second && second < third) {
		t.Errorf("SECOND and THIRD should follow the menu trigger: more=%d second=%d third=%d\n%s", more, second, third, phone)
	}
	// Items in the menu keep their hx-* attributes and confirm.
	menu := phone[more:]
	for _, want := range []string{`hx-post="/2"`, `hx-target="#results"`, `hx-confirm="Sure?"`, `hx-delete="/3"`, `role="menuitem"`} {
		if !strings.Contains(menu, want) {
			t.Errorf("missing %q in menu %s", want, menu)
		}
	}
	// Desktop rendering shows all three inline.
	desktop := out[:phoneStart]
	for _, want := range []string{"FIRST", "SECOND", "THIRD", "md:flex"} {
		if !strings.Contains(desktop, want) {
			t.Errorf("missing %q in desktop rendering %s", want, desktop)
		}
	}
}

func TestToolbar_OverflowDefaultIsOne(t *testing.T) {
	out := render(t, Props{
		Left: []action.Action{{Label: "A", Href: "/a"}, {Label: "B", Href: "/b"}},
	}, nil)
	if !strings.Contains(out, `aria-label="More actions"`) {
		t.Errorf("Overflow 0 should mean 1: expected a menu for the 2nd action in %s", out)
	}
}

func TestToolbar_NoMenuWhenActionsFit(t *testing.T) {
	out := render(t, Props{
		Overflow: 2,
		Left:     []action.Action{{Label: "A", Href: "/a"}, {Label: "B", Href: "/b"}},
	}, nil)
	if strings.Contains(out, "More actions") {
		t.Errorf("2 actions with Overflow 2 need no menu: %s", out)
	}
}

func TestToolbar_NoCollapse(t *testing.T) {
	out := render(t, Props{
		Overflow: -1,
		Left:     []action.Action{{Label: "A", Href: "/a"}, {Label: "B", Href: "/b"}, {Label: "C", Href: "/c"}},
	}, nil)
	if strings.Contains(out, "More actions") {
		t.Errorf("Overflow -1 must never collapse: %s", out)
	}
	if strings.Contains(out, "md:hidden") || strings.Contains(out, "md:flex") {
		t.Errorf("Overflow -1 should render one width-independent row: %s", out)
	}
}

func TestToolbar_OverflowMenuDestructiveConfirm(t *testing.T) {
	out := render(t, Props{
		Overflow: 1,
		Right: []action.Action{
			{Label: "FIRST", Href: "/1"},
			{Label: "DELETE", HxDelete: "/2", Variant: action.VariantDestructive,
				Confirm: &action.Confirm{Title: "Delete all", Message: "Sure?", ConfirmLabel: "Delete", CancelLabel: "Keep"}},
		},
	}, nil)
	menu := out[indexOf(t, out, `aria-label="More actions"`):]
	for _, want := range []string{`role="menuitem"`, `hx-delete="/2"`, `hx-confirm="Sure?"`, `data-wf-confirm-title="Delete all"`, `data-wf-confirm-label="Delete"`, `data-wf-confirm-cancel="Keep"`, `data-wf-confirm-destructive="true"`} {
		if !strings.Contains(menu, want) {
			t.Errorf("missing %q in menu %s", want, menu)
		}
	}
}
