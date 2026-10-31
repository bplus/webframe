// Package dashboardpage is the page type for a dashboard: a page header,
// a row of stat tiles and then titled sections, all placed on one
// dashboardgrid. App code declares the page as a Props value and calls
// Respond; it writes no markup and makes no layout decision beyond each
// item's column span.
//
// The header is a pageheader.Props, not dashboardheader.Props as the spec
// row (§6.3) first said: pageheader is the page layer's header (the h1 that
// detailpage, listpage and formpage also render, with breadcrumbs, back
// link, status badge and actions), so a dashboard page's title bar reads
// like every other page's. dashboardheader stays as the standalone
// dashboard's header for apps that render a dashboard outside the page
// layer.
package dashboardpage

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/dashboardgrid"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/stattile"
)

// Default spans and heading level, applied when the zero value is given.
const (
	defaultTileSpan    = 3  // four tiles across a 12-column grid
	defaultSectionSpan = 12 // a section takes the full row
	defaultLevel       = 2  // h2 under the page's h1
)

// Tile is one stat tile on the grid: a stattile.Props (embedded, so a Tile
// literal names the tile's fields directly) and its column span on the
// 12-column grid. ColSpan 0 means 3, so four tiles fill a row; 4 gives
// three across, 6 two across. dashboardgrid clamps the span to [1, 12].
type Tile struct {
	stattile.Props
	ColSpan int
}

func (t Tile) colSpan() int {
	if t.ColSpan == 0 {
		return defaultTileSpan
	}
	return t.ColSpan
}

func (t Tile) slotProps() dashboardgrid.SlotProps {
	return dashboardgrid.SlotProps{ColSpan: t.colSpan()}
}

// Section is one titled block on the grid, rendered as a layout.Section
// holding Content. ColSpan 0 means 12 (the full row); 8 beside a 4 puts
// two sections side by side on md and up. Level is the heading level of
// Title: 0 means 2 (an h2 under the page's h1); 3 and 4 nest, as in
// layout.SectionProps. ID goes on the <section> element. Actions render on
// the heading row's right. A nil Content renders the heading row alone.
type Section struct {
	ID          string
	Title       string
	Description string
	ColSpan     int
	Level       int
	Actions     []action.Action
	Content     templ.Component
}

func (s Section) colSpan() int {
	if s.ColSpan == 0 {
		return defaultSectionSpan
	}
	return s.ColSpan
}

func (s Section) level() int {
	if s.Level == 0 {
		return defaultLevel
	}
	return s.Level
}

func (s Section) slotProps() dashboardgrid.SlotProps {
	return dashboardgrid.SlotProps{ColSpan: s.colSpan()}
}

func (s Section) sectionProps() layout.SectionProps {
	return layout.SectionProps{
		ID:          s.ID,
		Title:       s.Title,
		Description: s.Description,
		Actions:     s.Actions,
		Level:       s.level(),
	}
}

// Props declares a dashboard page. Header is the pageheader. Tiles come
// first on the grid, in order, then Sections, in order; both are optional
// and the grid is omitted when both are empty. Gap is the grid gap in
// pixels (dashboardgrid's default, 16, when 0). Class is merged onto the
// root with utils.TwMerge; Attributes spread last on the root element.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Header   pageheader.Props
	Tiles    []Tile
	Sections []Section
	Gap      int
}

func (p Props) hasGrid() bool { return len(p.Tiles) > 0 || len(p.Sections) > 0 }

// gridProps is the dashboardgrid: Flush, because the page's own gutter
// (the shell's content padding) already insets it; the grid's 32px page
// margin would double it.
func (p Props) gridProps() dashboardgrid.Props {
	return dashboardgrid.Props{Flush: true, Gap: p.Gap}
}

// Respond writes the dashboard page for r: the full document (shell and
// all) for a normal or boosted request, the bare page body for a plain
// htmx request, per appshell's Render rule. The App comes from the request
// context, so the handler chain must include (*appshell.App).Middleware;
// otherwise Respond returns the error from appshell.Must without writing.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	return app.Render(w, r, title, Page(p))
}
