// Package detailpage is the page type for one record: a page header, a
// summary description list, optional tabs (inline or lazily loaded over
// htmx), optional free content, and an optional right rail. App code
// declares the page as a Props value and calls Respond; it writes no
// markup of its own.
package detailpage

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/tabs"
)

// Tab is one tab of the detail page. ID must be unique within the page and
// also names the panel element (see PanelID). Content renders inline in the
// panel. Setting HxGet makes the tab lazy instead (Content is ignored): the
// panel renders empty and is filled on the tab's first click, or at page
// load when it is the active tab. The HxGet handler returns the panel body
// as a plain fragment.
//
// The trigger is a click, not "revealed": under tabs' default sizing,
// inactive panels are laid out (only visibility-hidden), so a
// viewport-based trigger would fire at page load.
type Tab struct {
	ID, Label string
	Content   templ.Component
	HxGet     string
}

// Lazy reports whether the tab loads its panel over htmx.
func (t Tab) Lazy() bool { return t.HxGet != "" }

// PanelID is the element id of the tab's panel: ID+"-panel".
func (t Tab) PanelID() string { return t.ID + "-panel" }

// triggerAttrs returns the htmx attributes for a lazy tab's Trigger, nil
// for an inline one.
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

// Props declares a detail page. Summary is the description list under the
// header. ActiveTab names the open tab; empty or unknown means the first.
// With Aside set, everything after the header sits in a layout.TwoColumn.
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

// Respond writes the detail page for r via appshell's Render: the full
// document for a normal or boosted request, the bare body for a plain htmx
// request. The handler chain must include (*appshell.App).Middleware;
// otherwise Respond returns an error without writing. Lazy tab panels are
// served by the app's own handlers for each Tab.HxGet.
//
// Respond renders Page(p) with no children. For free content under the
// summary or tabs, render Page(p) with templ.WithChildren and pass it to
// app.Render.
func Respond(w http.ResponseWriter, r *http.Request, title string, p Props) error {
	app, err := appshell.Must(r.Context())
	if err != nil {
		return err
	}
	if err := action.ValidateAll(p.Header.Actions()); err != nil {
		return err
	}
	return app.Render(w, r, title, Page(p))
}
