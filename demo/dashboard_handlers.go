package demo

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/cardgrid"
	"github.com/bplus/webframe/components/dashboardpage"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/stattile"
	"github.com/bplus/webframe/components/timeline"
)

// dashboardArt is the art every card on the dashboard shows: the embedded
// 16:9 placeholder, which fits cardgrid's default (video) frame.
const dashboardArt = "/static/img/placeholder-wide.svg"

// Dashboard sizes: how many items the timeline and the card grid show.
const (
	dashboardRecent = 8
	dashboardNewest = 4
)

// HandleDashboard renders the Items dashboard (GET /dashboard): four stat
// tiles computed from the store, a timeline of the most recent changes and
// a card grid of the newest items, through dashboardpage.Respond.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	if err := dashboardpage.Respond(w, r, "Dashboard", dashboardProps(time.Now())); err != nil {
		log.Printf("demo: render dashboard: %v", err)
	}
}

// dashboardProps declares the dashboard from the store as it is now. The
// tiles (items, published, drafts, packs) take the default span of 3, so
// they fill one row of four on md and up. The two sections sit side by
// side there: Recent changes (a timeline, span 4) beside Newest items (a
// two-column card grid, span 8); below 900px dashboardgrid stacks every
// tile and section one per row.
func dashboardProps(now time.Time) dashboardpage.Props {
	items := newestFirst(Items.All())
	total, published, drafts := Items.Counts()
	packs := packNames(items)

	return dashboardpage.Props{
		ID:     "dashboard",
		Header: pageheader.Props{Title: "Dashboard", Subtitle: "What the Items store holds and what changed last."},
		Tiles: []dashboardpage.Tile{
			{Props: stattile.Props{Label: "Items", Value: strconv.Itoa(total), Sub: strconv.Itoa(published) + " published · " + plural(drafts, "draft"), Href: "/items"}},
			{Props: stattile.Props{Label: "Published", Value: strconv.Itoa(published), Sub: "Visible to other builders", Href: "/items?status=published"}},
			{Props: stattile.Props{Label: "Drafts", Value: strconv.Itoa(drafts), Sub: "Not yet published", Href: "/items?status=draft", Variant: stattile.VariantMuted}},
			{Props: stattile.Props{Label: "Packs", Value: strconv.Itoa(len(packs)), Sub: packsLine(packs)}},
		},
		Sections: []dashboardpage.Section{
			{
				ID:          "recent-changes",
				Title:       "Recent changes",
				Description: "The last " + plural(min(dashboardRecent, len(items)), "update") + ", newest first.",
				ColSpan:     4,
				Content:     timeline.Timeline(recentChanges(items, now)),
			},
			{
				ID:          "newest-items",
				Title:       "Newest items",
				Description: "The most recently updated items as cards; open one or edit it from the card.",
				ColSpan:     8,
				Actions:     []action.Action{{Label: "All items", Href: "/items"}},
				Content:     cardgrid.CardGrid(newestCards(items, now)),
			},
		},
	}
}

// newestFirst returns items sorted by Updated, newest first (stable, so
// equal times keep store order).
//
// @pure
func newestFirst(items []Item) []Item {
	out := make([]Item, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out
}

// packNames returns the distinct pack names items are used in, in first
// seen order.
//
// @pure
func packNames(items []Item) []string {
	seen := make(map[string]bool)
	var out []string
	for _, it := range items {
		if it.UsedIn != "" && !seen[it.UsedIn] {
			seen[it.UsedIn] = true
			out = append(out, it.UsedIn)
		}
	}
	return out
}

// packsLine is the Packs tile's sub line: the pack names joined, or a note
// when no item is in a pack.
//
// @pure
func packsLine(packs []string) string {
	if len(packs) == 0 {
		return "No item is in a pack"
	}
	return strings.Join(packs, " · ")
}

// recentChanges declares the Recent changes timeline: one event per item
// for the dashboardRecent most recently updated, titled with the item's
// name and linking to it, the body naming the status and type, the dot
// toned by the status (published is the success tone, draft secondary),
// and the pack as the meta line. An empty store shows the empty state.
func recentChanges(items []Item, now time.Time) timeline.Props {
	var events []timeline.Item
	for _, it := range items[:min(dashboardRecent, len(items))] {
		meta := ""
		if it.UsedIn != "" {
			meta = "In " + it.UsedIn
		}
		events = append(events, timeline.Item{
			At:      it.Updated,
			Title:   it.Name + " updated",
			Body:    badge.StatusLabel(it.Status) + " " + strings.ToLower(itemTypeLabel(it.Type)) + " · " + relativeTime(it.Updated, now) + " ago",
			Href:    "/items/" + it.ID,
			Variant: badge.Status(it.Status).Variant,
			Meta:    meta,
		})
	}
	return timeline.Props{
		ID:    "recent-changes-timeline",
		Items: events,
		Empty: emptystate.Props{Title: "No changes yet", Description: "Create an item and it shows up here."},
	}
}

// newestCards declares the Newest items card grid: the dashboardNewest
// most recently updated items, each with the placeholder art, the type as
// subtitle, the status badge, the pack and age as meta, and an Edit action.
// Two columns at most, since the section takes 8 of 12 columns.
func newestCards(items []Item, now time.Time) cardgrid.Props {
	var cards []cardgrid.Card
	for _, it := range items[:min(dashboardNewest, len(items))] {
		usedIn := "Not in a pack"
		if it.UsedIn != "" {
			usedIn = it.UsedIn
		}
		cards = append(cards, cardgrid.Card{
			ID:       "card-item-" + it.ID,
			Title:    it.Name,
			Subtitle: itemTypeLabel(it.Type),
			Href:     "/items/" + it.ID,
			Art:      &cardgrid.Art{Src: dashboardArt, Alt: ""},
			Badges:   []cardgrid.Badge{cardgrid.StatusBadge("", it.Status)},
			Meta:     []string{usedIn, "Updated " + relativeTime(it.Updated, now) + " ago"},
			Actions:  []action.Action{{Label: "Edit", Href: "/items/" + it.ID + "/edit"}},
		})
	}
	return cardgrid.Props{
		ID:      "newest-items-grid",
		Items:   cards,
		Columns: 2,
		Empty:   emptystate.Props{Title: "No items yet", Action: &action.Action{Label: "+ Create item", Href: "/items/new"}},
	}
}
