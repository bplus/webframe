package utils

import "regexp"

// validColorRe matches safe CSS color values: hex (#fff, #aabbcc),
// named colors (red, currentColor), rgb/hsl functions, percentages.
// Rejects anything that could be used for CSS injection (semicolons,
// quotes, url(), etc.).
//
// This is a character allow-list, not a color-syntax parser: strings that
// aren't valid colors (e.g. "red,,(") still pass. That's acceptable because
// escaping the value context requires a character it blocks — ';', ':', '/',
// quotes, '<', '{', '}' — so url(...) payloads and property injection fail on
// ':' or '/' even though the letters "url" themselves are allowed.
var validColorRe = regexp.MustCompile(`^[a-zA-Z0-9#(),. %_-]+$`)

// SanitizeColor returns the color string if it contains only safe
// CSS color characters, otherwise returns a neutral gray fallback.
// Empty input also yields the "#888" fallback, so chart components can pass
// user-provided colors straight through without their own empty check. Use
// this for any color that reaches an SVG attribute or inline style; the
// fallback deliberately renders visibly (gray) rather than failing.
func SanitizeColor(c string) string {
	if c != "" && validColorRe.MatchString(c) {
		return c
	}
	return "#888"
}
