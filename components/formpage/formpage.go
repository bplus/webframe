// Package formpage renders a create/edit form page: a page header, then
// one <form> holding an error summary banner (errors only), one titled
// section per group of fields, and a right-aligned Cancel/Submit row.
// Create and edit are the same page: the caller declares the fields once
// and hands over the current Values (and, after a failed POST, the
// Errors) by field name.
//
// Phone width (390px): the header, banner, sections and submit row stack
// in one column; each Fieldset's grid is one column below md, so every
// field takes the full width; the submit row wraps if the labels are long.
package formpage

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/pageheader"
)

// Section is one titled group of fields, rendered as a layout.Section
// (h3) holding a formfield.Fieldset.
type Section struct {
	Title, Description string
	Fields             []formfield.Field
}

// Props configures a form page. Only Sections and Action are needed for a
// working page; everything else has a usable zero value.
type Props struct {
	// ID is the <form> id and the htmx swap target; "" means "form".
	ID     string
	Header pageheader.Props
	// Action is the submit URL (plain; templ sanitizes it).
	Action string
	// Method is "post" when empty.
	Method string
	// HxPost submits over htmx: the form gets hx-post=Action,
	// hx-target="#"+ID and hx-swap="outerHTML", so an error re-render
	// (Respond on a plain htmx request targeting the form) swaps the
	// whole form in place. Action and Method are still emitted, so the
	// form works without JS.
	HxPost   bool
	Sections []Section
	// Values override Field.Value by Name. A key that is present wins
	// even when its value is "" (that is how an unchecked checkbox or
	// toggle, which submits nothing, comes back unchecked); an absent key
	// leaves Field.Value alone. Keys naming no field are ignored.
	Values map[string]string
	// Errors set Field.Error by Name and drive the summary banner, which
	// renders only when len(Errors) > 0. A key naming no field is still
	// counted and listed (as a form-level error, without a link).
	Errors map[string]string
	// Validate is the default blur-validation URL (formfield.Field.Validate)
	// for every field that has none of its own; "" wires nothing.
	Validate string
	// Hidden renders as hidden inputs at the top of the form (form.Form).
	Hidden map[string]string
	// Submit is the submit button; the zero value is Label "Save" with
	// VariantPrimary. Href and Hx* on it are ignored: it always renders
	// as <button type="submit">.
	Submit action.Action
	// Cancel, when set, renders before Submit (secondary unless given a
	// Variant) and is usually a plain link back to the list.
	Cancel *action.Action
	// Multipart emits enctype="multipart/form-data"; needed for TypeFile.
	Multipart bool
}

// id returns the <form> id with the default applied.
func (p Props) id() string {
	if p.ID == "" {
		return "form"
	}
	return p.ID
}

// resolve applies Values, Errors and the default Validate to one declared
// field, without mutating the caller's slice.
func (p Props) resolve(f formfield.Field) formfield.Field {
	if v, ok := p.Values[f.Name]; ok {
		f.Value = v
	}
	if e, ok := p.Errors[f.Name]; ok {
		f.Error = e
	}
	if f.Validate == "" {
		f.Validate = p.Validate
	}
	return f
}

// fields returns the resolved fields of one section.
func (p Props) fields(s Section) []formfield.Field {
	out := make([]formfield.Field, len(s.Fields))
	for i, f := range s.Fields {
		out[i] = p.resolve(f)
	}
	return out
}

// Field returns the declared field named name with Values, Errors and the
// default Validate applied, or false when no section declares it.
func (p Props) Field(name string) (formfield.Field, bool) {
	for _, s := range p.Sections {
		for _, f := range s.Fields {
			if f.Name == name {
				return p.resolve(f), true
			}
		}
	}
	return formfield.Field{}, false
}

// formProps builds the form.Form Props: Action/Method/Hidden/Multipart
// always, the htmx trio when HxPost is set.
func (p Props) formProps() form.Props {
	fp := form.Props{
		ID:        p.id(),
		Action:    p.Action,
		Method:    p.Method,
		Hidden:    p.Hidden,
		Multipart: p.Multipart,
	}
	if p.HxPost {
		fp.HxPost = p.Action
		fp.HxTarget = "#" + p.id()
		fp.HxSwap = "outerHTML"
	}
	return fp
}

// errorItem is one line of the summary banner.
type errorItem struct {
	Label, Message string
	Href           string // "#"+ControlID(), or "" for an error naming no field
}

// errorItems lists the errors for the banner: field errors in declared
// order (label from the field, falling back to its name), then errors
// whose key names no field, sorted by key so output is deterministic.
func (p Props) errorItems() []errorItem {
	if len(p.Errors) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(p.Errors))
	var items []errorItem
	for _, s := range p.Sections {
		for _, f := range s.Fields {
			msg, ok := p.Errors[f.Name]
			if !ok || seen[f.Name] {
				continue
			}
			seen[f.Name] = true
			label := f.Label
			if label == "" {
				label = f.Name
			}
			items = append(items, errorItem{Label: label, Message: msg, Href: "#" + f.ControlID()})
		}
	}
	var rest []string
	for k := range p.Errors {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		items = append(items, errorItem{Message: p.Errors[k]})
	}
	return items
}

// bannerTitle is the summary line: "Please fix the N errors below."
// (singular for one).
func (p Props) bannerTitle() string {
	n := len(p.Errors)
	if n == 1 {
		return "Please fix the 1 error below."
	}
	return "Please fix the " + strconv.Itoa(n) + " errors below."
}

// submit returns the Submit action with the zero-value defaults applied.
func (p Props) submit() action.Action {
	a := p.Submit
	if a.Label == "" {
		a.Label = "Save"
	}
	if a.Variant == "" {
		a.Variant = action.VariantPrimary
	}
	return a
}

// submitVariant maps an action Variant onto the button Variant, as
// action.Button does (its mapping is unexported).
func submitVariant(v action.Variant) button.Variant {
	switch v {
	case action.VariantDestructive:
		return button.VariantDestructive
	case action.VariantGhost:
		return button.VariantGhost
	case action.VariantSecondary:
		return button.VariantSecondary
	default:
		return button.VariantDefault
	}
}

// submitButtonProps renders Submit as a real submit button. action.Button
// always emits type="button" (so a toolbar button never submits a form by
// accident), which is exactly wrong for the one button that must, so the
// submit row builds button.Props itself with Type submit.
func (p Props) submitButtonProps() button.Props {
	a := p.submit()
	return button.Props{
		ID:         a.ID,
		Type:       button.TypeSubmit,
		Variant:    submitVariant(a.Variant),
		Size:       a.Size,
		Disabled:   a.Disabled,
		Attributes: a.Attributes,
	}
}

// Respond writes the form page for r through the appshell App installed by
// (*App).Middleware, as a page type's handler does: the whole document for
// a normal request (plain GET/POST, boosted navigation, history restore),
// the bare page body (header plus form) for a plain htmx request with any
// other target, and the <form> alone (Body) for a plain htmx request whose
// HX-Target is the form's id, which is what an HxPost submit sends, so the
// error re-render swaps the form in place.
//
// Respond never redirects. The POST handler decides: on validation errors
// it sets p.Values from the submitted form and p.Errors, then calls Respond
// (status 200, so htmx performs the swap); on success it redirects to the
// saved record (303 See Other, or for an htmx submit an HX-Redirect header
// with an empty 200 body) and renders no success message.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	return app.RenderParts(w, r, title, Page(p), map[string]templ.Component{p.id(): Body(p)})
}
