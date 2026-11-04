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
// Sticky header: with MaxHeight, the wrapper is a capped scroll box and the
// header sticks to its top at every width. Without MaxHeight, on md and up
// the wrapper stops clipping so the header sticks to the page (offset by
// the shell's --wf-sticky-top); below md the wrapper keeps its horizontal
// scroll, so the header sticks only within that box. A header stuck to the
// page draws the table's top border itself and the wrapper is square, so
// the header looks the same at rest and once stuck (a rounded wrapper's top
// edge would scroll away and leave a borderless strip).
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

// Align is a horizontal text alignment. On a Column the zero value is
// left; on a Cell it means "use the column's".
type Align string

const (
	// AlignLeft is explicit left alignment, for a cell in a right or
	// center column.
	AlignLeft Align = "left"
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
//
// Align overrides the column's alignment for this cell only; the zero value
// inherits it. Width (Shrink) and phone visibility (Hidden) belong to the
// whole column, so they are set on Column only.
type Cell struct {
	Text, Href string
	Mono       bool
	Muted      bool
	Align      Align
	// Badge renders Text inside a badge.Badge with these props.
	Badge *badge.Props
	// Component replaces the cell content entirely.
	Component templ.Component
}

// BadgeCell is a Cell showing text inside a status badge whose variant is
// picked from kind by badge.StatusPtr (unknown words get outline). Pass
// badge.StatusLabel(kind) as text to display the status word itself.
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

// Props configures DataTable. Spec is the current sort state; header links
// are Spec.URL(Base, listquery.WithSort(key)). An empty Base falls back to
// the request path stored by appshell.Middleware, else to a query-only href.
//
// Selection adds a leading checkbox column (the header box drives
// checkbox.min.js select-all). Row boxes are named SelectionName (default
// "ids") and carry SelectionForm as their form attribute, so the bulk form
// can live outside the table. HxTarget makes sort links htmx requests that
// push the URL.
//
// StickyHeader pins the thead; see the package doc for where it sticks.
// Empty is shown in one full-width cell when there are no rows; its zero
// value reads "Nothing here yet". Caption is visually hidden.
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

func (p Props) colspan() string { return strconv.Itoa(p.columnCount()) }

// empty resolves the zero-value Empty props to the default title.
func (p Props) empty() emptystate.Props {
	e := p.Empty
	if e.Title == "" && e.Description == "" && e.Icon == nil {
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

// cellAlign is the alignment a cell renders with: its own, else the column's.
func cellAlign(col Column, c Cell) Align {
	if c.Align != "" {
		return c.Align
	}
	return col.Align
}

func alignClass(a Align) string {
	switch a {
	case AlignLeft:
		return "text-left"
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
	if a.Variant == action.VariantDefault {
		a.Variant = action.VariantGhost
	}
	if a.Size == "" {
		a.Size = button.SizeSm
	}
	return a
}
