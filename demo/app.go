package demo

import (
	"log"
	"net/http"
	"strings"

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
// groups. MINE opens with the batch 4 pages (Dashboard, Browse, Settings)
// and the Items list; the remaining entries are views of the one list.
// RegisterItemRoutes and RegisterPageRoutes wrap their handlers in
// SidebarApp.Middleware, so Respond renders them in this shell while the
// rest of the demo keeps the topbar. Items, Browse and Settings carry Match
// so every page under their path marks them active; the query-string views
// are never active themselves (the shell compares paths). Flash is wired
// exactly as on App: the two shells share the one wf_flash cookie, so a
// flash set by an Items handler shows wherever the browser lands next.
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
// showcase and traffic pages, the Items app and the batch 4 pages, the auth
// and error pages and the htmx endpoints. There is no catch-all: a path no
// pattern claims, and a claimed path asked with the wrong method, are left
// to the mux, and styledErrors (which Handler wraps around the mux) turns
// the mux's plain 404 and 405 into the error pages. cmd/demo and the tests
// call it so both serve the same table.
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

	// The batch 4 pages on the same shell: GET /dashboard; GET and POST
	// /settings, POST /settings/delete-account; GET /browse, /browse/{id}.
	RegisterPageRoutes(mux)

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
	mux.HandleFunc("GET /api/boom", HandleBoom)
}

// Handler returns the demo server's whole handler: Routes on a fresh mux,
// styledErrors around the mux, and App.Middleware around that so every
// page (the Items pages re-wrap themselves in SidebarApp.Middleware) and
// both error pages render in a shell.
func Handler() http.Handler {
	mux := http.NewServeMux()
	Routes(mux)
	return App.Middleware(styledErrors(mux))
}

// probeMethods are the methods styledErrors tries when a path matched no
// pattern for the request's own method, to tell "no such page" from
// "wrong method" and to build the Allow header.
var probeMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}

// styledErrors is the wrapper the errorpage package documents: ServeMux
// answers a method mismatch itself with a plain-text 405 before any
// handler runs, and reports both "no route" and "wrong method" as an empty
// pattern from mux.Handler, so the wrapper probes the other methods to
// tell them apart. A path some method accepts gets the 405 page with the
// Allow header the mux would have sent; a path nothing accepts gets the
// 404 page. Any request the mux has a pattern for (including its own
// trailing-slash redirects) goes to the mux untouched. It replaces the
// catch-all "/" route the demo used to register: with a catch-all in the
// table every path has a pattern and the 405 case can never be seen.
func styledErrors(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, pattern := mux.Handler(r); pattern != "" {
			mux.ServeHTTP(w, r)
			return
		}
		var allowed []string
		probe := r.Clone(r.Context())
		for _, m := range probeMethods {
			probe.Method = m
			if _, pattern := mux.Handler(probe); pattern != "" {
				allowed = append(allowed, m)
			}
		}
		if len(allowed) == 0 {
			HandleNotFound(w, r)
			return
		}
		w.Header().Set("Allow", strings.Join(allowed, ", "))
		if err := errorpage.MethodNotAllowed(w, r); err != nil {
			log.Printf("demo: render method not allowed %s %s: %v", r.Method, r.URL.Path, err)
		}
	})
}
