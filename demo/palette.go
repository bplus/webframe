package demo

import (
	"github.com/bplus/webframe/components/commandpalette"
)

// demoPalette is the command palette both shells carry (App and
// SidebarApp set Palette to it): the five pages of the Items app in the
// untitled first group, then every item in the store under "Items", with
// the item's type as hidden keywords so typing "weapon" lists the weapons.
// It is built once, when the package loads, from the seeded store: the
// App.Palette field is a value, not a per-request function, so an item
// created or deleted afterwards is not reflected until the server restarts.
var demoPalette = paletteProps(Items.All())

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
