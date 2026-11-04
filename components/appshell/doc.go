// Package appshell is the page chrome and per-request wiring every webframe
// app starts from. An app configures one App (name, navigation, user menu,
// flash, palette), wraps its mux in App.Middleware, and renders each page
// with App.Render, which sends the whole document for a normal request and
// only the body or a named part for an htmx request.
package appshell
