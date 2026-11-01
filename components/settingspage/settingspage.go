// Package settingspage renders a settings screen: a page header, then one
// <form> holding an error summary banner (errors only) and one h2 section
// per Group of fields, then optional After content (a section that is not
// a group of fields, such as a "Danger zone" with its own actions), and
// last a save bar that stays stuck to the bottom of the viewport while the
// page is longer than the screen. The save bar is always rendered (it
// never appears on the first edit) and holds Cancel (ghost) and Submit
// ("Save changes", primary), right-aligned. It is the page's last child,
// not the form's: the submit button carries form=<form id>, so it submits
// the form from outside it, and an htmx error re-render that swaps the
// form leaves the bar (and After) where they are. Like formpage, the
// caller declares the fields once and hands over the current Values (and,
// after a failed POST, the Errors) by field name; the field resolution,
// the banner and the submit button are formpage's own
// (formpage.ResolveField, formpage.ErrorBanner, formpage.Submit), so a
// settings page and a form page behave identically on a round trip.
//
// Phone width (390px): the header, banner, sections and After stack in
// one column and each Fieldset's grid is one column below md, so every
// field takes the full width. The save bar spans the width of the content
// column and its two buttons stay on one row (a third would wrap); its
// bottom padding adds the device's safe-area inset, so on a phone with a
// home indicator the buttons sit above it. The bar is the last child of
// the page root, so once the page is scrolled to the end it sits in flow
// below the last field (or below After) and covers nothing. The page root
// carries the wf-settings class; the stylesheet gives every focusable
// control under it a scroll-margin-bottom, so a control the browser
// scrolls into view on Tab lands clear of the bar. While the on-screen
// keyboard is open, mobile browsers pin bottom-stuck elements to the
// layout viewport, which may be behind the keyboard until the field loses
// focus; the bar is reachable again as soon as the keyboard closes.
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
	// Content, when non-nil, renders inside the section after the fieldset
	// and inside the <form>, so a widget that is not a formfield.Field (a
	// keyvalueeditor, a prose note, a codeblock) is posted by the save bar's
	// button with the fields and is re-rendered by an error swap of the
	// form; the handler reads it from r.Form like any field. A group with
	// Content and no Fields renders no fieldset, only the Content.
	Content templ.Component
}

// Props configures a settings page. Only Groups and Action are needed for
// a working page; everything else has a usable zero value.
type Props struct {
	// ID is the <form> id and the htmx swap target; "" means "settings".
	// The save bar's submit button names it in its form attribute.
	ID string
	// Class merges into the <form>'s classes; Attributes spread on it.
	Class      string
	Attributes templ.Attributes

	Header pageheader.Props
	// After, when set, renders after the form and before the save bar:
	// content that belongs on the settings page but is not a group of
	// fields, such as a layout.Section titled "Danger zone" holding a
	// destructive action with its own request. It is outside the form, so
	// a submit never posts it and an htmx error re-render (which swaps
	// the form alone) never redraws it; the save bar stays stuck to the
	// viewport bottom while it scrolls past.
	After templ.Component
	// Action is the submit URL (plain; templ sanitizes it).
	Action string
	// Method is "post" when empty.
	Method string
	// HxPost submits over htmx: the form gets hx-post=Action,
	// hx-target="#"+ID and hx-swap="outerHTML", so an error re-render
	// (Respond on a plain htmx request targeting the form) swaps the
	// whole form in place (the save bar lives outside it). Action and Method are
	// still emitted, so the form works without JS.
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
	// with VariantPrimary. Href and Hx* on it are ignored: it always
	// renders as <button type="submit" form=ID>, since the save bar sits
	// outside the form (Form on it is overwritten with the form's id).
	Submit action.Action
	// Cancel, when set, renders before Submit in the save bar as a ghost
	// button unless given another Variant (the zero Variant is the
	// secondary style, so ghost is the default here, not secondary). It
	// is usually a plain link back to where the user came from.
	Cancel *action.Action
	// Hidden renders as hidden inputs at the top of the form (form.Form).
	Hidden map[string]string
	// Validate is the default blur-validation URL (formfield.Field.Validate)
	// for every field that has none of its own; "" wires nothing.
	Validate string
	// Multipart emits enctype="multipart/form-data"; required for a
	// TypeFile field. The handler of a multipart form must call
	// r.ParseMultipartForm before reading any value: a prior r.ParseForm
	// parses only the URL query and leaves the body unread, so r.FormValue
	// then returns "" for every field in the body.
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

// submit returns the Submit action with the defaults applied ("Save
// changes", primary), marked Submit, and bound to the form by id: the
// button renders in the save bar outside the <form>, and the form
// attribute is what makes a button submit a form it is not inside (htmx
// honors it too, so an HxPost form still submits over htmx).
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

// saveBarClass is the sticky save bar: stuck to the bottom of the viewport
// above the page content (z-20, under dialogs and sheets), separated by a
// top border, on a translucent blurred page background so the fields
// scrolling under it stay faintly visible, with the safe-area inset added
// to its bottom padding.
const saveBarClass = "wf-settings-bar sticky bottom-0 z-20 border-t border-border bg-background/95 backdrop-blur px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))]"

// Respond writes the settings page for r through the appshell App installed
// by (*App).Middleware, as formpage.Respond does: the whole document for a
// normal request (plain GET/POST, boosted navigation, history restore),
// the bare page (header, form, After, save bar) for a plain htmx request
// with any other target, and the <form> alone (Body) for a plain htmx
// request whose HX-Target is the form's id, which is what an HxPost
// submit sends, so the error re-render swaps the form in place and leaves
// After and the save bar untouched.
//
// Respond never redirects. The POST handler decides: on validation errors
// it sets p.Values from the submitted form and p.Errors, then calls Respond,
// which answers 422 Unprocessable Entity (appshell.Document configures htmx
// to swap error responses, so the form re-renders in place with every
// submitted value kept and a truthful status); on success it redirects
// (303 See Other, or for an htmx submit an HX-Redirect header with an
// empty 200 body) and renders no success message.
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
