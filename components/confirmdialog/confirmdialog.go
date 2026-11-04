// Package confirmdialog is a styled replacement for the browser's native
// hx-confirm box. appshell.Document renders one Host per page, and
// static/js/confirm.js intercepts htmx:confirm, fills the Host from the
// trigger's data-wf-confirm-* attributes and hx-confirm text, and opens it.
// App code sets Confirm on an action.Action rather than rendering a
// ConfirmDialog itself.
//
// Needs JavaScript: the dialog never opens; the actions it guards need
// JavaScript for their request too, so nothing runs unconfirmed.
package confirmdialog

import "github.com/a-h/templ"

// HostID is the id of the page's confirm dialog. confirm.js looks it up by
// this id; if it is missing, the native confirm() box runs.
const HostID = "wf-confirm"

// Default texts, used when Props and the trigger leave a field empty.
const (
	DefaultTitle        = "Are you sure?"
	DefaultConfirmLabel = "Confirm"
	DefaultCancelLabel  = "Cancel"
)

// Props configures ConfirmDialog. Message has no default; confirm.js fills
// it from the hx-confirm text.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Title        string
	Message      string
	ConfirmLabel string
	CancelLabel  string
	// Destructive shows the destructive confirm button instead of the
	// primary one. Both are always rendered (one hidden) so confirm.js can
	// switch per trigger.
	Destructive bool
}

func (p Props) withDefaults() Props {
	if p.Title == "" {
		p.Title = DefaultTitle
	}
	if p.ConfirmLabel == "" {
		p.ConfirmLabel = DefaultConfirmLabel
	}
	if p.CancelLabel == "" {
		p.CancelLabel = DefaultCancelLabel
	}
	return p
}

// hostAttrs puts the default texts on the dialog root under the same
// data-wf-confirm-* names a trigger uses, so confirm.js can fall back to
// them. The caller's Attributes win on conflict.
func hostAttrs(p Props) templ.Attributes {
	attrs := templ.Attributes{
		"data-wf-confirm-title":  p.Title,
		"data-wf-confirm-label":  p.ConfirmLabel,
		"data-wf-confirm-cancel": p.CancelLabel,
	}
	if p.Destructive {
		attrs["data-wf-confirm-destructive"] = "true"
	}
	for k, v := range p.Attributes {
		attrs[k] = v
	}
	return attrs
}
