// Package streampage is the page type for a live feed: a page header, an
// optional intro line, an optional filter bar, one trafficstream panel and
// an optional right rail, plus the trafficstream script, so an operator
// view of an event stream is one Props value and one Respond call. App
// code writes no markup of its own.
//
// The stream is trafficstream.TrafficStream exactly as the app configures
// it in Props.Stream (stream URL, height, options, initial rows); the page
// adds nothing to the panel and takes nothing away. The panel keeps its own
// search box and provider/event/direction selects, which filter its rows
// client-side.
//
// Filters is a companion filterbar.FilterBar above the stream. It is
// client-side only: the bar renders with no hx-get, so filterbar.min.js
// (loaded by appshell.Scripts) issues no request when a control changes;
// it dispatches the bubbling "filterbar-change" CustomEvent (detail.params
// is a URLSearchParams of the controls' values) and mirrors those values
// into the page URL with history.replaceState. Nothing in trafficstream
// listens for that event today: an app that wants the bar to drive the
// stream wires the event to the panel's own controls itself. The stream's
// controls stay whether or not Filters is set.
//
// Script: Page renders trafficstream.Script() once, after the content,
// because appshell.Scripts deliberately leaves it out of the shared
// bundle. A page that already loads the script another way (two stream
// pages' fragments swapped into one document, a custom head) sets NoScript.
// trafficstream.min.js marks each panel it has initialized, so a repeated
// load does not double-connect a panel.
//
// Phone width (390px, 358px of content inside the shell's gutters): the
// header, intro, filter bar, stream and aside stack in one column (the
// aside under the stream: layout.TwoColumn's md:flex-row). trafficstream
// needs no change for that width, measured against its markup: its
// title/actions row is flex-col below md with the badge and the two
// buttons in a flex-wrap row (about 270px together); its search and three
// selects are a one-column grid below md (the md:grid-cols-[...] fixed
// 180px/140px tracks apply from md up only), so each control takes the
// full width; the table sits in its own overflow-auto box of Props.Stream.
// Height pixels, so its seven columns scroll horizontally inside the panel
// while the page itself never overflows (the same arrangement datatable
// uses below md). Height is left as the app sets it: the box scrolls on
// its own, so a tall panel costs page scroll, not width.
package streampage

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/filterbar"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/trafficstream"
)

// Props declares a stream page. Header is the pageheader; Intro is one
// muted line under it; Stream configures the trafficstream panel as the
// app would call it directly; Filters, when set, render a client-side
// filterbar between the intro and the stream (see the package doc for
// what it does and does not do); Aside puts everything under the header
// in the main column of a layout.TwoColumn whose aside is Aside, sized by
// AsideWidth and pinned by AsideSticky. NoScript leaves out the
// trafficstream.Script() tag Page otherwise renders once. Class is merged
// onto the root with utils.TwMerge; Attributes spread last on the root
// element, which carries id ID only when ID is set.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Header      pageheader.Props
	Intro       string
	Stream      trafficstream.Props
	Filters     []filterbar.FilterConfig
	Aside       templ.Component
	AsideWidth  layout.AsideWidth
	AsideSticky bool
	NoScript    bool
}

// FiltersID is the filter bar's element id: ID+"-filters", or "" (no id
// attribute) when ID is empty.
func (p Props) FiltersID() string {
	if p.ID == "" {
		return ""
	}
	return p.ID + "-filters"
}

// hasFilters reports whether the companion filter bar renders.
func (p Props) hasFilters() bool { return len(p.Filters) > 0 }

// filterbarProps builds the companion bar: the configs as given, the id
// from FiltersID, and no HxGet/HxTarget, so filterbar.min.js dispatches
// its change event and syncs the URL without issuing a request.
func (p Props) filterbarProps() filterbar.Props {
	return filterbar.Props{ID: p.FiltersID(), Filters: p.Filters}
}

func (p Props) twoColumnProps() layout.TwoColumnProps {
	return layout.TwoColumnProps{Aside: p.Aside, AsideWidth: p.AsideWidth, Sticky: p.AsideSticky}
}

// Respond writes the stream page for r: the full document (shell and all)
// for a normal or boosted request, the bare page body for a plain htmx
// request, per appshell's Render rule. The App comes from the request
// context, so the handler chain must include (*appshell.App).Middleware;
// otherwise Respond returns the error from appshell.Must without writing.
// The SSE endpoint named by Stream.StreamURL is the app's own handler; the
// page only points the panel at it.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	return app.Render(w, r, title, Page(p))
}
