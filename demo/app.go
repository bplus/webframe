package demo

import (
	"net/http"

	"github.com/bplus/webframe/assets"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/flash"
)

// App is the demo's application shell configuration: name, navigation and
// theme toggle. Every showcase page is rendered through App.Render, and the
// server wraps its mux in App.Middleware so the shell can mark the active
// nav item. The Items pages are served through SidebarApp instead (see
// RegisterItemRoutes); the "Items" entry here links across to them. Flash
// is flash.Take, so a one-shot message set before a redirect shows once on
// the next page the shell renders.
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
		{Label: "Feedback", Href: "/#feedback"},
	},
	Flash: flash.Take,
}

// SidebarApp is the shell of the Items app (a dashboard sketch of
// 2026-10-21): the sidebar layout with the sketch's MINE and DISCOVER
// groups, every entry a view of the one list. RegisterItemRoutes wraps the
// Items handlers in SidebarApp.Middleware, so Respond renders them in this
// shell while the rest of the demo keeps the topbar. The Items entry
// carries Match so every /items/... page marks it active; the other
// entries are query-string views of the same path and are never active
// themselves (the shell compares paths). Flash is wired exactly as on App:
// the two shells share the one wf_flash cookie, so a flash set by an Items
// handler shows wherever the browser lands next.
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
	Flash: flash.Take,
}

// Routes mounts every demo route on mux: the embedded static files, the
// showcase and traffic pages, the Items app, the auth and error pages, the
// htmx endpoints, and last the catch-all that answers anything else with
// the 404 error page. cmd/demo and the tests call it so both serve the same
// table.
func Routes(mux *http.ServeMux) {
	// Static files, served from the assets package's embedded copy.
	assets.Mount(mux)

	// Pages.
	mux.HandleFunc("GET /{$}", HandleShowcase)
	mux.HandleFunc("GET /traffic", HandleTraffic)

	// The Items app (list, detail, create/edit form) on the sidebar shell:
	// GET /items, /items/new, /items/{id}, /items/{id}/edit,
	// /items/{id}/history; POST /items, /items/{id}, /items/validate,
	// /items/publish; DELETE /items/{id}; GET /sidebar → /items.
	RegisterItemRoutes(mux)

	// The pages outside the app's own content: sign in, signed out,
	// forbidden (no shell) and the three error pages (in the shell).
	mux.HandleFunc("GET /signin", HandleSignIn)
	mux.HandleFunc("POST /signin", HandleSignIn)
	mux.HandleFunc("GET /signed-out", HandleSignedOut)
	mux.HandleFunc("GET /forbidden", HandleForbidden)
	mux.HandleFunc("GET /errors/{code}", HandleErrorPage)

	// htmx demo endpoints.
	mux.HandleFunc("GET /api/traffic/stream", HandleTrafficStream)
	mux.HandleFunc("POST /api/click", HandleClick)
	mux.HandleFunc("GET /api/search", HandleSearch)
	mux.HandleFunc("POST /api/validate", HandleValidate)
	mux.HandleFunc("POST /api/picked", HandlePicked)
	mux.HandleFunc("POST /api/toast", HandleToast)
	mux.HandleFunc("POST /api/flash", HandleFlash)
	mux.HandleFunc("GET /api/slow", HandleSlow)

	// Everything else is the 404 page. "/" without the {$} suffix matches
	// every path no other pattern claims.
	mux.HandleFunc("/", HandleNotFound)
}

// Handler returns the demo server's whole handler: Routes on a fresh mux,
// wrapped in App.Middleware so every page (the Items pages re-wrap
// themselves in SidebarApp.Middleware) renders in a shell.
func Handler() http.Handler {
	mux := http.NewServeMux()
	Routes(mux)
	return App.Middleware(mux)
}
