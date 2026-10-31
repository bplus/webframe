// Package action defines the shared Action type that page-layer
// components (headers, toolbars, panels, menus) use to describe a
// button or link, plus renderers for it.
package action

import (
	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/utils"
)

// Variant is the visual emphasis of an Action. The zero value is the
// secondary style.
type Variant string

const (
	// VariantSecondary is the muted style (zero value).
	VariantSecondary Variant = ""
	// VariantPrimary is the filled primary style.
	VariantPrimary Variant = "primary"
	// VariantDestructive is for dangerous actions.
	VariantDestructive Variant = "destructive"
	// VariantGhost has no fill until hovered.
	VariantGhost Variant = "ghost"
)

// Confirm describes the confirmation shown before an Action runs. Message
// is emitted as hx-confirm: with confirm.js and the confirmdialog host on
// the page (appshell.Document renders both) that opens the styled dialog,
// without them the browser's native box runs. Title, ConfirmLabel and
// CancelLabel ride along as data-wf-confirm-* attributes for the dialog;
// empty fields fall back to confirmdialog's defaults ("Are you sure?",
// "Confirm", "Cancel").
type Confirm struct {
	Title, Message, ConfirmLabel, CancelLabel string
}

// Action describes one button or link. A plain Href renders an
// anchor; any Hx* field renders a button with the matching hx-*
// attributes (Href is then ignored).
type Action struct {
	ID    string
	Label string
	Href  string

	HxGet, HxPost, HxDelete, HxTarget, HxSwap string

	Variant  Variant
	Size     button.Size
	Icon     templ.Component
	Disabled bool
	Confirm  *Confirm

	Attributes templ.Attributes
}

// IsLink reports whether the action is plain navigation: Href set and
// no Hx* request attribute set.
func (a Action) IsLink() bool {
	return a.Href != "" && a.HxGet == "" && a.HxPost == "" && a.HxDelete == ""
}

func buttonVariant(v Variant) button.Variant {
	switch v {
	case VariantPrimary:
		return button.VariantDefault
	case VariantDestructive:
		return button.VariantDestructive
	case VariantGhost:
		return button.VariantGhost
	default:
		return button.VariantSecondary
	}
}

// attrs builds the hx-* attributes merged with caller Attributes.
func (a Action) attrs() templ.Attributes {
	hx := templ.Attributes{}
	if a.HxGet != "" {
		hx["hx-get"] = a.HxGet
	}
	if a.HxPost != "" {
		hx["hx-post"] = a.HxPost
	}
	if a.HxDelete != "" {
		hx["hx-delete"] = a.HxDelete
	}
	if a.HxTarget != "" {
		hx["hx-target"] = a.HxTarget
	}
	if a.HxSwap != "" {
		hx["hx-swap"] = a.HxSwap
	}
	return utils.MergeAttributes(hx, a.ConfirmAttributes(), a.Attributes)
}

// ConfirmAttributes returns the attributes that route the action through
// the confirm dialog, or an empty map when Confirm is nil: hx-confirm with
// the Message (kept as both the trigger and the native fallback), then
// data-wf-confirm-title, data-wf-confirm-label and data-wf-confirm-cancel
// for the non-empty Title, ConfirmLabel and CancelLabel, and
// data-wf-confirm-destructive="true" when the Variant is destructive so the
// dialog shows its destructive confirm button. Button uses it; renderers that
// build a menu item's attributes themselves should too, so every place an
// action renders agrees.
func (a Action) ConfirmAttributes() templ.Attributes {
	attrs := templ.Attributes{}
	if a.Confirm == nil {
		return attrs
	}
	attrs["hx-confirm"] = a.Confirm.Message
	if a.Confirm.Title != "" {
		attrs["data-wf-confirm-title"] = a.Confirm.Title
	}
	if a.Confirm.ConfirmLabel != "" {
		attrs["data-wf-confirm-label"] = a.Confirm.ConfirmLabel
	}
	if a.Confirm.CancelLabel != "" {
		attrs["data-wf-confirm-cancel"] = a.Confirm.CancelLabel
	}
	if a.Variant == VariantDestructive {
		attrs["data-wf-confirm-destructive"] = "true"
	}
	return attrs
}

func (a Action) buttonProps() button.Props {
	p := button.Props{
		ID:         a.ID,
		Variant:    buttonVariant(a.Variant),
		Size:       a.Size,
		Disabled:   a.Disabled,
		Attributes: a.attrs(),
	}
	if a.IsLink() {
		p.Href = a.Href
	}
	return p
}
