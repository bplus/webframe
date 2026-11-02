package demo

import (
	"log"
	"net/http"
	"time"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/descriptionlist"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/splitpane"
	"github.com/bplus/webframe/demo/pages"
)

// browseID is the splitpane ID; its detail region is browseDetailID.
const (
	browseID       = "browse"
	browseDetailID = browseID + "-detail"
)

// HandleBrowse renders the two-pane browser with no item selected (GET
// /browse): the list of item names on the left and, from md up, the
// placeholder in the detail region; on a phone the list alone.
func HandleBrowse(w http.ResponseWriter, r *http.Request) {
	if err := splitpane.Respond(w, r, "Browse", browseProps("", nil)); err != nil {
		log.Printf("demo: render browse: %v", err)
	}
}

// HandleBrowseItem renders the browser with one item selected (GET
// /browse/{id}): the same list, the item's description list on the right.
// A list click from md up arrives as an htmx request targeting the detail
// region and receives the detail alone; a phone's click, a reload or a
// bookmark gets the whole page, which shows the detail with a Back link.
// An unknown id is the 404 error page.
func HandleBrowseItem(w http.ResponseWriter, r *http.Request) {
	it, ok := Items.Get(r.PathValue("id"))
	if !ok {
		notFound(w, r)
		return
	}
	detail := pages.BrowseDetail(it.Name, "/items/"+it.ID, browseSummary(it, time.Now()))
	if err := splitpane.Respond(w, r, it.Name+" — Browse", browseProps(it.ID, detail)); err != nil {
		log.Printf("demo: render browse %s: %v", it.ID, err)
	}
}

// browseProps declares the pane: the list column is splitpane.List with one
// row per item (name, type as the meta line), the item whose id is
// selectedID marked selected. Detail is nil on /browse and the item's
// detail on /browse/{id}; Back leads to /browse.
func browseProps(selectedID string, detail templ.Component) splitpane.Props {
	p := splitpane.Props{
		ID:          browseID,
		DetailID:    browseDetailID,
		Detail:      detail,
		BackHref:    "/browse",
		BackLabel:   "All items",
		Title:       "Item details",
		Placeholder: emptystate.EmptyState(emptystate.Props{Title: "Select an item", Description: "Pick one from the list to see its details here."}),
	}
	var items []splitpane.ListItem
	for _, it := range Items.All() {
		items = append(items, splitpane.ListItem{
			Href:     "/browse/" + it.ID,
			Title:    it.Name,
			Meta:     itemTypeLabel(it.Type),
			Selected: it.ID == selectedID,
		})
	}
	p.List = splitpane.List(p, items)
	return p
}

// browseSummary is the item's description list for the detail region: the
// same facts the detail page's summary shows.
func browseSummary(it Item, now time.Time) []descriptionlist.Item {
	usedIn := descriptionlist.Item{Label: "Used in", Value: it.UsedIn}
	if it.UsedIn == "" {
		usedIn = descriptionlist.Item{Label: "Used in", Value: "—", Muted: true}
	}
	return []descriptionlist.Item{
		{Label: "Type", Value: itemTypeLabel(it.Type)},
		descriptionlist.BadgeItem("Status", badge.StatusLabel(it.Status), it.Status),
		usedIn,
		{Label: "Updated", Value: relativeTime(it.Updated, now) + " ago · " + it.Updated.Format("Jan 2, 2006")},
		{Label: "ID", Value: it.ID, Mono: true, Copyable: true},
	}
}
