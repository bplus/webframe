// Package datatable renders a declarative data table on top of the table
// primitive: columns and rows as data, sort links built from a
// listquery.Spec, an optional selection column, per-row actions, and an
// empty state that keeps the header in place.
//
// Phone collapse (below md, 768px): columns marked Hidden are removed
// (`hidden md:table-cell` on their th and td), and each row's actions turn
// from inline ghost buttons (`hidden md:flex`) into one "⋯" dropdown
// labelled "Row actions" (`md:hidden`). Both renderings are in the DOM at
// every width; CSS picks one.
//
// Sticky header: StickyHeader pins the thead (sticky top-0 z-10 bg-card,
// opaque so rows scroll under it). With MaxHeight the table's wrapper is a
// capped scroll box and the header sticks to its top at every width.
// Without MaxHeight the wrapper stops clipping on md and up
// (`md:overflow-visible`) so the header sticks to the page; below md the
// wrapper keeps its own horizontal scroll (Hidden columns are what make a
// table fit a phone, and a table that still overflows must stay reachable),
// so there the header sticks only within that scroll box.
package datatable

import (
	"strconv"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/listquery"
)

// Align is a column's horizontal text alignment.
type Align string

const (
	// AlignLeft is the default.
	AlignLeft Align = ""
	// AlignRight suits numbers and dates.
	AlignRight Align = "right"
	// AlignCenter suits short status glyphs.
	AlignCenter Align = "center"
)

// Height is a cap on the table wrapper's height (Props.MaxHeight). The
// wrapper is already overflow-auto, so a capped table scrolls inside it and
// a StickyHeader sticks to the top of that scroll box.
type Height string

const (
	// HeightNone leaves the wrapper as tall as its rows (the default).
	HeightNone Height = ""
	// HeightSm caps the wrapper at 240px.
	HeightSm Height = "sm"
	// HeightMd caps the wrapper at 400px.
	HeightMd Height = "md"
	// HeightLg caps the wrapper at 600px.
	HeightLg Height = "lg"
)

// Column describes one table column. Key is the listquery sort key when
// Sortable is set; Label is the header text.
type Column struct {
	Key, Label string
	Sortable   bool
	Align      Align
	// Mono renders the column's cells in tabular mono (numbers, ids, dates).
	Mono bool
	// Hidden drops the column below md (phone): the th and every td carry
	// `hidden md:table-cell`.
	Hidden bool
	// Shrink makes the column as narrow as its content (w-px
	// whitespace-nowrap): status chips, dates.
	Shrink bool
}

// Cell is one table cell. Precedence: Component, then Badge (wrapping
// Text), then Text. Href on a cell makes its content a link; the first
// cell of a Row with Href is linked to the row's Href automatically.
type Cell struct {
	Text, Href string
	Mono       bool
	Muted      bool
	// Badge renders Text inside a badge.Badge with these props.
	Badge *badge.Props
	// Component replaces the cell content entirely.
	Component templ.Component
}

// BadgeCell is a Cell whose Text is shown inside a status badge: kind goes
// through badge.StatusPtr to pick the variant (published → success, failed →
// destructive, and so on; unknown words → outline) and text is what the
// badge says. Pass badge.StatusLabel(kind) as text for the display form of
// the status word itself.
//
// @pure
func BadgeCell(text, kind string) Cell {
	return Cell{Text: text, Badge: badge.StatusPtr(kind)}
}

// Row is one table row. ID is the selection value and the <tr> id suffix
// (`<tableID>-row-<ID>`, or `row-<ID>` when the table has no ID). Href
// links the first cell to it and gives the row a pointer cursor. Actions
// render inline on md+ and as a "⋯" menu below md; an Action.ID therefore
// appears twice in the DOM, so target row actions by class or by row id.
type Row struct {
	ID      string
	Href    string
	Cells   []Cell
	Actions []action.Action
}

// Props configures DataTable. Columns and Rows are the data; Spec is the
// current sort state and the source of every header link (via
// Spec.URL(Base, listquery.WithSort(key))). Base "" falls back to the
// request path stored by appshell.Middleware, and to "" (a query-only href
// relative to the current URL) when there is none either.
//
// Selection adds a leading checkbox column at every width (the app asked
// for selection; when a phone row runs out of room, Hidden columns are
// the lever, not the checkbox): the header box is the group
// parent for checkbox.min.js select-all; row boxes carry SelectionName
// ("" → "ids"), the row ID as value, and SelectionForm as their form
// attribute so the bulk form can live outside the table. HxTarget makes
// sort links also htmx requests (hx-get to the same URL, hx-target,
// hx-push-url="true").
//
// MaxHeight caps the wrapper (max-h-[240px|400px|600px] for HeightSm, Md,
// Lg; HeightNone/"" sets no cap) and the rows scroll inside it, at every
// width. StickyHeader makes the thead `sticky top-0 z-10 bg-card` (opaque,
// so rows scroll under it). Where it sticks depends on MaxHeight: with a
// cap, to the top of the wrapper's scroll box; without one, the wrapper
// becomes `md:overflow-visible` so on md and up (768px+) the header sticks
// to the page as the user scrolls past the table, while below md the
// wrapper keeps `overflow-auto` for horizontal overflow and the header only
// sticks within that box (Hidden columns are the lever that makes a phone
// table fit without needing to scroll it). Dense tightens cell padding to
// py-1.5. Empty is shown in one full-width cell when there are no rows; its
// zero value reads "Nothing here yet". Caption is a visually hidden
// <caption>.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Columns []Column
	Rows    []Row
	Spec    listquery.Spec
	Base    string

	HxTarget string

	Selection     bool
	SelectionName string
	SelectionForm string

	StickyHeader bool
	MaxHeight    Height
	Dense        bool
	Empty        emptystate.Props
	Caption      string
}

// selectionName resolves SelectionName's default.
func (p Props) selectionName() string {
	if p.SelectionName == "" {
		return "ids"
	}
	return p.SelectionName
}

// selectionGroup names the checkbox group: unique per table when the table
// has an ID, otherwise the selection name.
func (p Props) selectionGroup() string {
	if p.ID != "" {
		return p.ID + "-selection"
	}
	return p.selectionName()
}

// hasActions reports whether any row carries actions; the actions column
// is rendered only then, for every row, so cells line up.
func (p Props) hasActions() bool {
	for _, r := range p.Rows {
		if len(r.Actions) > 0 {
			return true
		}
	}
	return false
}

// columnCount is the total number of columns including selection and
// actions, used for the empty state's colspan.
func (p Props) columnCount() int {
	n := len(p.Columns)
	if p.Selection {
		n++
	}
	if p.hasActions() {
		n++
	}
	return n
}

// colspan returns columnCount as an attribute value.
func (p Props) colspan() string { return strconv.Itoa(p.columnCount()) }

// empty resolves the zero-value Empty props to the default title.
func (p Props) empty() emptystate.Props {
	e := p.Empty
	if e.Title == "" && e.Description == "" && e.Icon == "" {
		e.Title = "Nothing here yet"
	}
	return e
}

// rowID builds the <tr> id for a row, or "" when the row has no ID.
func (p Props) rowID(r Row) string {
	if r.ID == "" {
		return ""
	}
	if p.ID == "" {
		return "row-" + r.ID
	}
	return p.ID + "-row-" + r.ID
}

// cellAt returns the row's cell for column index i, or a zero Cell when the
// row is short.
func cellAt(r Row, i int) Cell {
	if i < len(r.Cells) {
		return r.Cells[i]
	}
	return Cell{}
}

// alignClass maps Align to a text alignment utility.
func alignClass(a Align) string {
	switch a {
	case AlignRight:
		return "text-right"
	case AlignCenter:
		return "text-center"
	default:
		return ""
	}
}

// ariaSort is the aria-sort value for a sortable column under spec.
func ariaSort(spec listquery.Spec, key string) string {
	if !spec.IsSorted(key) {
		return "none"
	}
	if spec.Dir == listquery.Desc {
		return "descending"
	}
	return "ascending"
}

// sortAttrs builds the attributes of one sort link: the href, and when
// HxTarget is set, the matching hx-get/hx-target/hx-push-url trio.
func (p Props) sortAttrs(url string) templ.Attributes {
	attrs := templ.Attributes{"href": url}
	if p.HxTarget != "" {
		attrs["hx-get"] = url
		attrs["hx-target"] = p.HxTarget
		attrs["hx-push-url"] = "true"
	}
	return attrs
}

// rowAction fills the defaults for an inline row action: ghost, small.
func rowAction(a action.Action) action.Action {
	if a.Variant == "" {
		a.Variant = action.VariantGhost
	}
	if a.Size == "" {
		a.Size = button.SizeSm
	}
	return a
}

// menuItemAttrs builds the hx-* attributes for a non-link action as
// dropdown item attributes, merged with the action's own Attributes. The
// dropdown item is itself the <button>, so action.Button is not nested
// inside it (a button inside a button is invalid HTML). Same shape as
// toolbar's helper.
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
