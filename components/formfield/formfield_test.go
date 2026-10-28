package formfield

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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

func TestFormFieldDescribedBy(t *testing.T) {
	withHelp := render(t, FormField(Field{Name: "email", Label: "Email", Help: "Work address"}))
	mustContain(t, withHelp, `id="field-email-help"`, `aria-describedby="field-email-help field-email-feedback"`)
	noHelp := render(t, FormField(Field{Name: "email", Label: "Email"}))
	mustContain(t, noHelp, `aria-describedby="field-email-feedback"`)
	if strings.Contains(noHelp, "field-email-help") {
		t.Errorf("unexpected help id without Help: %s", noHelp)
	}
	for _, ty := range []Type{TypeTextarea, TypeSelect, TypeCheckbox, TypeRadio} {
		out := render(t, FormField(Field{Name: "x", Label: "X", Type: ty, Options: []Option{{Label: "A", Value: "a"}}}))
		mustContain(t, out, `aria-describedby="field-x-feedback"`)
	}
}

func TestFeedbackErrorIsAlert(t *testing.T) {
	mustContain(t, render(t, Feedback("f", "Bad", "")), `role="alert"`)
	if out := render(t, Feedback("f", "", "")); strings.Contains(out, `role="alert"`) {
		t.Errorf("unexpected role=alert without error: %s", out)
	}
}

func TestFormFieldTextareaDisabledSuccess(t *testing.T) {
	mustContain(t, render(t, FormField(Field{Name: "bio", Label: "Bio", Type: TypeTextarea})), `<textarea`, `id="field-bio"`)
	if !regexp.MustCompile(`\sdisabled[\s/>]`).MatchString(render(t, FormField(Field{Name: "e", Label: "E", Disabled: true}))) {
		t.Errorf("Disabled did not reach the control")
	}
	mustContain(t, render(t, FormField(Field{Name: "e", Label: "E", Success: "Looks good"})), "Looks good")
}

// tag returns the first element start tag whose opening matches re (a
// regexp over the whole tag, e.g. `<button[^>]*>`), or fails the test.
func tag(t *testing.T, out, re string) string {
	t.Helper()
	m := regexp.MustCompile(re).FindString(out)
	if m == "" {
		t.Fatalf("no tag matching %s in %s", re, out)
	}
	return m
}

func TestField_ValidateEmitsHx(t *testing.T) {
	out := render(t, FormField(Field{Name: "email", Label: "Email", Type: TypeEmail, Validate: "/validate/email"}))
	in := tag(t, out, `<input[^>]*>`)
	mustContain(t, in,
		`hx-post="/validate/email"`,
		`hx-trigger="blur changed"`,
		`hx-target="#field-email-feedback"`,
		`hx-swap="outerHTML"`,
		`hx-params="email"`)
	// Without Validate no hx-* attributes appear at all.
	plain := render(t, FormField(Field{Name: "email", Label: "Email", Type: TypeEmail}))
	if strings.Contains(plain, "hx-") {
		t.Errorf("unexpected hx-* without Validate: %s", plain)
	}
}

func TestField_ValidateSelectUsesChange(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "pick", Label: "Pick", Type: TypeSelect, Validate: "/v",
		Options: []Option{{Label: "A", Value: "a"}},
	}))
	hidden := tag(t, out, `<input[^>]*type="hidden"[^>]*>`)
	mustContain(t, hidden, `hx-post="/v"`, `hx-trigger="change"`, `hx-target="#field-pick-feedback"`, `hx-swap="outerHTML"`)
	if strings.Contains(tag(t, out, `<button[^>]*>`), "hx-") {
		t.Errorf("hx-* must stay off the button: %s", out)
	}
}

func TestField_ValidateChangeTypes(t *testing.T) {
	for _, ty := range []Type{TypeCheckbox, TypeRadio, TypeToggle} {
		out := render(t, FormField(Field{Name: "x", Label: "X", Type: ty, Validate: "/v", Options: []Option{{Label: "A", Value: "a"}}}))
		mustContain(t, out, `hx-trigger="change"`, `hx-post="/v"`, `hx-target="#field-x-feedback"`, `hx-swap="outerHTML"`)
	}
	for _, ty := range []Type{TypeText, TypeTextarea, TypeNumber} {
		out := render(t, FormField(Field{Name: "x", Label: "X", Type: ty, Validate: "/v"}))
		mustContain(t, out, `hx-trigger="blur changed"`)
	}
}

func TestField_ValidateCallerAttributesWin(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "email", Label: "Email", Validate: "/v",
		Attributes: templ.Attributes{"hx-trigger": "keyup changed delay:300ms"},
	}))
	mustContain(t, out, `hx-trigger="keyup changed delay:300ms"`, `hx-post="/v"`)
	if strings.Contains(out, `hx-trigger="blur changed"`) {
		t.Errorf("caller hx-trigger must win: %s", out)
	}
}

func TestField_Toggle(t *testing.T) {
	on := render(t, FormField(Field{Name: "notify", Label: "Notify me", Type: TypeToggle, Value: "on", Help: "Weekly digest", Required: true}))
	in := tag(t, on, `<input[^>]*>`)
	mustContain(t, in, `role="switch"`, `type="checkbox"`, `id="field-notify"`, `name="notify"`,
		`aria-describedby="field-notify-help field-notify-feedback"`, `aria-required="true"`)
	if !checkedAttr.MatchString(in) {
		t.Errorf("missing checked in %s", in)
	}
	mustContain(t, on, "Notify me", `id="field-notify-help"`, `id="field-notify-feedback"`)
	// The Toggle root is itself a <label>; no second label element.
	if n := strings.Count(on, "<label"); n != 1 {
		t.Errorf("got %d <label, want 1: %s", n, on)
	}
	off := render(t, FormField(Field{Name: "notify", Label: "Notify me", Type: TypeToggle, Error: "Pick one"}))
	if checkedAttr.MatchString(tag(t, off, `<input[^>]*>`)) {
		t.Errorf("unexpected checked in %s", off)
	}
	mustContain(t, off, `aria-invalid="true"`, "Pick one", `role="alert"`)
	if strings.Contains(off, "aria-required") {
		t.Errorf("unexpected aria-required without Required: %s", off)
	}
}

func TestField_SelectAriaOnButton(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "pick", Label: "Pick", Type: TypeSelect, Required: true, Error: "Required", Help: "Choose",
		Options: []Option{{Label: "A", Value: "a"}},
	}))
	btn := tag(t, out, `<button[^>]*>`)
	mustContain(t, btn, `aria-describedby="field-pick-help field-pick-feedback"`, `aria-invalid="true"`, `aria-required="true"`)
	hidden := tag(t, out, `<input[^>]*type="hidden"[^>]*>`)
	for _, bad := range []string{"aria-describedby", "aria-invalid", "aria-required"} {
		if strings.Contains(hidden, bad) {
			t.Errorf("%s must not be on the hidden input: %s", bad, hidden)
		}
	}
	mustContain(t, hidden, `name="pick"`)
	// Not required, no error: no aria-required, no aria-invalid="true".
	ok := render(t, FormField(Field{Name: "pick", Label: "Pick", Type: TypeSelect, Options: []Option{{Label: "A", Value: "a"}}}))
	okBtn := tag(t, ok, `<button[^>]*>`)
	if strings.Contains(okBtn, "aria-required") || strings.Contains(okBtn, `aria-invalid="true"`) {
		t.Errorf("unexpected aria state on a plain select: %s", okBtn)
	}
}

func TestRespondFeedback_Error(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/validate/email", nil)
	if err := RespondFeedback(rec, req, Field{Name: "email"}, "Email is taken"); err != nil {
		t.Fatalf("RespondFeedback: %v", err)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	mustContain(t, rec.Body.String(), `id="field-email-feedback"`, "Email is taken", `role="alert"`)
}

func TestRespondFeedback_Valid(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/validate/email", nil)
	if err := RespondFeedback(rec, req, Field{Name: "email"}, ""); err != nil {
		t.Fatalf("RespondFeedback: %v", err)
	}
	body := rec.Body.String()
	mustContain(t, body, `id="field-email-feedback"`, "empty:hidden")
	if len(body) >= 80 {
		t.Errorf("valid feedback is %d bytes, want < 80: %s", len(body), body)
	}
	if strings.Contains(body, "alert") {
		t.Errorf("valid feedback must not alert: %s", body)
	}
}
