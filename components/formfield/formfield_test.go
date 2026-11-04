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

// TestFormFieldRadioGroupHasControlID: the radiogroup carries ControlID() so
// an error-summary link to "#"+ControlID() lands on the group; the radios
// themselves are ControlID()+"-"+i.
func TestFormFieldRadioGroupHasControlID(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "size", Label: "Size", Type: TypeRadio,
		Options: []Option{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}},
	}))
	mustContain(t, out, `id="field-size" role="radiogroup"`, `id="field-size-0"`, `id="field-size-1"`)
	if n := strings.Count(out, `id="field-size"`); n != 1 {
		t.Errorf("got %d id=\"field-size\", want 1: %s", n, out)
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

// requiredAttr matches the boolean `required` attribute, not aria-required
// or a Tailwind variant.
var requiredAttr = regexp.MustCompile(`\srequired[\s/>]`)

func TestField_Tags(t *testing.T) {
	out := render(t, FormField(Field{
		Name: "tags", Label: "Tags", Type: TypeTags, Value: "go, templ,,htmx", Help: "Comma separated",
		Required: true, Placeholder: "Add a tag", Options: []Option{{Label: "Go"}, {Label: "Rust"}},
	}))
	// The chips come from the comma-split Value; the hidden field re-joins them.
	mustContain(t, out, `data-tag="go"`, `data-tag="templ"`, `data-tag="htmx"`,
		`<input type="hidden" name="tags" value="go,templ,htmx"`)
	if n := strings.Count(out, "<li data-tag="); n != 3 {
		t.Errorf("got %d chips, want 3: %s", n, out)
	}
	// The label targets the text input, which is f.ControlID() and carries the
	// aria wiring merged with the widget's own hint id.
	mustContain(t, out, `for="field-tags"`)
	text := tag(t, out, `<input id="field-tags"[^>]*>`)
	mustContain(t, text, `type="text"`, `placeholder="Add a tag"`,
		`aria-describedby="field-tags-hint field-tags-help field-tags-feedback"`, `aria-required="true"`,
		`list="field-tags-options"`)
	// `required` must not land on the text input: it is not what submits.
	if requiredAttr.MatchString(text) {
		t.Errorf("required must not be on the tags text input: %s", text)
	}
	if strings.Contains(text, `aria-invalid="true"`) {
		t.Errorf("unexpected aria-invalid without Error: %s", text)
	}
	// Options' labels are the datalist suggestions.
	mustContain(t, out, `<datalist id="field-tags-options"><option value="Go"></option><option value="Rust"></option></datalist>`)
	mustContain(t, out, `id="field-tags-help"`, `id="field-tags-feedback"`)
}

func TestField_TagsErrorAndEmpty(t *testing.T) {
	out := render(t, FormField(Field{Name: "tags", Label: "Tags", Type: TypeTags, Error: "Too many"}))
	text := tag(t, out, `<input id="field-tags"[^>]*>`)
	if n := strings.Count(text, `aria-invalid="true"`); n != 1 {
		t.Errorf("got %d aria-invalid on the text input, want 1: %s", n, text)
	}
	mustContain(t, out, "Too many", `role="alert"`, `<input type="hidden" name="tags" value=""`)
	if strings.Contains(out, "<li data-tag=") {
		t.Errorf("unexpected chips for an empty Value: %s", out)
	}
	if strings.Contains(out, "<datalist") {
		t.Errorf("unexpected datalist without Options: %s", out)
	}
}

func TestField_Rating(t *testing.T) {
	out := render(t, FormField(Field{Name: "stars", Label: "Stars", Type: TypeRating, Value: "3", Max: 4, Help: "Be honest", Required: true}))
	fs := tag(t, out, `<fieldset[^>]*>`)
	// The fieldset is the control: it carries the id, the label link and the aria state.
	mustContain(t, fs, `id="field-stars"`, `aria-labelledby="field-stars-label"`,
		`aria-describedby="field-stars-help field-stars-feedback"`, `aria-required="true"`)
	if strings.Contains(fs, `aria-invalid="true"`) || requiredAttr.MatchString(fs) {
		t.Errorf("fieldset must not carry aria-invalid or required here: %s", fs)
	}
	mustContain(t, out, `id="field-stars-label"`, "Stars")
	if n := strings.Count(out, `type="radio"`); n != 4 {
		t.Errorf("got %d radios, want Max=4: %s", n, out)
	}
	if n := len(checkedAttr.FindAllString(out, -1)); n != 1 {
		t.Errorf("got %d checked, want 1: %s", n, out)
	}
	mustContain(t, tag(t, out, `<input[^>]*value="3"[^>]*>`), "checked")
	// Required reaches the radio group through the widget (first radio).
	mustContain(t, tag(t, out, `<input[^>]*value="1"[^>]*>`), " required")
	mustContain(t, out, `id="field-stars-help"`, `id="field-stars-feedback"`)
}

func TestField_RatingDefaultsAndError(t *testing.T) {
	out := render(t, FormField(Field{Name: "stars", Label: "Stars", Type: TypeRating, Value: "junk", Error: "Pick one"}))
	if n := strings.Count(out, `type="radio" data-star`); n != DefaultRatingMax {
		t.Errorf("got %d star radios, want %d: %s", n, DefaultRatingMax, out)
	}
	if checkedAttr.MatchString(out) {
		t.Errorf("a non-numeric Value must check nothing: %s", out)
	}
	mustContain(t, tag(t, out, `<fieldset[^>]*>`), `aria-invalid="true"`)
	mustContain(t, out, "Pick one", `role="alert"`)
}

func TestField_DateTimeKinds(t *testing.T) {
	cases := []struct {
		ty    Type
		typ   string
		value string
	}{
		{TypeDate, "date", "2026-10-31"},
		{TypeDateTime, "datetime-local", "2026-10-31T09:30"},
		{TypeTime, "time", "09:30"},
	}
	for _, tc := range cases {
		out := render(t, FormField(Field{Name: "when", Label: "When", Type: tc.ty, Value: tc.value, MinValue: "min-v", MaxValue: "max-v", Help: "Local", Required: true}))
		in := tag(t, out, `<input[^>]*>`)
		mustContain(t, in, `id="field-when"`, `type="`+tc.typ+`"`, `name="when"`, `value="`+tc.value+`"`,
			`min="min-v"`, `max="max-v"`, `aria-describedby="field-when-help field-when-feedback"`, "scheme-light")
		if n := len(requiredAttr.FindAllString(in, -1)); n != 1 {
			t.Errorf("%s: got %d required, want 1: %s", tc.ty, n, in)
		}
		if strings.Contains(in, `aria-invalid="true"`) {
			t.Errorf("%s: unexpected aria-invalid without Error: %s", tc.ty, in)
		}
		mustContain(t, out, `for="field-when"`, `id="field-when-help"`, `id="field-when-feedback"`)
	}
}

func TestField_DateError(t *testing.T) {
	out := render(t, FormField(Field{Name: "when", Label: "When", Type: TypeDate, Error: "Too early"}))
	in := tag(t, out, `<input[^>]*>`)
	if n := strings.Count(in, `aria-invalid="true"`); n != 1 {
		t.Errorf("got %d aria-invalid, want 1: %s", n, in)
	}
	for _, bad := range []string{"min=", "max=", "value="} {
		if strings.Contains(in, bad) {
			t.Errorf("unexpected %s on an empty field: %s", bad, in)
		}
	}
	mustContain(t, out, "Too early", `role="alert"`)
}

func TestField_File(t *testing.T) {
	out := render(t, FormField(Field{Name: "art", Label: "Artwork", Type: TypeFile, Accept: "image/*", Multiple: true, Preview: true, Required: true, Help: "PNG"}))
	// f.ControlID() is the hidden file input, the labelled control; the
	// Alpine root is ControlID()+"-root".
	mustContain(t, out, `id="field-art-root"`, `for="field-art"`)
	if strings.Contains(out, "Choose the files again") {
		t.Errorf("no ChooseAgain line without Value or Error: %s", out)
	}
	in := tag(t, out, `<input type="file"[^>]*>`)
	mustContain(t, in, `id="field-art"`, `name="art"`, `accept="image/*"`, " multiple",
		`aria-describedby="field-art-help field-art-feedback"`)
	if n := len(requiredAttr.FindAllString(in, -1)); n != 1 {
		t.Errorf("got %d required, want 1: %s", n, in)
	}
	// A Value cannot refill the input; it asks for the file again.
	withValue := render(t, FormField(Field{Name: "art", Label: "Artwork", Type: TypeFile, Value: "ignored.png"}))
	if strings.Contains(tag(t, withValue, `<input type="file"[^>]*>`), "value=") || strings.Contains(withValue, "ignored.png") {
		t.Errorf("a file input cannot carry a value: %s", withValue)
	}
	mustContain(t, withValue, "Choose the file again")
	if strings.Contains(in, `aria-invalid="true"`) {
		t.Errorf("unexpected aria-invalid without Error: %s", in)
	}
	// Preview turns on the thumbnail template; a single-file field without
	// Preview has none.
	mustContain(t, out, `x-if="f.url"`, "Drop files here or choose")
	single := render(t, FormField(Field{Name: "art", Label: "Artwork", Type: TypeFile}))
	if strings.Contains(single, `x-if="f.url"`) || strings.Contains(tag(t, single, `<input type="file"[^>]*>`), " multiple") {
		t.Errorf("unexpected preview/multiple on a plain file field: %s", single)
	}
	mustContain(t, single, "Drop a file here or choose")
}

func TestField_FileError(t *testing.T) {
	out := render(t, FormField(Field{Name: "art", Label: "Artwork", Type: TypeFile, Error: "Too big"}))
	in := tag(t, out, `<input type="file"[^>]*>`)
	if n := strings.Count(in, `aria-invalid="true"`); n != 1 {
		t.Errorf("got %d aria-invalid, want 1: %s", n, in)
	}
	// The zone styles itself from the input's aria-invalid.
	mustContain(t, out, "has-aria-invalid:border-destructive", "Too big", `role="alert"`)
	// The file the user chose is gone after the failed POST: say so.
	mustContain(t, out, "Choose the file again")
	mustContain(t, in, `aria-describedby="field-art-again field-art-feedback"`)
}

// TestField_FileControlIDIsInput: Field.ControlID() resolves to the
// <input type="file">, so a caller's "#"+ControlID() link or label lands
// on the control, not on a wrapper.
func TestField_FileControlIDIsInput(t *testing.T) {
	f := Field{Name: "art", Label: "Artwork", Type: TypeFile}
	out := render(t, FormField(f))
	el := tag(t, out, `<[a-z]+ [^>]*id="`+f.ControlID()+`"[^>]*>`)
	if !strings.HasPrefix(el, `<input type="file"`) {
		t.Errorf("id %q is on %s, want the file input", f.ControlID(), el)
	}
}

func TestField_WidgetValidateUsesChange(t *testing.T) {
	for _, ty := range []Type{TypeTags, TypeRating, TypeFile} {
		out := render(t, FormField(Field{Name: "x", Label: "X", Type: ty, Validate: "/v"}))
		mustContain(t, out, `hx-trigger="change"`, `hx-post="/v"`, `hx-target="#field-x-feedback"`, `hx-swap="outerHTML"`, `hx-params="x"`)
	}
	for _, ty := range []Type{TypeDate, TypeDateTime, TypeTime} {
		out := render(t, FormField(Field{Name: "x", Label: "X", Type: ty, Validate: "/v"}))
		mustContain(t, out, `hx-trigger="blur changed"`, `hx-post="/v"`)
	}
	// The rating's hx-* sit on the fieldset (change bubbles from the radios).
	mustContain(t, tag(t, render(t, FormField(Field{Name: "x", Label: "X", Type: TypeRating, Validate: "/v"})), `<fieldset[^>]*>`), `hx-post="/v"`)
}

func TestField_WidgetAttributesReachControl(t *testing.T) {
	attrs := templ.Attributes{"data-x": "1"}
	mustContain(t, tag(t, render(t, FormField(Field{Name: "a", Label: "A", Type: TypeTags, Attributes: attrs})), `<input id="field-a"[^>]*>`), `data-x="1"`)
	mustContain(t, tag(t, render(t, FormField(Field{Name: "a", Label: "A", Type: TypeRating, Attributes: attrs})), `<fieldset[^>]*>`), `data-x="1"`)
	mustContain(t, tag(t, render(t, FormField(Field{Name: "a", Label: "A", Type: TypeDate, Attributes: attrs})), `<input[^>]*>`), `data-x="1"`)
	mustContain(t, tag(t, render(t, FormField(Field{Name: "a", Label: "A", Type: TypeFile, Attributes: attrs})), `<input type="file"[^>]*>`), `data-x="1"`)
}

func TestField_WidgetsDisabled(t *testing.T) {
	disabledAttr := regexp.MustCompile(`\sdisabled[\s/>]`)
	for _, ty := range []Type{TypeTags, TypeRating, TypeDate, TypeFile} {
		out := render(t, FormField(Field{Name: "x", Label: "X", Type: ty, Disabled: true}))
		if !disabledAttr.MatchString(out) {
			t.Errorf("%s: Disabled did not reach the widget: %s", ty, out)
		}
	}
}

// TestField_TagsMaxTags: MaxTags reaches tagsinput as its cap (data-max on
// the root and the "n/max" count); zero means no cap (data-max="0").
func TestField_TagsMaxTags(t *testing.T) {
	out := render(t, FormField(Field{Name: "tags", Label: "Tags", Type: TypeTags, Value: "a,b", MaxTags: 5}))
	mustContain(t, out, `id="field-tags-root"`, `data-max="5"`, "2/5")
	out = render(t, FormField(Field{Name: "tags", Label: "Tags", Type: TypeTags, Value: "a,b"}))
	mustContain(t, out, `data-max="0"`)
	if strings.Contains(out, "2/0") {
		t.Errorf("count shown without a cap: %s", out)
	}
}
