// Package toolbar renders a row of actions (left group, right group)
// around optional middle content such as a search field, with a phone
// collapse that moves surplus actions into a "⋯" menu.
package toolbar

import (
	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
)

// Props configures Toolbar. Left and Right are the two action groups;
// children render between them and grow to fill the remaining width.
//
// Overflow is the per-side inline budget on a phone: actions beyond the
// first Overflow go into a "⋯" dropdown labelled "More actions". The
// zero value means 1; -1 disables the collapse entirely, so one row is
// rendered at every width and wraps as needed.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Left, Right []action.Action
	Overflow    int
}

// overflow resolves the Overflow default: 0 → 1. A negative value is
// returned as is and means "never collapse".
func (p Props) overflow() int {
	if p.Overflow == 0 {
		return 1
	}
	return p.Overflow
}

// collapses reports whether a phone rendering is needed at all: the
// collapse is enabled and at least one side has more actions than fit.
func (p Props) collapses() bool {
	n := p.overflow()
	return n > 0 && (len(p.Left) > n || len(p.Right) > n)
}

// split divides one side's actions into the inline prefix and the menu
// remainder for the phone rendering.
func split(actions []action.Action, n int) (inline, menu []action.Action) {
	if n < 0 || len(actions) <= n {
		return actions, nil
	}
	return actions[:n], actions[n:]
}

// menuItemAttrs builds the hx-* attributes for a non-link action as
// dropdown item attributes, merged with the action's own Attributes.
// The dropdown item is itself the <button>, so action.Button is not
// nested inside it (a button inside a button is invalid HTML). Mirrors
// appshell's user-menu helper; action.Action keeps its attribute
// builder unexported.
func menuItemAttrs(a action.Action) templ.Attributes {
	attrs := templ.Attributes{}
	set := func(k, v string) {
		if v != "" {
			attrs[k] = v
		}
	}
	set("hx-get", a.HxGet)
	set("hx-post", a.HxPost)
	set("hx-delete", a.HxDelete)
	set("hx-target", a.HxTarget)
	set("hx-swap", a.HxSwap)
	if a.Confirm != nil {
		set("hx-confirm", a.Confirm.Message)
	}
	for k, v := range a.Attributes {
		attrs[k] = v
	}
	return attrs
}
