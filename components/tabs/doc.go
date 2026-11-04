// Package tabs renders a row of tab buttons over panels, one visible at a
// time. By default the panels are all as tall as the tallest, so switching
// tabs does not move the page; detailpage builds its tabs on it.
//
// Needs JavaScript: only the first panel shows; the other tab buttons do
// nothing, and a lazy panel (hx-get) never loads.
package tabs
