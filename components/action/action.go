// Package action defines the shared Action type that page-layer
// components (headers, toolbars, panels, menus) use to describe a
// button or link, plus renderers for it.
package action

import (
	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/utils"
)

// Variant is the visual emphasis of an Action. The zero value,
// VariantDefault, means "not chosen": each component that renders the
// action picks its own default for it (a page's Submit is primary, a
// table row action ghost, a plain action.Button secondary). Any other
// value is always rendered as named.
type Variant string

const (
	// VariantDefault is the zero value: the rendering component's default.
	// action.Button and action.Group render it secondary.
	VariantDefault Variant = ""
	// VariantSecondary is the muted style.
	VariantSecondary Variant = "secondary"
	// VariantPrimary is the filled primary style.
	VariantPrimary Variant = "primary"
	// VariantDestructive is for dangerous actions.
	VariantDestructive Variant = "destructive"
	// VariantGhost has no fill until hovered.
	VariantGhost Variant = "ghost"
	// VariantOutline is a bordered button on the page background.
	VariantOutline Variant = "outline"
)

// Confirm describes the confirmation shown before an Action runs. Message
// is emitted as hx-confirm: with confirm.js and the confirmdialog host on
// the page (appshell.Document renders both) it opens the styled dialog,
// otherwise the browser's native confirm box. Empty Title, ConfirmLabel
// and CancelLabel fall back to confirmdialog's defaults.
type Confirm struct {
	Title, Message, ConfirmLabel, CancelLabel string
}

// Action describes one button or link. A plain Href renders an
// anchor; any Hx* field renders a button with the matching hx-*
// attributes (Href is then ignored); Submit renders a form submit
// button.
type Action struct {
	ID    string
	Label string
	Href  string

	HxGet, HxPost, HxDelete, HxTarget, HxSwap string

	// Submit renders <button type="submit">. It wins over Href. Don't
	// combine it with an Hx* field: the button would both submit and send
	// the request; let the form carry hx-post instead (Validate reports
	// the combination). MenuAttrs ignores it.
	Submit bool
	// Form is the button's form attribute, so it can sit outside its
	// form. A plain link has no form attribute, so Form is dropped there
	// and Validate reports it.
	Form string
	// FullWidth makes the button or link span its container.
	FullWidth bool

	Variant  Variant
	Size     button.Size
	Icon     templ.Component
	Disabled bool
	Confirm  *Confirm

	Attributes templ.Attributes
}

// IsLink reports whether the action is plain navigation: Href set, no
// Hx* request attribute set and not Submit.
func (a Action) IsLink() bool {
	return a.Href != "" && !a.Submit && a.HxGet == "" && a.HxPost == "" && a.HxDelete == ""
}

func buttonVariant(v Variant) button.Variant {
	switch v {
	case VariantPrimary:
		return button.VariantDefault
	case VariantDestructive:
		return button.VariantDestructive
	case VariantGhost:
		return button.VariantGhost
	case VariantOutline:
		return button.VariantOutline
	default: // VariantSecondary, VariantDefault
		return button.VariantSecondary
	}
}

// MenuAttrs returns the hx-* and confirm attributes for a, with
// a.Attributes merged last so the caller's win. A dropdown item is itself
// a <button> and can't nest Button, so menus spread these onto the item
// instead. Submit, Form and FullWidth are not included.
func MenuAttrs(a Action) templ.Attributes {
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
// the confirm dialog (hx-confirm plus data-wf-confirm-*), or an empty map
// when Confirm is nil. A destructive Variant also gives the dialog a
// destructive confirm button.
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
		FullWidth:  a.FullWidth,
		Form:       a.Form,
		Disabled:   a.Disabled,
		Attributes: MenuAttrs(a),
	}
	if a.Submit {
		p.Type = button.TypeSubmit
	}
	if a.IsLink() {
		p.Href = a.Href
	}
	return p
}
