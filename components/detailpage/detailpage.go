// Package detailpage is the page type for one record: a page header, a
// summary description list, optional tabs (inline or lazily loaded over
// htmx), optional free content, and an optional right rail. App code
// declares the page as a Props value and calls Respond; it writes no
// markup of its own.
package detailpage

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/tabs"
)

// Tab is one tab of the detail page. ID is the tab value (unique within the
// page) and also names the panel element: ID+"-panel". Content is rendered
// inline inside the panel. HxGet makes the tab lazy instead: the panel is
// rendered empty and the tab's Trigger carries hx-get=HxGet,
// hx-target="#"+ID+"-panel" and hx-trigger="click once" (or "load" when the
// tab is the active one, so an active lazy tab is never blank). The
// handler behind HxGet returns the panel body as a plain fragment (no tabs
// wrapper needed; it is swapped into the panel's innerHTML). Content is
// ignored when HxGet is set.
//
// The trigger is a click, not "revealed" or "intersect": under tabs'
// default (tallest) sizing inactive panels are laid out, only
// visibility-hidden, so a viewport-based trigger would fire at page load.
type Tab struct {
	ID, Label string
	Content   templ.Component
	HxGet     string
}

// Lazy reports whether the tab loads its panel over htmx.
func (t Tab) Lazy() bool { return t.HxGet != "" }

// PanelID is the element id of the tab's panel: ID+"-panel".
func (t Tab) PanelID() string { return t.ID + "-panel" }

// triggerAttrs returns the htmx attributes for the tab's Trigger: none for
// an inline tab; hx-get/hx-target/hx-trigger for a lazy one. The active lazy
// tab fires on "load" (and never again); an inactive one on its first click.
func (t Tab) triggerAttrs(active bool) templ.Attributes {
	if !t.Lazy() {
		return nil
	}
	trigger := "click once"
	if active {
		trigger = "load"
	}
	return templ.Attributes{
		"hx-get":     t.HxGet,
		"hx-target":  "#" + t.PanelID(),
		"hx-trigger": trigger,
	}
}

// Props declares a detail page. Header is the pageheader; Summary is the
// description list directly under it (SummaryColumns 1 or 2; zero means 1);
// Tabs render below the summary, ActiveTab naming the open one ("" means
// the first); TabsSizing is passed to tabs.Tabs (default tallest). With
// Aside set, everything after the header sits in a layout.TwoColumn with
// AsideWidth and AsideSticky. Class is merged onto the root with
// utils.TwMerge; Attributes spread last on the root element.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	Header         pageheader.Props
	Summary        []descriptionlist.Item
	SummaryColumns int
	Tabs           []Tab
	ActiveTab      string
	TabsSizing     tabs.Sizing
	Aside          templ.Component
	AsideWidth     layout.AsideWidth
	AsideSticky    bool
}

// TabsID is the tab group id: ID+"-tabs", or "" (tabs picks a random one)
// when ID is empty.
func (p Props) TabsID() string {
	if p.ID == "" {
		return ""
	}
	return p.ID + "-tabs"
}

// activeTab returns the ID of the open tab: ActiveTab when it names a tab,
// otherwise the first tab's ID ("" with no tabs).
func (p Props) activeTab() string {
	for _, t := range p.Tabs {
		if t.ID == p.ActiveTab {
			return t.ID
		}
	}
	if len(p.Tabs) > 0 {
		return p.Tabs[0].ID
	}
	return ""
}

func (p Props) summaryProps() descriptionlist.Props {
	return descriptionlist.Props{Items: p.Summary, Columns: p.SummaryColumns}
}

func (p Props) tabsProps() tabs.Props {
	return tabs.Props{ID: p.TabsID(), Sizing: p.TabsSizing}
}

func (p Props) twoColumnProps() layout.TwoColumnProps {
	return layout.TwoColumnProps{Aside: p.Aside, AsideWidth: p.AsideWidth, Sticky: p.AsideSticky}
}

// Respond writes the detail page for r: the full document (shell and all)
// for a normal or boosted request, the bare page body for a plain htmx
// request, per appshell's Render rule. The App comes from the request
// context, so the handler chain must include (*appshell.App).Middleware;
// otherwise Respond returns the error from appshell.Must without writing.
// Lazy tab panels are not served here: the app registers its own handler
// for each Tab.HxGet URL and returns the panel body as a fragment.
//
// Respond renders Page(p) with no children. To put free content under the
// summary or tabs from a handler, render Page(p) yourself with
// templ.WithChildren and pass it to app.Render.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	return app.Render(w, r, title, Page(p))
}
