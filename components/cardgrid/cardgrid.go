// Package cardgrid renders a catalog of cards: the public-catalog
// multi-item display (a library grid, an Explore page, a list of mods or
// creators) declared as data. Each card is card.Card with optional art
// on top, a title (a link when Href is set), a muted subtitle, badges,
// a meta line and footer actions.
package cardgrid

import (
	"strings"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/aspectratio"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/components/emptystate"
)

// Art is the picture on top of a card, rendered through
// aspectratio.AspectRatio and image.Image with object-cover so the file
// is cropped to the frame rather than stretching it. The zero Ratio
// means aspectratio.RatioVideo (16:9): aspectratio.RatioAuto would
// collapse the frame to zero height, so it is not a useful default here.
// Alt is always emitted; an empty Alt marks the picture as decorative.
type Art struct {
	Src   string
	Alt   string
	Ratio aspectratio.Ratio
}

// Badge is one status pill on a card: the Label is the badge text and
// Props its badge.Props (variant, class, attributes). badge.Props carries
// no text of its own, which is why the plan's `[]badge.Props` became
// this pair. StatusBadge builds one from a status word.
type Badge struct {
	Label string
	Props badge.Props
}

// StatusBadge builds a Badge whose Label is badge.StatusLabel(kind) and
// whose variant is badge.Status(kind), so apps never pick colors:
// StatusBadge("", "published") shows "Published" in the success tone.
// A non-empty label overrides the derived display text.
//
// @pure
func StatusBadge(label, kind string) Badge {
	if label == "" {
		label = badge.StatusLabel(kind)
	}
	return Badge{Label: label, Props: badge.Status(kind)}
}

// Card is one item in the grid.
//
//   - ID, when set, is the id of the card's root element.
//   - Title is the heading (an h3 through card.Title). When Href is set
//     the title is a link whose hit area covers the whole card, so the
//     card has exactly one focusable element for the navigation.
//   - Subtitle renders muted under the title.
//   - Art, when non-nil, is the picture on top.
//   - Badges render in a wrapping row under the subtitle.
//   - Meta is a list of short facts joined with " · " on one small line.
//   - Actions render as small ghost buttons in the footer (an Action with
//     its own Variant keeps it); Footer is a free component rendered in
//     the same footer row, after the actions.
type Card struct {
	ID       string
	Title    string
	Subtitle string
	Href     string
	Art      *Art
	Badges   []Badge
	Meta     []string
	Actions  []action.Action
	Footer   templ.Component
}

// Props configures a CardGrid.
//
//   - Items are the cards, in order.
//   - Columns caps the column count on wide screens: 4 (the default for 0
//     and any other value), 3 or 2. The grid always starts at one column
//     and adds columns by breakpoint up to the cap (see CardGrid).
//   - Dense tightens card padding (p-4 instead of p-6) and the grid gap.
//   - Empty is shown full width when Items is empty; a zero value gets
//     the title "Nothing here yet".
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes
	Items      []Card
	Columns    int
	Dense      bool
	Empty      emptystate.Props
}

// columns resolves the cap: 2 and 3 are honored, anything else is 4.
func (p Props) columns() int {
	switch p.Columns {
	case 2, 3:
		return p.Columns
	default:
		return 4
	}
}

// gridClasses returns the responsive column classes for the cap and the
// gap for the density.
func (p Props) gridClasses() string {
	cols := "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
	switch p.columns() {
	case 2:
		cols = "grid-cols-1 sm:grid-cols-2"
	case 3:
		cols = "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"
	}
	gap := "gap-4"
	if p.Dense {
		gap = "gap-3"
	}
	return "grid " + cols + " " + gap
}

// padding is the card section padding for the density.
func (p Props) padding() string {
	if p.Dense {
		return "p-4"
	}
	return "p-6"
}

// empty resolves the zero-value Empty props to the default title.
func (p Props) empty() emptystate.Props {
	e := p.Empty
	if e.Title == "" && e.Description == "" && e.Icon == "" && e.Action == nil {
		e.Title = "Nothing here yet"
	}
	return e
}

// ratio resolves the art's frame: the zero Ratio is 16:9.
func (a Art) ratio() aspectratio.Ratio {
	if a.Ratio == "" || a.Ratio == aspectratio.RatioAuto {
		return aspectratio.RatioVideo
	}
	return a.Ratio
}

// metaLine joins the meta facts with a middle dot.
func (c Card) metaLine() string {
	return strings.Join(c.Meta, " · ")
}

// hasFooter reports whether the footer row renders at all.
func (c Card) hasFooter() bool {
	return len(c.Actions) > 0 || c.Footer != nil
}

// footerAction returns the action as rendered in the footer: ghost and
// small unless the caller chose a Variant or Size.
func footerAction(a action.Action) action.Action {
	if a.Variant == "" {
		a.Variant = action.VariantGhost
	}
	if a.Size == "" {
		a.Size = button.SizeSm
	}
	return a
}
