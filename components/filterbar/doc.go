// Package filterbar renders the row of search box and filter controls above a
// list, and keeps the URL and the list state in step when a filter changes.
// listpage renders it from its Filters; call it directly only for a list
// built without listpage.
//
// Needs JavaScript: the bar is not a form, so the search box, selects,
// toggle pills and Clear filters change nothing.
package filterbar
