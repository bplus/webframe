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
		{Label: "Forms", Href: "/#forms"},
		{Label: "Interactive", Href: "/#interactive"},
		{Label: "Patterns", Href: "/#patterns"},
		{Label: "Layout", Href: "/#layout"},
		{Label: "Actions", Href: "/#actions"},
	},
}
