// Package appshell is the page chrome and per-request wiring every webframe
// app starts from. An app configures one App (name, navigation, user menu,
// flash, palette), wraps its mux in App.Middleware, and renders each page
// with App.Render, which sends the whole document for a normal request and
// only the body or a named part for an htmx request.
//
// Needs JavaScript: the user menu (where Sign out usually is) does not open,
// nav badges and the live dot stay empty, and the theme toggle and Search
// button do nothing; navigation, the phone Menu and the More group work.
package appshell
