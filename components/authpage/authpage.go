// Package authpage renders the three pages an app shows outside its own
// chrome: sign in (a form), signed out (a message and a link back to sign
// in) and forbidden (a message and a way home). Each is one centered card
// with the app name as the heading; there is never a nav or a shell, so
// the page works before the user has an account and before the handler
// chain reaches App.Middleware.
//
// Phone width (390px): the centered column keeps 16px of padding on each
// side and the card is w-full inside it (max-w-sm only bites on wider
// screens); fields stack one per row; the submit button and every
// Provider button are full width at every width, and Providers stack one
// per row.
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
// Fields are needed; everything else has a usable zero value.
type Props struct {
	// ID and Class land on the outer centered column; Attributes spread
	// on it last.
	ID         string
	Class      string
	Attributes templ.Attributes

	// AppName is the card heading (the page's <h1>). Respond fills it
	// from App.Name when empty and an App is in the context; with
	// neither, the heading falls back to the mode's title.
	AppName string
	// Mode is the page to render; the zero value is ModeSignIn.
	Mode Mode
	// Action is the sign-in form's post URL (ModeSignIn) and the href of
	// the "Sign in again" link (ModeSignedOut, unless Submit.Href is set).
	// Emitted plain; templ sanitizes it.
	Action string
	// Method is the form method; "post" when empty.
	Method string
	// Fields are the sign-in form's fields, rendered one per row through
	// formfield.FormField in declared order. Field.Width is ignored: the
	// card is one column.
	Fields []formfield.Field
	// Values override Field.Value by Name, exactly as formpage does: a
	// present key wins even when "", an absent key leaves Field.Value
	// alone, keys naming no field are ignored.
	Values map[string]string
	// Errors set Field.Error by Name and, when Error is empty, drive the
	// summary banner ("Please fix the N errors below." with one linked
	// line per field). A key naming no field is listed without a link.
	Errors map[string]string
	// Submit is the page's one primary action. ModeSignIn: the submit
	// button (always <button type="submit">, full width; Href and Hx* are
	// ignored), default Label "Sign in". ModeSignedOut: a full-width link
	// to Submit.Href or Action, default Label "Sign in again".
	// ModeForbidden: a full-width link or button, default
	// {Label: "Go home", Href: "/"}. Variant defaults to primary.
	Submit action.Action
	// Providers render under the form, below a divider reading "or", as
	// full-width outline buttons ("Continue with …"), one per row. A
	// plain Href renders a link; Hx* fields render a button. ModeSignIn
	// only.
	Providers []action.Action
	// Message is the body text above the action for ModeSignedOut
	// (default "You have been signed out.") and ModeForbidden (default
	// "Your account does not have access."). ModeSignIn shows it under
	// the heading when set.
	Message string
	// Error is the title of an error banner (role="alert") rendered above
	// the form, inside the card: "Email or password is incorrect." Field
	// Errors, when any, are listed under it.
	Error string
	// Footer, when non-nil, renders centered under everything else, e.g.
	// a "Create account" link line.
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

// title returns the document title for the mode: "Sign in", "Signed out"
// or "Not allowed".
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

// resolve applies Values and Errors to one declared field without
// mutating the caller's slice, and drops Width (the card is one column).
func (p Props) resolve(f formfield.Field) formfield.Field {
	if v, ok := p.Values[f.Name]; ok {
		f.Value = v
	}
	if e, ok := p.Errors[f.Name]; ok {
		f.Error = e
	}
	f.Width = formfield.WidthFull
	// Browser and password-manager autofill key off autocomplete; a sign-in
	// form that forgets it is painful on a phone. Default the two fields a
	// sign-in has unless the app set its own (sign-up wants new-password).
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

// defaultAutocomplete is the autocomplete token a sign-in form's field
// gets by default: username for the email field, current-password for the
// password field, nothing for the rest.
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

// bannerTitle is the banner's first line: Error when set, otherwise
// "Please fix the N errors below." (singular for one), as formpage says.
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

// errorItems lists the field Errors for the banner: in declared field
// order (label from the field, falling back to its name), then errors
// whose key names no field, sorted by key so output is deterministic.
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

// submit returns Submit with the mode's defaults applied, always
// FullWidth: Label and, for ModeForbidden, Href "/" when the action has no
// destination at all. For ModeSignIn it is marked Submit with Href and the
// Hx* fields cleared, so action.Button renders the form's real
// <button type="submit"> and nothing else, as Props.Submit promises.
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

// provider returns one Provider as a full-width outline action: a link
// for a plain Href, a button carrying the hx-* attributes otherwise.
func provider(a action.Action) action.Action {
	a.Variant = action.VariantOutline
	a.FullWidth = true
	return a
}

// Respond writes the auth page for r without the shell. With an App in
// the context (the handler sits under App.Middleware) it renders through
// (*App).RenderBare, so the title suffix, theme init and App.Head match
// the rest of the app, and fills AppName from App.Name when empty;
// without one it renders through appshell.RenderBare, so a sign-in page
// served before App.Middleware still answers. Either way the response is
// the full document for a normal request and the bare card for a plain
// htmx request, and never includes the nav. The document title is
// "Sign in", "Signed out" or "Not allowed" (with the " — AppName" suffix
// when there is an app name).
//
// Respond answers 200, or 422 Unprocessable Entity for the re-render after
// a refused sign-in POST (Error or Errors set); appshell.Document
// configures htmx to swap error responses, so an htmx submit still shows
// it. A successful sign in redirects instead (303 See Other) and never
// renders this page.
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
