// Package streampage is the page type for a live feed: a page header, an
// optional intro line, one trafficstream panel and
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
// The stream's own search box and selects are its filters; streampage adds
// no second filter bar (a bar that drives nothing would be a dead control;
// a bridge from filterbar to the panel's controls is a batch 5 input).
//
// Script: Page renders trafficstream.Script() once, after the content,
// because appshell.Scripts deliberately leaves it out of the shared
// bundle. A page that already loads the script another way (two stream
// pages' fragments swapped into one document, a custom head) sets NoScript.
// trafficstream.min.js marks each panel it has initialized, so a repeated
// load does not double-connect a panel.
//
// Phone width (390px, 358px of content inside the shell's gutters): the
// header, intro, stream and aside stack in one column (the
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
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/trafficstream"
)

// Props declares a stream page. Header is the pageheader; Intro is one
// muted line under it; Stream configures the trafficstream panel as the
// app would call it directly; Aside puts everything under the header
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
	Aside       templ.Component
	AsideWidth  layout.AsideWidth
	AsideSticky bool
	NoScript    bool
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
