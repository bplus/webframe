// Package formfield renders a labelled form control with its help text
// and validation feedback, and groups such fields into a responsive
// fieldset. The feedback line renders nothing on success unless the
// caller opts in with Field.Success, so a valid field reserves no space.
package formfield

import (
	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/input"
)

// Type selects the control a Field renders. The zero value is a text input.
type Type string

const (
	TypeText     Type = ""
	TypeEmail    Type = "email"
	TypePassword Type = "password"
	TypeNumber   Type = "number"
	TypeDate     Type = "date"
	TypeDateTime Type = "datetime"
	TypeFile     Type = "file"
	TypeTextarea Type = "textarea"
	TypeSelect   Type = "select"
	TypeCheckbox Type = "checkbox"
	TypeRadio    Type = "radio"
	TypeHidden   Type = "hidden"
	// TypeToggle renders toggle.Toggle (a switch); Value != "" means
	// checked, and the switch submits "on" when checked. The label text
	// sits inline after the switch, inside the Toggle's own <label>.
	TypeToggle Type = "toggle"
)

// Width controls how many Fieldset grid columns a field spans on
// desktop. The zero value spans the full row.
type Width string

const (
	WidthFull Width = ""
	WidthHalf Width = "half"
)

// Option is one choice for TypeSelect and TypeRadio.
type Option struct{ Label, Value string }

// Field describes one form field. Name is required; everything else is
// optional.
type Field struct {
	ID          string // control id; default "field-" + Name
	Name        string // required
	Label       string
	Type        Type
	Options     []Option // Select and Radio
	Help        string   // muted line under the control
	Placeholder string
	Value       string // Checkbox/Toggle: checked when Value != ""; Radio/Select: selected option value
	// Required marks the label with * and puts `required` on the control,
	// which the browser enforces when the field sits inside form.Form (or
	// any <form>). For TypeSelect the control that submits is a hidden
	// input the browser never validates, so there Validate or a
	// server-side Error is the enforcement; aria-required on the select's
	// button still announces the requirement.
	Required bool
	Disabled bool
	Error    string // red line under help; also sets HasError on the control
	Success  string // muted line, rendered only when non-empty
	Width    Width  // inside Fieldset: WidthFull spans both columns on desktop
	// Validate is a URL. When set, the control gets hx-post={Validate},
	// hx-trigger="blur changed" (TypeSelect, TypeCheckbox, TypeRadio,
	// TypeToggle: "change"), hx-target="#"+FeedbackID() and
	// hx-swap="outerHTML" and hx-params={Name}, so even inside a form.Form
	// (where htmx would otherwise post every field) only this control's
	// name=value is sent; the handler reads r.FormValue(f.Name) and answers
	// with RespondFeedback. Caller Attributes still win on conflict.
	Validate string
	// Attributes are spread on the control (for TypeSelect: on the hidden
	// input that carries the value; aria-* for the select go on the button
	// automatically). They are merged after Validate's hx-* defaults, so a
	// caller's hx-trigger overrides the default.
	Attributes templ.Attributes
}

// ControlID is the id of the rendered control: f.ID, or "field-"+f.Name.
func (f Field) ControlID() string {
	if f.ID != "" {
		return f.ID
	}
	return "field-" + f.Name
}

// FeedbackID is the id of the field's Feedback wrapper, the target to
// swap when re-validating a single field. The swap MUST be
// hx-swap="outerHTML" targeting "#"+FeedbackID(): innerHTML nests the
// wrapper, duplicates the id and defeats `empty:hidden`. Setting
// Validate emits exactly that wiring; hand-written Attributes must
// match it. For TypeSelect the Attributes land on a hidden input that
// never blurs, so use hx-trigger="change" there.
func (f Field) FeedbackID() string {
	return f.ControlID() + "-feedback"
}

// FieldsetProps configures Fieldset.
type FieldsetProps struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Legend string
	Fields []Field
}

// inputType maps the input-backed Types onto input.Type.
func inputType(t Type) input.Type {
	switch t {
	case TypeEmail:
		return input.TypeEmail
	case TypePassword:
		return input.TypePassword
	case TypeNumber:
		return input.TypeNumber
	case TypeDate:
		return input.TypeDate
	case TypeDateTime:
		return input.TypeDateTime
	case TypeFile:
		return input.TypeFile
	default:
		return input.TypeText
	}
}

// HelpID is the id of the field's help line (rendered only when Help is
// set).
func (f Field) HelpID() string {
	return f.ControlID() + "-help"
}

// describedBy is the control's aria-describedby value: the help line
// (when Help is set) and the Feedback wrapper, which always exists, so
// the reference never dangles.
func (f Field) describedBy() string {
	if f.Help != "" {
		return f.HelpID() + " " + f.FeedbackID()
	}
	return f.FeedbackID()
}

// validateTrigger is the hx-trigger Validate uses for a Type: "change"
// for the controls that never blur meaningfully (the select's hidden
// input, checkboxes, radios, the switch), "blur changed" otherwise.
func validateTrigger(t Type) string {
	switch t {
	case TypeSelect, TypeCheckbox, TypeRadio, TypeToggle:
		return "change"
	default:
		return "blur changed"
	}
}

// validateAttrs returns the hx-* wiring Validate implies, or nil when
// Validate is empty.
func (f Field) validateAttrs() templ.Attributes {
	if f.Validate == "" {
		return nil
	}
	return templ.Attributes{
		"hx-post":    f.Validate,
		"hx-trigger": validateTrigger(f.Type),
		"hx-target":  "#" + f.FeedbackID(),
		"hx-swap":    "outerHTML",
		"hx-params":  f.Name,
	}
}

// controlAttrs returns the attributes for the submitting control: the
// Validate hx-* defaults, then f.Attributes (which win on conflict),
// then `required` when f.Required and, when describe is true,
// aria-describedby (unless the caller set one). The caller's map is
// never mutated.
func (f Field) controlAttrs(describe bool) templ.Attributes {
	out := make(templ.Attributes, len(f.Attributes)+6)
	for k, v := range f.validateAttrs() {
		out[k] = v
	}
	for k, v := range f.Attributes {
		out[k] = v
	}
	if f.Required {
		out["required"] = true
	}
	if _, set := out["aria-describedby"]; describe && !set {
		out["aria-describedby"] = f.describedBy()
	}
	return out
}

// ariaStateAttrs returns aria-describedby plus aria-invalid="true" when
// Error is set and aria-required="true" when Required: the announced
// state for controls whose submitting element is not the focused one
// (the select's button) or whose component has no HasError prop (the
// toggle's switch).
func (f Field) ariaStateAttrs() templ.Attributes {
	out := templ.Attributes{"aria-describedby": f.describedBy()}
	if f.Error != "" {
		out["aria-invalid"] = "true"
	}
	if f.Required {
		out["aria-required"] = "true"
	}
	return out
}

// toggleAttrs returns the switch's attributes: controlAttrs (name/value
// are Toggle props; required, hx-*, caller Attributes) plus the aria
// state, since toggle.Toggle has no HasError prop of its own.
func (f Field) toggleAttrs() templ.Attributes {
	out := f.controlAttrs(true)
	for k, v := range f.ariaStateAttrs() {
		if _, set := out[k]; !set {
			out[k] = v
		}
	}
	return out
}
