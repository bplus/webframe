// Package formpage renders a create/edit form page: a page header, then
// one <form> holding an error summary banner (errors only), one titled
// section per group of fields, and a right-aligned Cancel/Submit row.
// Create and edit are the same page: the caller declares the fields once
// and hands over the current Values (and, after a failed POST, the
// Errors) by field name.
package formpage

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/pageheader"
)

// Section is one group of fields, rendered as a layout.Section (h3)
// holding a formfield.Fieldset of Fields, then Content when set. A section
// with no Title and no Description has no heading, so it renders as a plain
// group without the section's vertical padding and divider.
type Section struct {
	Title, Description string
	Fields             []formfield.Field
	// Content, when non-nil, renders after the fieldset, inside the <form>,
	// so a widget that is not a formfield.Field (a keyvalueeditor, a note)
	// is submitted with the fields. A section with Content and no Fields
	// renders only the Content.
	Content templ.Component
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
	// HxPost submits over htmx and swaps the whole form in place on an
	// error re-render. Action and Method are still emitted, so the form
	// works without JS.
	HxPost   bool
	Sections []Section
	// Values override Field.Value by Name. A present key wins even when
	// its value is "" (that is how an unchecked checkbox comes back
	// unchecked); an absent key leaves Field.Value alone.
	Values map[string]string
	// Errors set Field.Error by Name and drive the summary banner. A key
	// naming no field is listed as a form-level error.
	Errors map[string]string
	// Validate is the default blur-validation URL (formfield.Field.Validate)
	// for every field that has none of its own; "" wires nothing.
	Validate string
	// Hidden renders as hidden inputs at the top of the form (form.Form).
	Hidden map[string]string
	// Submit is the submit button; the zero value is a primary "Save".
	// Href and Hx* on it are ignored.
	Submit action.Action
	// Cancel, when set, renders before Submit (secondary unless given a
	// Variant) and is usually a plain link back to the list.
	Cancel *action.Action
	// Multipart emits enctype="multipart/form-data"; required for a
	// TypeFile field. Its handler must call r.ParseMultipartForm before
	// reading values: after a plain r.ParseForm, r.FormValue returns "" for
	// every field in the body.
	Multipart bool
}

// id returns the <form> id with the default applied.
func (p Props) id() string {
	if p.ID == "" {
		return "form"
	}
	return p.ID
}

// ResolveField returns a copy of f with values, errors and the default
// validate URL applied, as described on Props.Values, Props.Errors and
// Props.Validate. Other form-shaped pages (settingspage) use it too.
func ResolveField(f formfield.Field, values, errors map[string]string, validate string) formfield.Field {
	if v, ok := values[f.Name]; ok {
		f.Value = v
	}
	if e, ok := errors[f.Name]; ok {
		f.Error = e
	}
	if f.Validate == "" {
		f.Validate = validate
	}
	return f
}

// resolve applies Values, Errors and the default Validate to one declared
// field, without mutating the caller's slice.
func (p Props) resolve(f formfield.Field) formfield.Field {
	return ResolveField(f, p.Values, p.Errors, p.Validate)
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

// allFields returns every declared field in section order, unresolved.
func (p Props) allFields() []formfield.Field {
	var out []formfield.Field
	for _, s := range p.Sections {
		out = append(out, s.Fields...)
	}
	return out
}

// errorItem is one line of the summary banner.
type errorItem struct {
	Label, Message string
	Href           string // "#"+ControlID(), or "" for an error naming no field
}

// errorItems lists the errors for the banner: field errors in declared
// order (label from the field, falling back to its name), then errors
// whose key names no field, sorted by key so output is deterministic.
func errorItems(fields []formfield.Field, errors map[string]string) []errorItem {
	if len(errors) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(errors))
	var items []errorItem
	for _, f := range fields {
		msg, ok := errors[f.Name]
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
	var rest []string
	for k := range errors {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	for _, k := range rest {
		items = append(items, errorItem{Message: errors[k]})
	}
	return items
}

// bannerTitle is the summary line: "Please fix the error below." for one
// error, "Please fix the N errors below." for more.
func bannerTitle(n int) string {
	if n == 1 {
		return "Please fix the error below."
	}
	return "Please fix the " + strconv.Itoa(n) + " errors below."
}

// Submit returns a as a submit button: Label defaults to defaultLabel,
// Variant to primary, and Href and the Hx* fields are cleared because the
// form, not the button, issues the request. settingspage uses it too.
func Submit(a action.Action, defaultLabel string) action.Action {
	if a.Label == "" {
		a.Label = defaultLabel
	}
	if a.Variant == "" {
		a.Variant = action.VariantPrimary
	}
	a.Submit = true
	a.Href, a.HxGet, a.HxPost, a.HxDelete, a.HxTarget, a.HxSwap = "", "", "", "", "", ""
	return a
}

// submit returns the Submit action with the zero-value defaults applied
// ("Save", primary) and marked Submit, as Props.Submit promises.
func (p Props) submit() action.Action {
	return Submit(p.Submit, "Save")
}

// Respond writes the form page through the appshell App installed by
// (*App).Middleware: the whole document for a normal request, the page body
// for an htmx request, and the <form> alone (Body) for an htmx request
// whose HX-Target is the form, which is what an HxPost submit sends.
//
// Respond never redirects. On validation errors the POST handler sets
// p.Values and p.Errors and calls Respond, which answers 422 (appshell
// configures htmx to swap error responses). On success the handler
// redirects itself, typically with a flash.Success.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	status := 0
	if len(p.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	return app.RenderPartsStatus(w, r, status, title, Page(p), map[string]templ.Component{p.id(): Body(p)})
}
