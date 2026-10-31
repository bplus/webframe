// Package confirmdialog is the styled confirmation that replaces the
// browser's native hx-confirm box. One instance per page is enough:
// appshell.Document renders Host() (ID "wf-confirm") after the page content,
// and static/js/confirm.js intercepts htmx:confirm, fills that instance from
// the trigger's data-wf-confirm-* attributes (action.Confirm emits them) and
// the hx-confirm text, and opens it through dialog.min.js. App code never
// renders a ConfirmDialog itself; it sets Confirm on an action.Action.
package confirmdialog

import "github.com/a-h/templ"

// HostID is the element id of the page's one confirm dialog instance (the
// dialog root and its dialog.min.js instance id). confirm.js looks it up by
// this id; without it the native confirm() box runs.
const HostID = "wf-confirm"

// Default texts used when Props (or a trigger's data attributes) leave a
// field empty.
const (
	DefaultTitle        = "Are you sure?"
	DefaultConfirmLabel = "Confirm"
	DefaultCancelLabel  = "Cancel"
)

// Props configures ConfirmDialog. All text fields fall back to the
// Default* constants; Message has no default (an empty message renders an
// empty description that confirm.js fills from the hx-confirm text).
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Title        string
	Message      string
	ConfirmLabel string
	CancelLabel  string
	// Destructive shows the destructive confirm button instead of the
	// primary one. The dialog always carries both buttons (one hidden) so
	// confirm.js can switch per trigger without touching classes.
	Destructive bool
}

// withDefaults returns p with empty text fields set to the defaults.
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

// hostAttrs are the attributes on the dialog root: the resolved default
// texts under the same data-wf-confirm-* names a trigger uses, so confirm.js
// reads the trigger first and falls back to the root, then the caller's
// Attributes (which win on conflict).
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
