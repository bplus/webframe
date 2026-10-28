package demo

import "github.com/bplus/webframe/components/appshell"

// App is the demo's application shell configuration: name, navigation and
// theme toggle. Every showcase page is rendered through App.Render, and the
// server wraps its mux in App.Middleware so the shell can mark the active
// nav item. The Items pages are served through SidebarApp instead (see
// RegisterItemRoutes); the "Items" entry here links across to them.
var App = &appshell.App{
	Name:        "Webframe",
	HomeHref:    "/",
	ThemeToggle: true,
	MaxVisible:  4,
	Nav: []appshell.NavItem{
		{Label: "Showcase", Href: "/"},
		{Label: "Items", Href: "/items", Match: "/items"},
		{Label: "Traffic", Href: "/traffic"},
		{Label: "Forms", Href: "/#forms"},
		{Label: "Interactive", Href: "/#interactive"},
		{Label: "Patterns", Href: "/#patterns"},
		{Label: "Layout", Href: "/#layout"},
		{Label: "Actions", Href: "/#actions"},
	},
}

// SidebarApp is the shell of the Items app (a dashboard sketch of
// 2026-10-21): the sidebar layout with the sketch's MINE and DISCOVER
// groups, every entry a view of the one list. RegisterItemRoutes wraps the
// Items handlers in SidebarApp.Middleware, so Respond renders them in this
// shell while the rest of the demo keeps the topbar. The Items entry
// carries Match so every /items/... page marks it active; the other
// entries are query-string views of the same path and are never active
// themselves (the shell compares paths).
var SidebarApp = &appshell.App{
	Name:        "Webframe",
	HomeHref:    "/",
	Layout:      appshell.LayoutSidebar,
	ThemeToggle: true,
	Groups: []appshell.NavGroup{
		{Title: "Mine", Items: []appshell.NavItem{
			{Label: "Items", Href: "/items", Match: "/items"},
			{Label: "Mods", Href: "/items?type=weapon"},
			{Label: "Drafts", Href: "/items?status=draft"},
			{Label: "Builds", Href: "/items?status=published"},
		}},
		{Title: "Discover", Items: []appshell.NavItem{
			{Label: "Trending", Href: "/items?sort=updated&dir=desc"},
			{Label: "New", Href: "/items?sort=updated&dir=desc&per=5"},
			{Label: "Saved", Href: "/items?type=accessory"},
		}},
	},
}
