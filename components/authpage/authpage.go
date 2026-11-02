// Package authpage renders the pages an app shows outside its own chrome:
// sign in, signed out, and forbidden. Each is one centered card with the
// app name as the heading and no nav, so it works before the user has an
// account and outside App.Middleware.
package authpage

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/form"
	"github.com/bplus/webframe/components/formfield"
)

// Mode selects which of the three pages renders. The zero value is
// sign in.
type Mode string

const (
	// ModeSignIn renders the sign-in form (zero value).
	ModeSignIn Mode = ""
	// ModeSignedOut confirms the user is signed out and links back to
	// sign in.
	ModeSignedOut Mode = "signed-out"
	// ModeForbidden tells a signed-in user their account has no access
	// and offers a way home.
	ModeForbidden Mode = "forbidden"
)

// Props configures an auth page. Only Mode and, for sign in, Action and
// Fields are required.
type Props struct {
	// ID, Class and Attributes apply to the outer centered column.
	ID         string
	Class      string
	Attributes templ.Attributes

	// AppName is the card heading. Respond fills it from App.Name when
	// empty; with neither, the heading is the mode's title.
	AppName string
	// Mode is the page to render; the zero value is ModeSignIn.
	Mode Mode
	// Action is the sign-in form's post URL, and the default href of the
	// "Sign in again" link in ModeSignedOut.
	Action string
	// Method is the form method; "post" when empty.
	Method string
	// Fields are the sign-in form's fields, one per row. Field.Width is
	// ignored. Email and password fields default to autocomplete
	// "username" and "current-password" unless the field sets its own.
	Fields []formfield.Field
	// Values override Field.Value by Name; a present key wins even when "".
	Values map[string]string
	// Errors set Field.Error by Name and are listed in the error banner,
	// linked to their fields.
	Errors map[string]string
	// Submit is the page's primary action, always full width. In
	// ModeSignIn it is the form's submit button (Href and Hx* are
	// ignored). Defaults per mode: "Sign in"; "Sign in again" linking to
	// Action; "Go home" linking to "/".
	Submit action.Action
	// Providers are alternative sign-in buttons (such as "Continue with
	// …") shown under the form after an "or" divider, outline unless
	// they set a Variant. ModeSignIn only.
	Providers []action.Action
	// Message is the body text. ModeSignedOut and ModeForbidden have
	// defaults; ModeSignIn shows it under the heading when set.
	Message string
	// Error is the title of an error banner above the form, such as
	// "Email or password is incorrect."
	Error string
	// Footer, when non-nil, renders centered at the bottom, such as a
	// "Create account" link.
	Footer templ.Component
	// Logo, when non-nil, renders centered above the app name.
	Logo templ.Component
}

// mode returns Mode with the zero value resolved.
func (p Props) mode() Mode {
	switch p.Mode {
	case ModeSignedOut, ModeForbidden:
		return p.Mode
	default:
		return ModeSignIn
	}
}

// title returns the document title for the mode.
func (p Props) title() string {
	switch p.mode() {
	case ModeSignedOut:
		return "Signed out"
	case ModeForbidden:
		return "Not allowed"
	default:
		return "Sign in"
	}
}

// heading is the card's <h1>: AppName, or the mode's title without one.
func (p Props) heading() string {
	if p.AppName != "" {
		return p.AppName
	}
	return p.title()
}

// message returns Message with the mode's default applied.
func (p Props) message() string {
	if p.Message != "" {
		return p.Message
	}
	switch p.mode() {
	case ModeSignedOut:
		return "You have been signed out."
	case ModeForbidden:
		return "Your account does not have access."
	default:
		return ""
	}
}

// formID is the <form> id: ID+"-form", or "signin-form".
func (p Props) formID() string {
	if p.ID != "" {
		return p.ID + "-form"
	}
	return "signin-form"
}

// resolve applies Values, Errors and the autocomplete default to one
// field, without mutating the caller's slice.
func (p Props) resolve(f formfield.Field) formfield.Field {
	if v, ok := p.Values[f.Name]; ok {
		f.Value = v
	}
	if e, ok := p.Errors[f.Name]; ok {
		f.Error = e
	}
	f.Width = formfield.WidthFull
	// Password managers key off autocomplete. Default it unless the app set
	// its own (a sign-up form wants new-password).
	if ac := defaultAutocomplete(f.Type); ac != "" {
		if _, set := f.Attributes["autocomplete"]; !set {
			attrs := make(templ.Attributes, len(f.Attributes)+1)
			for k, v := range f.Attributes {
				attrs[k] = v
			}
			attrs["autocomplete"] = ac
			f.Attributes = attrs
		}
	}
	return f
}

// defaultAutocomplete is the default autocomplete token for a sign-in
// field of type t, or "".
func defaultAutocomplete(t formfield.Type) string {
	switch t {
	case formfield.TypeEmail:
		return "username"
	case formfield.TypePassword:
		return "current-password"
	}
	return ""
}

// fields returns the resolved sign-in fields in declared order.
func (p Props) fields() []formfield.Field {
	out := make([]formfield.Field, len(p.Fields))
	for i, f := range p.Fields {
		out[i] = p.resolve(f)
	}
	return out
}

// Field returns the declared field named name with Values and Errors
// applied, or false when no field has that name.
func (p Props) Field(name string) (formfield.Field, bool) {
	for _, f := range p.Fields {
		if f.Name == name {
			return p.resolve(f), true
		}
	}
	return formfield.Field{}, false
}

// formProps builds the form.Form Props for the sign-in form.
func (p Props) formProps() form.Props {
	return form.Props{
		ID:     p.formID(),
		Class:  "flex flex-col gap-4",
		Action: p.Action,
		Method: p.Method,
	}
}

// hasBanner reports whether the error banner renders: an Error, or any
// field Errors.
func (p Props) hasBanner() bool {
	return p.Error != "" || len(p.Errors) > 0
}

// bannerTitle is Error when set, otherwise "Please fix the N errors
// below."
func (p Props) bannerTitle() string {
	if p.Error != "" {
		return p.Error
	}
	n := len(p.Errors)
	if n == 1 {
		return "Please fix the 1 error below."
	}
	return "Please fix the " + strconv.Itoa(n) + " errors below."
}

// errorItem is one line of the summary banner.
type errorItem struct {
	Label, Message string
	Href           string // "#"+ControlID(), or "" for an error naming no field
}

// errorItems lists Errors in declared field order, then errors naming no
// field (unlinked), sorted by key.
func (p Props) errorItems() []errorItem {
	if len(p.Errors) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(p.Errors))
	var items []errorItem
	for _, f := range p.Fields {
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

// submit returns Submit with the mode's defaults applied. For ModeSignIn
// Href and Hx* are cleared so it renders as a plain submit button.
func (p Props) submit() action.Action {
	a := p.Submit
	if a.Variant == "" {
		a.Variant = action.VariantPrimary
	}
	a.FullWidth = true
	switch p.mode() {
	case ModeSignedOut:
		if a.Label == "" {
			a.Label = "Sign in again"
		}
		if a.Href == "" {
			a.Href = p.Action
		}
		if a.Href == "" {
			a.Href = "/signin" // never a dead button
		}
	case ModeForbidden:
		if a.Label == "" {
			a.Label = "Go home"
		}
		if a.Href == "" && a.HxGet == "" && a.HxPost == "" && a.HxDelete == "" {
			a.Href = "/"
		}
	default:
		if a.Label == "" {
			a.Label = "Sign in"
		}
		a.Submit = true
		a.Href, a.HxGet, a.HxPost, a.HxDelete, a.HxTarget, a.HxSwap = "", "", "", "", "", ""
	}
	return a
}

// provider returns a Provider as a full-width action, outline unless it
// set a Variant.
func provider(a action.Action) action.Action {
	if a.Variant == "" {
		a.Variant = action.VariantOutline
	}
	a.FullWidth = true
	return a
}

// Respond writes the auth page for r without the shell, through
// (*App).RenderBare when an App is in the context (filling AppName from
// it) and appshell.RenderBare otherwise. The status is 200, or 422 when
// Error or Errors is set.
func Respond(w http.ResponseWriter, r *http.Request, p Props) error {
	if p.Error != "" || len(p.Errors) > 0 {
		appshell.SetHeaders(w)
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	if app, ok := appshell.FromContext(r.Context()); ok {
		if p.AppName == "" {
			p.AppName = app.Name
		}
		return app.RenderBare(w, r, p.title(), AuthPage(p))
	}
	// With no app name the document title is the mode title alone.
	return appshell.RenderBare(w, r, p.AppName, p.title(), AuthPage(p))
}
