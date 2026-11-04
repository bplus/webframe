package demo

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/datetimepicker"
	"github.com/bplus/webframe/components/flash"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/keyvalueeditor"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/settingspage"
	"github.com/bplus/webframe/components/toast"
	"github.com/bplus/webframe/demo/pages"
)

// maxEnvRows caps the Environment editor (MaxRows on the widget, and the
// handler truncates a longer submit).
const maxEnvRows = 8

// settingsFormID is the settingspage ID (the <form> id and htmx swap
// target).
const settingsFormID = "settings"

// Account is the demo's one account, the record the settings page edits.
// Values are the form's field values by name; an unchecked toggle is "".
// Env is the Environment editor's rows, saved with the values by the one
// Save button.
type Account struct {
	mu     sync.RWMutex
	values map[string]string
	env    []keyvalueeditor.Row
}

// NewAccount returns the demo account with its seed values: a display
// name and email, the weekly digest on, mention alerts off, quiet hours
// from ten in the evening, and two environment rows.
func NewAccount() *Account {
	return &Account{
		values: map[string]string{
			"display_name": "Morgan",
			"email":        "morgan@example.com",
			"digest":       "on",
			"mentions":     "",
			"quiet_start":  "22:00",
		},
		env: []keyvalueeditor.Row{{Key: "REGION", Value: "eu-west"}, {Key: "LOG_LEVEL", Value: "info"}},
	}
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

// Save replaces the stored values with vals and the environment rows with
// env, in one step: both come from the one settings form.
func (a *Account) Save(vals map[string]string, env []keyvalueeditor.Row) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.values = make(map[string]string, len(vals))
	for k, v := range vals {
		a.values[k] = v
	}
	a.env = append([]keyvalueeditor.Row(nil), env...)
}

// Env returns a copy of the environment rows.
func (a *Account) Env() []keyvalueeditor.Row {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]keyvalueeditor.Row(nil), a.env...)
}

// settingsProps declares the settings page: Profile (display name, email),
// Notifications (two toggles and the quiet-hours start time) and
// Environment, a Group whose Content is the key/value editor (not a
// formfield type, so it is Content rather than a Field): it sits inside the
// settings form, so the save bar's button posts its rows as env.key and
// env.value pairs with the fields, and an error re-render keeps what was
// typed. After the form comes the Danger zone, outside it. The save bar
// closes the page with Cancel back to the dashboard.
func settingsProps(values map[string]string, env []keyvalueeditor.Row) settingspage.Props {
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
				{Name: "quiet_start", Label: "Quiet hours start", Type: formfield.TypeTime, Help: "No email after this time until eight in the morning.", Width: formfield.WidthHalf},
			}},
			{ID: "environment", Title: "Environment", Description: "Variables the demo would pass to a build. Rows post as env.key and env.value pairs with the rest of the form; the handler zips them back together, and a row left with a blank name is dropped on save.",
				Content: keyvalueeditor.KeyValueEditor(environmentProps(env))},
		},
		Values: values,
		After:  pages.DangerZone(deleteAccountAction()),
		Cancel: &action.Action{Label: "Cancel", Href: "/dashboard"},
	}
}

// environmentProps declares the Environment editor: rows named env (so the
// handler reads keyvalueeditor.Parse(r.Form, "env")), capped at maxEnvRows.
func environmentProps(rows []keyvalueeditor.Row) keyvalueeditor.Props {
	return keyvalueeditor.Props{
		ID:               "env",
		Name:             "env",
		Rows:             rows,
		KeyPlaceholder:   "NAME",
		ValuePlaceholder: "value",
		MaxRows:          maxEnvRows,
		AddLabel:         "Add variable",
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
// not be blank; the quiet-hours start, when given, must be a time the
// native control writes ("22:00"). Other fields have no rules.
func validateSettings(vals map[string]string) map[string]string {
	errs := make(map[string]string)
	if strings.TrimSpace(vals["display_name"]) == "" {
		errs["display_name"] = "Display name is required"
	}
	if _, err := datetimepicker.Parse(datetimepicker.KindTime, vals["quiet_start"], nil); err != nil {
		errs["quiet_start"] = "Enter the time as HH:MM"
	}
	return errs
}

// HandleSettings renders the settings page with the stored values (GET
// /settings).
func HandleSettings(w http.ResponseWriter, r *http.Request) {
	if err := settingspage.Respond(w, r, "Settings", settingsProps(Settings.Values(), Settings.Env())); err != nil {
		log.Printf("demo: render settings: %v", err)
	}
}

// HandleSettingsSave handles the form's submit (POST /settings), fields and
// Environment rows together: keyvalueeditor.Parse zips the env.key/env.value
// fields into rows and drops the blank-keyed ones, and the first maxEnvRows
// are kept. With a blank display name or a bad time it re-renders the page
// with every submitted value, the submitted rows and the errors (422; htmx
// swaps the form in place; nothing is stored); otherwise it saves both and
// redirects to /settings (HX-Redirect for an htmx submit, else 303) with no
// success message: the page simply shows the saved values and rows.
func HandleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	rows := keyvalueeditor.Parse(r.Form, "env")
	if len(rows) > maxEnvRows {
		rows = rows[:maxEnvRows]
	}
	p := settingsProps(nil, rows)
	vals := submittedSettings(r, p)
	if errs := validateSettings(vals); len(errs) > 0 {
		p.Values, p.Errors = vals, errs
		if err := settingspage.Respond(w, r, "Settings", p); err != nil {
			log.Printf("demo: render settings errors: %v", err)
		}
		return
	}
	vals["display_name"] = strings.TrimSpace(vals["display_name"])
	Settings.Save(vals, rows)
	appshell.Redirect(w, r, "/settings")
}

// HandleDeleteAccount answers the Danger zone's confirmed action (POST
// /settings/delete-account) by refusing it: the demo has one account and
// keeps it. Over htmx the answer is a 409 whose only body is an error toast
// (nothing on the page changes); a plain request sets an error flash and
// returns to /settings (303), where the shell shows it once.
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
