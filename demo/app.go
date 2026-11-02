package demo

import (
	"net/http"

	"github.com/bplus/webframe/assets"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/errorpage"
	"github.com/bplus/webframe/components/flash"
)

// App is the demo's application shell configuration: name, navigation and
// theme toggle. Every showcase page is rendered through App.Render, and the
// server wraps its mux in App.Middleware so the shell can mark the active
// nav item. The Items pages are served through SidebarApp instead (see
// RegisterItemRoutes); the "Items" entry here links across to them. Flash
// is flash.Take, so a one-shot message set before a redirect shows once on
// the next page the shell renders. PaletteFor builds the same command
// palette as the Items app's per request, so the showcase's palette section
// can be tried where it is read; OnPanic answers a handler panic with the
// styled 500 page.
var App = &appshell.App{
	Name:        "Webframe",
	HomeHref:    "/",
	ThemeToggle: true,
	MaxVisible:  4,
	Nav: []appshell.NavItem{
		{Label: "Showcase", Href: "/"},
		{Label: "Items", Href: "/items", Match: "/items"},
		{Label: "Forms", Href: "/#forms"},
		{Label: "Content", Href: "/#content"},
		{Label: "Interactive", Href: "/#interactive"},
		{Label: "Patterns", Href: "/#patterns"},
		{Label: "Layout", Href: "/#layout"},
		{Label: "Actions", Href: "/#actions"},
		{Label: "Feedback", Href: "/#feedback"},
	},
	Flash:      flash.Take,
	PaletteFor: demoPaletteFor,
	OnPanic:    errorpage.PanicHandler,
}

// SidebarApp is the shell of the Items app: the sidebar layout with MINE
// and DISCOVER groups. MINE opens with the Dashboard, Browse and Settings pages
// and the Items list; the remaining entries are views of the one list.
// RegisterItemRoutes and RegisterPageRoutes wrap their handlers in
// SidebarApp.Middleware, so Respond renders them in this shell while the
// rest of the demo keeps the topbar. Items, Browse and Settings carry Match
// so every page under their path marks them active; the query-string views
// are never active themselves (the shell compares paths). Flash is wired
// exactly as on App: the two shells share the one wf_flash cookie, so a
// flash set by an Items handler shows wherever the browser lands next.
// PaletteFor (demoPaletteFor) lists the pages and the store's items as of
// each render; the shell renders its trigger in the sidebar footer and the
// dialog once per page.
var SidebarApp = &appshell.App{
	Name:        "Webframe",
	HomeHref:    "/",
	Layout:      appshell.LayoutSidebar,
	ThemeToggle: true,
	Groups: []appshell.NavGroup{
		{Title: "Mine", Items: []appshell.NavItem{
			{Label: "Dashboard", Href: "/dashboard"},
			{Label: "Items", Href: "/items", Match: "/items"},
			{Label: "Browse", Href: "/browse", Match: "/browse"},
			{Label: "Settings", Href: "/settings", Match: "/settings"},
			{Label: "Logs", Href: "/logs"},
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
	Flash:      flash.Take,
	PaletteFor: demoPaletteFor,
	OnPanic:    errorpage.PanicHandler,
}

// Routes mounts every demo route on mux: the embedded static files, the
// showcase page, the Items app, the dashboard, settings,
// browse and logs pages, the auth and error pages and the htmx endpoints. There is no
// catch-all: a path no pattern claims, and a claimed path asked with the
// wrong method, are left to the mux, and errorpage.Handler (which Handler
// wraps around the mux) turns the mux's plain 404 and 405 into the error
// pages. cmd/demo and the tests call it so both serve the same table.
func Routes(mux *http.ServeMux) {
	// Static files, served from the assets package's embedded copy.
	assets.Mount(mux)

	// Pages.
	mux.HandleFunc("GET /{$}", HandleShowcase)
	mux.HandleFunc("GET /graph", HandleGraph)
	mux.HandleFunc("GET /api/graph", HandleGraphData)

	// The Items app (list, detail, create/edit form) on the sidebar shell:
	// GET /items, /items/new, /items/{id}, /items/{id}/edit,
	// /items/{id}/history; POST /items, /items/{id}, /items/validate,
	// /items/publish; DELETE /items/{id}; GET /sidebar → /items.
	RegisterItemRoutes(mux)

	// The dashboard, settings, browse and logs pages on the same shell: GET /dashboard; GET and
	// POST /settings, POST /settings/delete-account; GET /browse,
	// /browse/{id}; GET /logs and its poll, GET /logs/tail.
	RegisterPageRoutes(mux)

	// The pages outside the app's own content: sign in, signed out,
	// forbidden (no shell) and the three error pages (in the shell).
	mux.HandleFunc("GET /signin", HandleSignIn)
	mux.HandleFunc("POST /signin", HandleSignIn)
	mux.HandleFunc("GET /signed-out", HandleSignedOut)
	mux.HandleFunc("GET /forbidden", HandleForbidden)
	mux.HandleFunc("GET /errors/{code}", HandleErrorPage)

	// htmx demo endpoints.
	mux.HandleFunc("POST /api/click", HandleClick)
	mux.HandleFunc("GET /api/search", HandleSearch)
	mux.HandleFunc("POST /api/validate", HandleValidate)
	mux.HandleFunc("POST /api/picked", HandlePicked)
	mux.HandleFunc("POST /api/toast", HandleToast)
	mux.HandleFunc("POST /api/flash", HandleFlash)
	mux.HandleFunc("GET /api/slow", HandleSlow)
	mux.HandleFunc("GET /api/boom", HandleBoom)
	mux.HandleFunc("GET /api/panic", HandlePanic)
}

// Handler returns the demo server's whole handler: Routes on a fresh mux,
// errorpage.Handler around the mux (the mux's plain 404 and 405 become the
// styled pages), and App.Middleware around that so every page (the Items
// pages re-wrap themselves in SidebarApp.Middleware), both error pages and
// a recovered panic (App.OnPanic) render in a shell.
func Handler() http.Handler {
	mux := http.NewServeMux()
	Routes(mux)
	return App.Middleware(errorpage.Handler(mux))
}
