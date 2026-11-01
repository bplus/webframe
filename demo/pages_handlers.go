package demo

import "net/http"

// The batch 4 pages: a dashboard, a settings screen and a two-pane browser,
// each a page type from the library declared as Props and answered through
// its Respond, on the Items app's sidebar shell, plus the batch 5 Logs page.
// The handlers live in dashboard_handlers.go, settings_handlers.go,
// browse_handlers.go and logs.go.

// RegisterPageRoutes mounts the batch 4 and 5 pages on mux, every page
// handler wrapped in SidebarApp.Middleware exactly as RegisterItemRoutes
// wraps the Items app, so the pages render in the sidebar shell with its
// MINE group. /logs/tail is a fragment (no shell) and is not wrapped.
func RegisterPageRoutes(mux *http.ServeMux) {
	side := func(fn http.HandlerFunc) http.Handler { return SidebarApp.Middleware(fn) }
	mux.Handle("GET /dashboard", side(HandleDashboard))
	mux.Handle("GET /settings", side(HandleSettings))
	mux.Handle("POST /settings", side(HandleSettingsSave))
	mux.Handle("POST /settings/delete-account", side(HandleDeleteAccount))
	mux.Handle("GET /browse", side(HandleBrowse))
	mux.Handle("GET /browse/{id}", side(HandleBrowseItem))
	mux.Handle("GET /logs", side(HandleLogs))
	mux.HandleFunc("GET /logs/tail", HandleLogsTail)
}
