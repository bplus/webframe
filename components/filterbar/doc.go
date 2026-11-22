// Package filterbar renders the row of search box and filter controls above a
// list, and keeps the URL and the list state in step when a filter changes.
// listpage renders it from its Filters; call it directly only for a list
// built without listpage.
//
// A control is a FilterConfig of one FilterType: TypeSearch (a debounced
// text box), TypeSelect, TypeToggle (pills, any number selected) or
// TypeChoice (a row of buttons on one line, one selected at most, with an
// optional first "All" button from Placeholder). FilterOption.Count puts a
// number after an option's label in the last three; the handler supplies
// it. WithSelected marks the options and fills the search text from the
// query.
//
// Needs JavaScript: the bar is not a form, so the search box, selects,
// toggle pills, choice buttons and Clear filters change nothing.
package filterbar
