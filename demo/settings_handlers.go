package demo

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/flash"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/settingspage"
	"github.com/bplus/webframe/components/toast"
	"github.com/bplus/webframe/demo/pages"
)

// settingsFormID is the settingspage ID (the <form> id and htmx swap
// target).
const settingsFormID = "settings"

// Account is the demo's one account, the record the settings page edits.
// Values are the form's field values by name; an unchecked toggle is "".
type Account struct {
	mu     sync.RWMutex
	values map[string]string
}

// NewAccount returns the demo account with its seed values: a display
// name and email, the weekly digest on, mention alerts off.
func NewAccount() *Account {
	return &Account{values: map[string]string{
		"display_name": "Morgan",
		"email":        "morgan@example.com",
		"digest":       "on",
		"mentions":     "",
	}}
}

// Settings is the demo server's account. Tests replace it with a fresh one.
var Settings = NewAccount()

// Values returns a copy of the stored field values.
func (a *Account) Values() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make(map[string]string, len(a.values))
	for k, v := range a.values {
		out[k] = v
	}
	return out
}

// Save replaces the stored values with vals.
func (a *Account) Save(vals map[string]string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.values = make(map[string]string, len(vals))
	for k, v := range vals {
		a.values[k] = v
	}
}

// settingsProps declares the settings page: Profile (display name, email),
// Notifications (two toggles) and the save bar with Cancel back to the
// dashboard. The Danger zone is not a Group: settingspage.Group holds
// fields only, with no slot for an action, so the demo renders it as its
// own section after the form (pages.Settings) with deleteAccountAction.
func settingsProps(values map[string]string) settingspage.Props {
	return settingspage.Props{
		ID:     settingsFormID,
		Header: pageheader.Props{Title: "Settings", Subtitle: "Your profile and how the demo reaches you."},
		Action: "/settings",
		HxPost: true,
		Groups: []settingspage.Group{
			{ID: "profile", Title: "Profile", Description: "How other builders see you.", Fields: []formfield.Field{
				{Name: "display_name", Label: "Display name", Required: true, Placeholder: "Morgan", Help: "Shown on everything you publish."},
				{Name: "email", Label: "Email", Type: formfield.TypeEmail, Placeholder: "you@example.com", Help: "Never shown to other builders."},
			}},
			{ID: "notifications", Title: "Notifications", Description: "What the demo would send if it sent anything.", Fields: []formfield.Field{
				{Name: "digest", Label: "Weekly digest", Type: formfield.TypeToggle, Help: "One email a week with what changed in your packs."},
				{Name: "mentions", Label: "Mentions", Type: formfield.TypeToggle, Help: "An email when another builder names one of your items."},
			}},
		},
		Values: values,
		Cancel: &action.Action{Label: "Cancel", Href: "/dashboard"},
	}
}

// deleteAccountAction is the Danger zone's one action: a destructive button
// that asks first and then posts to /settings/delete-account, which the demo
// refuses (HandleDeleteAccount), so the dialog and the refusal can both be
// seen.
func deleteAccountAction() action.Action {
	return action.Action{
		Label:   "Delete account",
		Variant: action.VariantDestructive,
		HxPost:  "/settings/delete-account",
		HxSwap:  "none",
		Confirm: &action.Confirm{
			Title:        "Delete your account?",
			Message:      "Every item and pack you own goes with it. This cannot be undone.",
			ConfirmLabel: "Delete account",
			CancelLabel:  "Keep it",
		},
	}
}

// settingsFields lists every declared field of the settings form, in
// order.
func settingsFields(p settingspage.Props) []formfield.Field {
	var out []formfield.Field
	for _, g := range p.Groups {
		out = append(out, g.Fields...)
	}
	return out
}

// submittedSettings reads every declared field from the parsed form, so an
// error re-render keeps exactly what was typed (an unchecked toggle comes
// back as "", which renders unchecked).
func submittedSettings(r *http.Request, p settingspage.Props) map[string]string {
	vals := make(map[string]string)
	for _, f := range settingsFields(p) {
		vals[f.Name] = r.FormValue(f.Name)
	}
	return vals
}

// validateSettings returns the errors by field name: the display name must
// not be blank. Other fields have no rules.
//
// @pure
func validateSettings(vals map[string]string) map[string]string {
	errs := make(map[string]string)
	if strings.TrimSpace(vals["display_name"]) == "" {
		errs["display_name"] = "Display name is required"
	}
	return errs
}

// respondSettings writes the settings page as settingspage.Respond would
// (RenderPartsStatus with the form as the named part and 422 when Errors is
// set), but with pages.Settings as the full page: settingspage.Page
// followed by the Danger zone section. settingspage has no slot for a
// section that is not a group of fields, so the demo composes the page
// itself; the form part is settingspage.Body, so an htmx submit still
// swaps the form alone.
func respondSettings(w http.ResponseWriter, r *http.Request, p settingspage.Props) {
	app, err := appshell.Must(r.Context())
	if err != nil {
		log.Printf("demo: render settings: %v", err)
		return
	}
	status := 0
	if len(p.Errors) > 0 {
		status = http.StatusUnprocessableEntity
	}
	full := pages.Settings(p, deleteAccountAction())
	parts := map[string]templ.Component{settingsFormID: settingspage.Body(p)}
	if err := app.RenderPartsStatus(w, r, status, "Settings", full, parts); err != nil {
		log.Printf("demo: render settings: %v", err)
	}
}

// HandleSettings renders the settings page with the stored values (GET
// /settings).
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleSettings(w http.ResponseWriter, r *http.Request) {
	respondSettings(w, r, settingsProps(Settings.Values()))
}

// HandleSettingsSave handles the form's submit (POST /settings). With a
// blank display name it re-renders the page with every submitted value and
// the error (422; htmx swaps the form in place); otherwise it saves and
// redirects to /settings (HX-Redirect for an htmx submit, else 303) with no
// success message: the page simply shows the saved values.
//
// @boundary HTTP
// @side-effects mutates Settings; writes HTML or a redirect
func HandleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	p := settingsProps(nil)
	vals := submittedSettings(r, p)
	if errs := validateSettings(vals); len(errs) > 0 {
		p.Values, p.Errors = vals, errs
		respondSettings(w, r, p)
		return
	}
	vals["display_name"] = strings.TrimSpace(vals["display_name"])
	Settings.Save(vals)
	redirectTo(w, r, "/settings")
}

// HandleDeleteAccount answers the Danger zone's confirmed action (POST
// /settings/delete-account) by refusing it: the demo has one account and
// keeps it. Over htmx the answer is a 409 whose only body is an error toast
// (nothing on the page changes); a plain request sets an error flash and
// returns to /settings (303), where the shell shows it once.
//
// @boundary HTTP
// @side-effects writes HTML or a redirect; may set the flash cookie
func HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	title := "Could not delete the account"
	body := "The demo has one account and keeps it; nothing was changed."
	if appshell.IsHX(r) {
		if err := toast.Respond(w, r, http.StatusConflict, toast.Error(title, body)); err != nil {
			log.Printf("demo: render refused account delete toast: %v", err)
		}
		return
	}
	flash.Error(w, r, title, body)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
