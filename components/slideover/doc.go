// Package slideover renders a panel that slides in from the left, the right
// or the bottom of the screen over a dimmed page. Use it for a side task,
// such as editing one row or showing filters on a phone, that should not
// leave the page.
//
// A Panel holds a Body, which scrolls, and a Footer under it. A Pinned part
// before the Body is pinned (it stays in view) while the Body scrolls; use
// it for filters and a count above a long list.
//
// Needs JavaScript: the trigger does not open the panel, so its content
// cannot be reached.
package slideover
