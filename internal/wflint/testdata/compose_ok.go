package x

import (
	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/card"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/compose"
)

// An app written in plain Go fills containers with compose: no markup, no
// templ.ComponentFunc, nothing for wflint to report.
func status(name string, fresh bool) templ.Component {
	var tag templ.Component
	if fresh {
		tag = compose.With(badge.Badge(badge.Props{Variant: badge.VariantSecondary}), compose.Text("New"))
	}
	return compose.With(card.Card(),
		compose.With(layout.Inline(), compose.Text(name), tag),
		compose.Join(compose.Text("Ready "), compose.Text("to ship.")),
	)
}
