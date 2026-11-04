// Package settingspage renders a settings screen: a page header, then one
// <form> holding an error summary banner and one h2 section per Group of
// fields, then optional After content (such as a "Danger zone"), and last a
// save bar stuck to the bottom of the viewport with Cancel and Submit.
//
// The save bar is the page's last child, not the form's: its submit button
// carries form=<form id>, so an htmx error re-render that swaps the form
// leaves the bar and After in place. Field resolution, the banner and the
// submit button are formpage's own, so a settings page and a form page
// behave identically on a round trip.
//
// On a phone everything stacks in one column and the save bar's bottom
// padding includes the safe-area inset. The page root carries the
// wf-settings class, under which focusable controls get a
// scroll-margin-bottom so a control scrolled into view lands clear of the
// bar.
package settingspage

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/formpage"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/utils"
)

// Group is one titled group of settings, rendered as a layout.Section
// (h2, with ID as the section id when set) holding a formfield.Fieldset of
// Fields, then Content when set.
type Group struct {
	ID, Title, Description string
	Fields                 []formfield.Field
	// Content, when non-nil, renders inside the section after the fieldset,
	// inside the <form>, so a widget that is not a formfield.Field is posted
	// with the fields. A group with no Fields renders only the Content.
	Content templ.Component
}

// Props configures a settings page. Only Groups and Action are needed for
// a working page; everything else has a usable zero value.
type Props struct {
	// ID is the <form> id and the htmx swap target; "" means "settings".
	// Any id works: Respond and the save bar's submit button both use it.
	// The save bar's submit button names it in its form attribute.
	ID string
	// Class merges into the <form>'s classes; Attributes spread on it.
	Class      string
	Attributes templ.Attributes

	Header pageheader.Props
	// After, when set, renders after the form and before the save bar, for
	// content that is not a group of fields, such as a "Danger zone"
	// section with its own actions. It is outside the form, so a submit
	// never posts it and an error re-render never redraws it.
	After templ.Component
	// Action is the submit URL (plain; templ sanitizes it).
	Action string
	// Method is "post" when empty.
	Method string
	// HxPost submits over htmx, swapping the whole form in place on an
	// error re-render. Action and Method are still emitted, so the form
	// works without JS.
	HxPost bool
	Groups []Group
	// Values override Field.Value by Name, exactly as formpage.Props.Values
	// (a present key wins even when ""; an absent key leaves Field.Value
	// alone; keys naming no field are ignored).
	Values map[string]string
	// Errors set Field.Error by Name and drive the summary banner, which
	// renders only when len(Errors) > 0. A key naming no field is still
	// counted and listed (as a form-level error, without a link).
	Errors map[string]string
	// Submit is the submit button; the zero value is Label "Save changes"
	// with VariantPrimary. It always renders as <button type="submit"
	// form=ID>; Href, Hx* and Form on it are ignored.
	Submit action.Action
	// Cancel, when set, renders before Submit in the save bar, as a ghost
	// button unless given a Variant. It is usually a link back.
	Cancel *action.Action
	// Hidden renders as hidden inputs at the top of the form (form.Form).
	Hidden map[string]string
	// Validate is the default blur-validation URL (formfield.Field.Validate)
	// for every field that has none of its own; "" wires nothing.
	Validate string
	// Multipart emits enctype="multipart/form-data", required for a
	// TypeFile field. Its handler must call r.ParseMultipartForm before
	// reading values; r.ParseForm leaves a multipart body unread.
	Multipart bool
}

// id returns the <form> id with the default applied.
func (p Props) id() string {
	if p.ID == "" {
		return "settings"
	}
	return p.ID
}

// fields returns the resolved fields of one group.
func (p Props) fields(g Group) []formfield.Field {
	out := make([]formfield.Field, len(g.Fields))
	for i, f := range g.Fields {
		out[i] = formpage.ResolveField(f, p.Values, p.Errors, p.Validate)
	}
	return out
}

// allFields returns every declared field in group order, unresolved; the
// banner lists errors in this order.
func (p Props) allFields() []formfield.Field {
	var out []formfield.Field
	for _, g := range p.Groups {
		out = append(out, g.Fields...)
	}
	return out
}

// Field returns the declared field named name with Values, Errors and the
// default Validate applied, or false when no group declares it.
func (p Props) Field(name string) (formfield.Field, bool) {
	for _, g := range p.Groups {
		for _, f := range g.Fields {
			if f.Name == name {
				return formpage.ResolveField(f, p.Values, p.Errors, p.Validate), true
			}
		}
	}
	return formfield.Field{}, false
}

// formProps builds the form.Form Props: the id, classes and Attributes,
// Action/Method/Hidden/Multipart always, the htmx trio when HxPost is set.
// The form is a column with the same gap as the page root, so the banner
// and the sections are spaced like the page's own children.
func (p Props) formProps() form.Props {
	fp := form.Props{
		ID:         p.id(),
		Class:      utils.TwMerge("flex flex-col gap-8", p.Class),
		Attributes: p.Attributes,
		Action:     p.Action,
		Method:     p.Method,
		Hidden:     p.Hidden,
		Multipart:  p.Multipart,
	}
	if p.HxPost {
		fp.HxPost = p.Action
		fp.HxTarget = "#" + p.id()
		fp.HxSwap = "outerHTML"
	}
	return fp
}

// submit returns the Submit action with its defaults, bound to the form by
// id because the button renders outside the <form>.
func (p Props) submit() action.Action {
	a := formpage.Submit(p.Submit, "Save changes")
	a.Form = p.id()
	return a
}

// cancel returns the Cancel action with the ghost default applied, or
// false when there is none.
func (p Props) cancel() (action.Action, bool) {
	if p.Cancel == nil {
		return action.Action{}, false
	}
	a := *p.Cancel
	if a.Variant == "" {
		a.Variant = action.VariantGhost
	}
	return a, true
}

// saveBarClass is the sticky save bar, above page content but under
// dialogs and sheets, with the safe-area inset added to its bottom padding.
const saveBarClass = "wf-settings-bar sticky bottom-0 z-20 border-t border-border bg-background/95 backdrop-blur px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))]"

// Respond writes the settings page for r through the appshell App installed
// by (*App).Middleware, as formpage.Respond does. An htmx request targeting
// the form's id (an HxPost submit) gets the <form> alone, so an error
// re-render leaves After and the save bar untouched.
//
// Respond never redirects. On validation errors the POST handler sets
// p.Values and p.Errors and calls Respond, which answers 422 Unprocessable
// Entity; on success the handler redirects itself.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	if err := p.validateActions(); err != nil {
		return err
	}
	status := 0
	if len(p.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	return app.RenderPartsStatus(w, r, status, title, Page(p), map[string]templ.Component{p.id(): Body(p)})
}

// validateActions runs action.ValidateAll on the header, the resolved
// submit button and Cancel.
func (p Props) validateActions() error {
	groups := [][]action.Action{p.Header.Actions(), {p.submit()}}
	if c, ok := p.cancel(); ok {
		groups = append(groups, []action.Action{c})
	}
	return action.ValidateAll(groups...)
}
