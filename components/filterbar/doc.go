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
// A filter change swaps Props.HxTarget only; the bar is never rendered
// again, because its search box may have the focus and a half-typed value.
// A response that swaps the target can still change the bar, with two
// components rendered at its top level (hx-swap-oob, matched by id, so the
// bar needs Props.ID): HiddenOOB renders the bar's hidden inputs and
// state-carrying controls again (ids from HiddenID), so list state changed
// outside the bar is what the next filter change sends; ControlsOOB renders
// selects, toggles and choice groups again (ids from ControlID) with the
// options the response computed, so labels and counts can follow the
// search. Neither renders a search box.
//
// Needs JavaScript: the bar is not a form, so the search box, selects,
// toggle pills, choice buttons and Clear filters change nothing.
package filterbar
