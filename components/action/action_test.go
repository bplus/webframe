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
