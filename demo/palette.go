package demo

import (
	"net/http"

	"github.com/bplus/webframe/components/commandpalette"
)

// demoPaletteFor is the command palette both shells carry (App and
// SidebarApp set PaletteFor to it): the five pages of the Items app in the
// untitled first group, then every item in the store under "Items", with
// the item's type as hidden keywords so typing "weapon" lists the weapons.
// It is built per shell render from the store, so an item created or
// deleted appears in (or leaves) the palette on the next page.
func demoPaletteFor(*http.Request) *commandpalette.Props {
	return paletteProps(Items.All())
}

// paletteProps declares the palette for items.
//
// @pure
func paletteProps(items []Item) *commandpalette.Props {
	entries := []commandpalette.Item{
		{Label: "Dashboard", Href: "/dashboard", Keywords: "home tiles overview"},
		{Label: "Items", Href: "/items", Keywords: "list mods table"},
		{Label: "Browse", Href: "/browse", Keywords: "split pane"},
		{Label: "Settings", Href: "/settings", Keywords: "profile account environment"},
		{Label: "Logs", Href: "/logs", Keywords: "log viewer tail"},
	}
	for _, it := range items {
		entries = append(entries, commandpalette.Item{
			Label:    it.Name,
			Href:     "/items/" + it.ID,
			Keywords: it.Type,
			Group:    "Items",
		})
	}
	return &commandpalette.Props{
		Items:       entries,
		Placeholder: "Jump to a page or an item…",
	}
}
