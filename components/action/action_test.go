package action

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/button"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestButton_LinkRendersAnchor(t *testing.T) {
	out := render(t, Button(Action{Label: "Open", Href: "/x"}))
	for _, want := range []string{"<a", `href="/x"`, ">Open<"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestButton_HxPostRendersButton(t *testing.T) {
	out := render(t, Button(Action{Label: "Go", HxPost: "/api/go", HxTarget: "#out"}))
	for _, want := range []string{"<button", `hx-post="/api/go"`, `hx-target="#out"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<a") {
		t.Errorf("unexpected <a in output:\n%s", out)
	}
}

func TestButton_ConfirmEmitsHxConfirm(t *testing.T) {
	out := render(t, Button(Action{Label: "Del", HxDelete: "/d", Confirm: &Confirm{Message: "Delete it?"}}))
	if !strings.Contains(out, `hx-confirm="Delete it?"`) {
		t.Errorf("missing hx-confirm:\n%s", out)
	}
}

func TestButton_VariantMapping(t *testing.T) {
	direct := render(t, button.Button(button.Props{Variant: button.VariantDestructive}))
	if !strings.Contains(direct, "bg-destructive") {
		t.Fatalf("button destructive no longer contains bg-destructive:\n%s", direct)
	}
	out := render(t, Button(Action{Label: "X", Variant: VariantDestructive}))
	if !strings.Contains(out, "bg-destructive") {
		t.Errorf("destructive action missing bg-destructive:\n%s", out)
	}
}

func TestGroup_RendersAllInOrder(t *testing.T) {
	out := render(t, Group([]Action{{Label: "First"}, {Label: "Second"}}))
	i, j := strings.Index(out, "First"), strings.Index(out, "Second")
	if i < 0 || j < 0 || i > j {
		t.Errorf("labels missing or out of order (%d, %d):\n%s", i, j, out)
	}
}

func TestIsLink(t *testing.T) {
	tests := []struct {
		name string
		a    Action
		want bool
	}{
		{"href only", Action{Href: "/x"}, true},
		{"href and HxGet", Action{Href: "/x", HxGet: "/y"}, false},
		{"nothing", Action{}, false},
	}
	for _, tt := range tests {
		if got := tt.a.IsLink(); got != tt.want {
			t.Errorf("%s: IsLink() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestGroup_Wraps(t *testing.T) {
	out := render(t, Group([]Action{{Label: "A"}, {Label: "B"}}))
	if !strings.Contains(out, "flex-wrap") {
		t.Errorf("Group should wrap at narrow widths: %s", out)
	}
}

func TestButton_ConfirmEmitsDialogDataAttributes(t *testing.T) {
	out := render(t, Button(Action{
		Label: "Del", HxDelete: "/d", Variant: VariantDestructive,
		Confirm: &Confirm{Title: "Delete item", Message: "Delete it?", ConfirmLabel: "Delete", CancelLabel: "Keep"},
	}))
	for _, want := range []string{
		`hx-confirm="Delete it?"`,
		`data-wf-confirm-title="Delete item"`,
		`data-wf-confirm-label="Delete"`,
		`data-wf-confirm-cancel="Keep"`,
		`data-wf-confirm-destructive="true"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestButton_ConfirmOmitsEmptyFieldsAndDestructiveFlag(t *testing.T) {
	out := render(t, Button(Action{Label: "Go", HxPost: "/go", Confirm: &Confirm{Message: "Sure?"}}))
	if !strings.Contains(out, `hx-confirm="Sure?"`) {
		t.Errorf("missing hx-confirm:\n%s", out)
	}
	for _, unwanted := range []string{"data-wf-confirm-title", "data-wf-confirm-label", "data-wf-confirm-cancel", "data-wf-confirm-destructive"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("empty Confirm field or non-destructive action emitted %q:\n%s", unwanted, out)
		}
	}
}

func TestButton_NoConfirmEmitsNothing(t *testing.T) {
	out := render(t, Button(Action{Label: "Go", HxPost: "/go", Variant: VariantDestructive}))
	if strings.Contains(out, "hx-confirm") || strings.Contains(out, "data-wf-confirm") {
		t.Errorf("action without Confirm emitted confirm attributes:\n%s", out)
	}
	if len(Action{}.ConfirmAttributes()) != 0 {
		t.Errorf("ConfirmAttributes with nil Confirm should be empty")
	}
}

func TestMenuAttrs_HxConfirmAndCallerAttributes(t *testing.T) {
	attrs := MenuAttrs(Action{
		HxDelete: "/d", HxTarget: "#t", HxSwap: "outerHTML", Variant: VariantDestructive,
		Confirm:    &Confirm{Title: "Delete item", Message: "Delete it?", ConfirmLabel: "Delete", CancelLabel: "Keep"},
		Attributes: templ.Attributes{"data-x": "1", "hx-target": "#caller-wins"},
	})
	want := map[string]string{
		"hx-delete":                   "/d",
		"hx-swap":                     "outerHTML",
		"hx-target":                   "#caller-wins",
		"hx-confirm":                  "Delete it?",
		"data-wf-confirm-title":       "Delete item",
		"data-wf-confirm-label":       "Delete",
		"data-wf-confirm-cancel":      "Keep",
		"data-wf-confirm-destructive": "true",
		"data-x":                      "1",
	}
	if len(attrs) != len(want) {
		t.Errorf("MenuAttrs has %d entries, want %d: %v", len(attrs), len(want), attrs)
	}
	for k, v := range want {
		if got := attrs[k]; got != v {
			t.Errorf("MenuAttrs[%q] = %v, want %q", k, got, v)
		}
	}
	for _, k := range []string{"hx-get", "hx-post", "type", "form"} {
		if _, ok := attrs[k]; ok {
			t.Errorf("MenuAttrs emitted %q for an action without it: %v", k, attrs)
		}
	}
}

func TestButton_SubmitRendersSubmitButton(t *testing.T) {
	out := render(t, Button(Action{Label: "Save", Submit: true, Variant: VariantPrimary}))
	for _, want := range []string{"<button", `type="submit"`, "<span>Save</span>", "bg-primary"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `type="button"`) || strings.Contains(out, "<a") {
		t.Errorf("Submit must render only <button type=\"submit\">:\n%s", out)
	}
}

func TestButton_SubmitWinsOverHref(t *testing.T) {
	a := Action{Label: "Save", Submit: true, Href: "/ignored"}
	if a.IsLink() {
		t.Errorf("IsLink() should be false for a Submit action")
	}
	out := render(t, Button(a))
	if strings.Contains(out, "<a") || strings.Contains(out, `href=`) {
		t.Errorf("Submit action rendered an anchor:\n%s", out)
	}
	if !strings.Contains(out, `type="submit"`) {
		t.Errorf("missing type=\"submit\":\n%s", out)
	}
}

func TestButton_FormAttribute(t *testing.T) {
	out := render(t, Button(Action{Label: "Delete selected", HxPost: "/bulk", Form: "items-bulk"}))
	if !strings.Contains(out, `form="items-bulk"`) {
		t.Errorf("missing form attribute:\n%s", out)
	}
	link := render(t, Button(Action{Label: "Open", Href: "/x", Form: "items-bulk"}))
	if strings.Contains(link, `form=`) {
		t.Errorf("a plain link must not carry a form attribute:\n%s", link)
	}
}

func TestButton_FullWidth(t *testing.T) {
	for name, a := range map[string]Action{
		"button": {Label: "Sign in", Submit: true, FullWidth: true},
		"link":   {Label: "GitHub", Href: "/auth/github", FullWidth: true},
	} {
		if out := render(t, Button(a)); !strings.Contains(out, "w-full") {
			t.Errorf("%s: FullWidth missing w-full:\n%s", name, out)
		}
	}
	if out := render(t, Button(Action{Label: "Plain"})); strings.Contains(out, "w-full") {
		t.Errorf("w-full without FullWidth:\n%s", out)
	}
}

func TestButton_VariantOutline(t *testing.T) {
	direct := render(t, button.Button(button.Props{Variant: button.VariantOutline}))
	if !strings.Contains(direct, "bg-background") {
		t.Fatalf("button outline no longer contains bg-background:\n%s", direct)
	}
	out := render(t, Button(Action{Label: "X", Variant: VariantOutline}))
	for _, want := range []string{"border", "bg-background"} {
		if !strings.Contains(out, want) {
			t.Errorf("outline action missing %q:\n%s", want, out)
		}
	}
}

// TestButton_DefaultAndSecondaryRenderSecondary: a plain Button renders
// both the unset Variant and an explicit VariantSecondary secondary.
func TestButton_DefaultAndSecondaryRenderSecondary(t *testing.T) {
	if VariantDefault == VariantSecondary {
		t.Fatal("VariantSecondary must differ from the zero value")
	}
	for _, v := range []Variant{VariantDefault, VariantSecondary} {
		out := render(t, Button(Action{Label: "Go", Variant: v}))
		if !strings.Contains(out, "bg-secondary") {
			t.Errorf("Variant %q: want secondary:\n%s", v, out)
		}
	}
}
