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
	Value       string // Checkbox: checked when Value != ""; Radio/Select: selected option value
	Required    bool
	Disabled    bool
	Error       string // red line under help; also sets HasError on the control
	Success     string // muted line, rendered only when non-empty
	Width       Width  // inside Fieldset: WidthFull spans both columns on desktop
	// Attributes are spread on the control (e.g. hx-post for blur validation).
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
// swap when re-validating a single field.
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

// controlAttrs returns f.Attributes plus `required` when f.Required,
// without mutating the caller's map.
func (f Field) controlAttrs() templ.Attributes {
	if !f.Required {
		return f.Attributes
	}
	out := make(templ.Attributes, len(f.Attributes)+1)
	for k, v := range f.Attributes {
		out[k] = v
	}
	out["required"] = true
	return out
}
