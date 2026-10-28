package demo

import "github.com/bplus/webframe/components/appshell"

// App is the demo's application shell configuration: name, navigation and
// theme toggle. Every demo page is rendered through App.Render, and the
// server wraps its mux in App.Middleware so the shell can mark the active
// nav item.
var App = &appshell.App{
	Name:        "Webframe",
	HomeHref:    "/",
	ThemeToggle: true,
	MaxVisible:  4,
	Nav: []appshell.NavItem{
		{Label: "Showcase", Href: "/"},
		{Label: "Traffic", Href: "/traffic"},
		{Label: "Sidebar", Href: "/sidebar"},
		{Label: "Forms", Href: "/#forms"},
		{Label: "Interactive", Href: "/#interactive"},
		{Label: "Patterns", Href: "/#patterns"},
		{Label: "Layout", Href: "/#layout"},
		{Label: "Actions", Href: "/#actions"},
	},
}

// SidebarApp renders the /sidebar demo page with the sidebar layout. It
// shares the server's middleware with App (Render takes its nav from the
// App it is called on, and the active item from the request path).
var SidebarApp = &appshell.App{
	Name:        "Webframe",
	HomeHref:    "/",
	Layout:      appshell.LayoutSidebar,
	ThemeToggle: true,
	Groups: []appshell.NavGroup{
		{Title: "Mine", Items: []appshell.NavItem{
			{Label: "Items", Href: "/sidebar"},
			{Label: "Mods", Href: "/sidebar#mods"},
			{Label: "Drafts", Href: "/sidebar#drafts"},
		}},
		{Title: "Discover", Items: []appshell.NavItem{
			{Label: "Trending", Href: "/sidebar#trending"},
			{Label: "New", Href: "/sidebar#new"},
			{Label: "Saved", Href: "/sidebar#saved"},
		}},
	},
}
