package demo

import "net/http"

// The batch 4 pages: a dashboard, a settings screen and a two-pane browser,
// each a page type from the library declared as Props and answered through
// its Respond, on the Items app's sidebar shell. The handlers live in
// dashboard_handlers.go, settings_handlers.go and browse_handlers.go.

// RegisterPageRoutes mounts the batch 4 pages on mux, every handler wrapped
// in SidebarApp.Middleware exactly as RegisterItemRoutes wraps the Items
// app, so the pages render in the sidebar shell with its MINE group.
func RegisterPageRoutes(mux *http.ServeMux) {
	side := func(fn http.HandlerFunc) http.Handler { return SidebarApp.Middleware(fn) }
	mux.Handle("GET /dashboard", side(HandleDashboard))
	mux.Handle("GET /settings", side(HandleSettings))
	mux.Handle("POST /settings", side(HandleSettingsSave))
	mux.Handle("POST /settings/delete-account", side(HandleDeleteAccount))
	mux.Handle("GET /browse", side(HandleBrowse))
	mux.Handle("GET /browse/{id}", side(HandleBrowseItem))
}
