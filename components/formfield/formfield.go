// Package formfield renders a labelled form control with its help text
// and validation feedback, and groups such fields into a responsive
// fieldset. The feedback line renders nothing on success unless the
// caller opts in with Field.Success, so a valid field reserves no space.
//
// Every Type submits a plain form field named Field.Name that a handler
// reads with r.FormValue (or r.Form[...], r.FormFile), and every Type
// re-renders from Field.Value as a string, so a failed POST echoes what
// the user sent without parsing it first:
//
//   - text-like types, TypeTextarea, TypeSelect, TypeRadio, TypeHidden:
//     Value is the control's value;
//   - TypeCheckbox, TypeToggle: checked when Value != "", submit "on";
//   - TypeTags (tagsinput): Value is the tags joined with commas, as the
//     widget submits them; Options' labels are the suggestions;
//   - TypeRating (rating, input mode): Value is the chosen star as an
//     integer string ("3"), Max the star count (5 by default); nothing is
//     submitted until a star is chosen;
//   - TypeDate, TypeDateTime, TypeTime (datetimepicker): Value is the
//     native wire format ("2006-01-02", "2006-01-02T15:04", "15:04");
//     MinValue and MaxValue bound it in the same format; the handler
//     parses with datetimepicker.Parse;
//   - TypeFile (filedrop): Accept, Multiple and Preview configure the
//     zone; the form must be multipart (formpage.Props.Multipart); a file
//     input cannot be refilled, so Value is ignored and a re-rendered page
//     shows the zone empty.
//
// The wiring is the same for every Type: the label points at the control
// that receives focus, that control carries aria-describedby naming the
// help line and the Feedback slot, aria-invalid="true" when Error is set,
// and `required` (or aria-required where the submitting control is not
// the one the browser validates); the Feedback slot sits last.
package formfield

import (
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/datetimepicker"
	"github.com/bplus/webframe/components/input"
	"github.com/bplus/webframe/components/rating"
)

// Type selects the control a Field renders. The zero value is a text input.
type Type string

const (
	TypeText     Type = ""
	TypeEmail    Type = "email"
	TypePassword Type = "password"
	TypeNumber   Type = "number"
	// TypeDate, TypeDateTime and TypeTime render datetimepicker (the
	// browser's native picker).
	TypeDate     Type = "date"
	TypeDateTime Type = "datetime"
	TypeTime     Type = "time"
	// TypeFile renders filedrop: a drop zone around a hidden file input.
	// The form must be multipart.
	TypeFile     Type = "file"
	TypeTextarea Type = "textarea"
	TypeSelect   Type = "select"
	TypeCheckbox Type = "checkbox"
	TypeRadio    Type = "radio"
	TypeHidden   Type = "hidden"
	// TypeToggle renders toggle.Toggle (a switch), with the label text
	// inline after it.
	TypeToggle Type = "toggle"
	// TypeTags renders tagsinput. Required only marks the label and sets
	// aria-required (`required` on the text input would block a form whose
	// tags are complete), so enforce it, and MaxTags, on the server.
	TypeTags Type = "tags"
	// TypeRating renders rating in input mode.
	TypeRating Type = "rating"
)

// DefaultRatingMax is the star count of a TypeRating field whose Max is
// zero or negative.
const DefaultRatingMax = rating.DefaultMax

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
	// Value is the control's value as a string; see the package doc for
	// what it means per Type.
	Value string
	// Max is the star count of a TypeRating field; DefaultRatingMax (5)
	// when zero or negative. Other types ignore it.
	Max int
	// MaxTags caps a TypeTags field; 0 means no cap. Other types ignore it.
	MaxTags int
	// MinValue and MaxValue bound a TypeDate, TypeDateTime or TypeTime
	// field (the min/max attributes) in the control's wire format. Other
	// types ignore them.
	MinValue, MaxValue string
	// Accept is the accept attribute of a TypeFile field ("image/*,.pdf").
	Accept string
	// Multiple lets a TypeFile field take more than one file.
	Multiple bool
	// Preview shows image thumbnails in a TypeFile field's file list.
	Preview bool
	// Required marks the label with * and puts `required` on the control.
	// For TypeSelect the submitting control is a hidden input the browser
	// never validates, so enforce it with Validate or a server-side Error.
	Required bool
	Disabled bool
	Error    string // red line under help; also sets HasError on the control
	Success  string // muted line, rendered only when non-empty
	Width    Width  // inside Fieldset: WidthFull spans both columns on desktop
	// Validate is a URL. When set, the control posts just its own
	// name=value there on blur (on change for selects, checkboxes, radios
	// and other non-text controls) and the response replaces the Feedback
	// slot; the handler answers with RespondFeedback.
	Validate string
	// Attributes are spread on the control (for TypeSelect: on the hidden
	// input that carries the value). They override Validate's hx-* defaults.
	Attributes templ.Attributes
}

// ControlID is the id of the rendered control: f.ID, or "field-"+f.Name.
func (f Field) ControlID() string {
	if f.ID != "" {
		return f.ID
	}
	return "field-" + f.Name
}

// FeedbackID is the id of the field's Feedback wrapper. Hand-written
// validation wiring must swap it with hx-swap="outerHTML": innerHTML nests
// the wrapper and duplicates the id.
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

// inputType maps the input-backed Types onto input.Type. Dates, times
// and files render through their own widgets, not here.
func inputType(t Type) input.Type {
	switch t {
	case TypeEmail:
		return input.TypeEmail
	case TypePassword:
		return input.TypePassword
	case TypeNumber:
		return input.TypeNumber
	default:
		return input.TypeText
	}
}

// dateKind maps the date-like Types onto datetimepicker.Kind.
func dateKind(t Type) datetimepicker.Kind {
	switch t {
	case TypeDateTime:
		return datetimepicker.KindDateTime
	case TypeTime:
		return datetimepicker.KindTime
	default:
		return datetimepicker.KindDate
	}
}

// tagValues splits a TypeTags Value on commas, trimming each tag and
// dropping empties, so "" yields no chips and "a, b" yields two.
func (f Field) tagValues() []string {
	var out []string
	for _, t := range strings.Split(f.Value, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// suggestions are the Options' labels, offered by a TypeTags field as
// completions.
func (f Field) suggestions() []string {
	if len(f.Options) == 0 {
		return nil
	}
	out := make([]string, 0, len(f.Options))
	for _, o := range f.Options {
		if o.Label != "" {
			out = append(out, o.Label)
		}
	}
	return out
}

// ratingValue parses a TypeRating Value; anything that is not a positive
// integer lights nothing.
func (f Field) ratingValue() int {
	n, err := strconv.Atoi(strings.TrimSpace(f.Value))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// ratingMax is the star count: Max, or DefaultRatingMax.
func (f Field) ratingMax() int {
	if f.Max <= 0 {
		return DefaultRatingMax
	}
	return f.Max
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

// validateTrigger is the hx-trigger Validate uses: "change" for controls
// that never blur meaningfully (or, for tags, commit on blur anyway), "blur
// changed" otherwise.
func validateTrigger(t Type) string {
	switch t {
	case TypeSelect, TypeCheckbox, TypeRadio, TypeToggle, TypeRating, TypeFile, TypeTags:
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

// widgetAttrs returns the attributes for a control a widget package
// renders: controlAttrs minus `required` (the widgets emit it from their
// own Required prop, or for tags must not carry it), plus aria-invalid when
// invalid and aria-required when announceRequired. Pass invalid=false when
// the widget's own HasError already writes aria-invalid.
func (f Field) widgetAttrs(invalid, announceRequired bool) templ.Attributes {
	out := f.controlAttrs(true)
	delete(out, "required")
	if invalid && f.Error != "" {
		if _, set := out["aria-invalid"]; !set {
			out["aria-invalid"] = "true"
		}
	}
	if announceRequired && f.Required {
		if _, set := out["aria-required"]; !set {
			out["aria-required"] = "true"
		}
	}
	return out
}

// ratingAttrs are the rating fieldset's attributes: widgetAttrs plus
// aria-labelledby pointing at the field's label, since the visible label
// is a form.Label above the stars (the rating's own legend is the hidden
// "Rating").
func (f Field) ratingAttrs() templ.Attributes {
	out := f.widgetAttrs(true, true)
	if f.Label != "" {
		if _, set := out["aria-labelledby"]; !set {
			out["aria-labelledby"] = f.ControlID() + "-label"
		}
	}
	return out
}
