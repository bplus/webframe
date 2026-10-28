package formfield

import (
	"bytes"
	"context"
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

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

// checkedAttr matches the boolean `checked` attribute, not Tailwind
// variants like `checked:bg-primary` or `peer-checked:` in class lists.
var checkedAttr = regexp.MustCompile(`\schecked[\s/>]`)

func TestFormFieldTextRendersLabelControlHelp(t *testing.T) {
	out := render(t, FormField(Field{Name: "email", Label: "Email", Type: TypeEmail, Help: "Work address"}))
	mustContain(t, out,
		`for="field-email"`, `id="field-email"`, `type="email"`, `name="email"`,
		`Work address`, `id="field-email-feedback"`)
}

func TestFormFieldRequiredMarksLabelAndControl(t *testing.T) {
	out := render(t, FormField(Field{Name: "email", Label: "Email", Required: true}))
	mustContain(t, out, `aria-hidden="true"`)
	if !regexp.MustCompile(`\srequired[\s/>]`).MatchString(out) {
		t.Errorf("missing required attribute in %s", out)
	}
}

func TestFormFieldErrorRendersMessageAndHasError(t *testing.T) {
	out := render(t, FormField(Field{Name: "email", Label: "Email", Error: "Email is required"}))
	// input.Props{HasError: true} emits aria-invalid="true" on the <input>.
	mustContain(t, out, "Email is required", `aria-invalid="true"`)
}

func TestFeedbackEmptyRendersOnlyWrapper(t *testing.T) {
	out := render(t, Feedback("f", "", ""))
	mustContain(t, out, "empty:hidden", `id="f"`, "data-wf-feedback")
	if len(out) >= 80 {
		t.Errorf("empty feedback is %d bytes, want < 80: %s", len(out), out)
	}
	if !strings.HasSuffix(out, "></div>") {
		t.Errorf("empty feedback must have no children: %s", out)
	}
}

func TestFeedbackEscapesError(t *testing.T) {
	out := render(t, Feedback("f", "<b>x</b>", ""))
	mustContain(t, out, "&lt;b&gt;")
	if strings.Contains(out, "<b>") {
		t.Errorf("unescaped error in %s", out)
	}
}

func TestFeedbackSuccessOnlyWhenSet(t *testing.T) {
	out := render(t, Feedback("f", "", "Saved"))
	mustContain(t, out, "Saved")
	if empty := render(t, Feedback("f", "", "")); strings.Contains(empty, "<p") {
		t.Errorf("unexpected <p in %s", empty)
	}
}

func TestFormFieldSelectMarksSelected(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "pick", Label: "Pick", Type: TypeSelect, Value: "b",
		Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}},
	}))
	mustContain(t, out,
		`data-tui-selectbox-value="b" data-tui-selectbox-selected="true"`,
		`data-tui-selectbox-value="a" data-tui-selectbox-selected="false"`)
}

func TestFormFieldRadioGroup(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "size", Label: "Size", Type: TypeRadio, Value: "a",
		Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}},
	}))
	mustContain(t, out, `role="radiogroup"`, `aria-labelledby="field-size-label"`, `id="field-size-label"`)
	if n := strings.Count(out, `type="radio"`); n != 2 {
		t.Errorf("got %d radios, want 2: %s", n, out)
	}
	if n := len(checkedAttr.FindAllString(out, -1)); n != 1 {
		t.Errorf("got %d checked, want 1: %s", n, out)
	}
}

func TestFormFieldCheckbox(t *testing.T) {
	on := render(t, FormField(Field{Name: "agree", Label: "Agree", Type: TypeCheckbox, Value: "on"}))
	if n := strings.Count(on, `type="checkbox"`); n != 1 {
		t.Errorf("got %d checkboxes, want 1: %s", n, on)
	}
	if !checkedAttr.MatchString(on) {
		t.Errorf("missing checked in %s", on)
	}
	off := render(t, FormField(Field{Name: "agree", Label: "Agree", Type: TypeCheckbox}))
	if checkedAttr.MatchString(off) {
		t.Errorf("unexpected checked in %s", off)
	}
}

func TestFormFieldHiddenRendersOnlyInput(t *testing.T) {
	out := render(t, FormField(Field{Name: "token", Label: "Token", Type: TypeHidden, Value: "xyz"}))
	mustContain(t, out, `type="hidden"`, `name="token"`, `value="xyz"`)
	if strings.Contains(out, "<label") {
		t.Errorf("unexpected <label in %s", out)
	}
}

func TestFieldsetGridAndFullWidth(t *testing.T) {
	out := render(t, Fieldset(FieldsetProps{
		Legend: "Profile",
		Fields: []Field{
			{Name: "bio", Label: "Bio"},
			{Name: "city", Label: "City", Width: WidthHalf},
		},
	}))
	mustContain(t, out, "<legend", "md:grid-cols-2")
	if n := strings.Count(out, "md:col-span-2"); n != 1 {
		t.Errorf("got %d md:col-span-2, want 1: %s", n, out)
	}
}

func TestFormFieldAttributesReachControl(t *testing.T) {
	out := render(t, FormField(Field{Name: "email", Label: "Email", Attributes: templ.Attributes{"hx-post": "/v"}}))
	mustContain(t, out, `hx-post="/v"`)
}
