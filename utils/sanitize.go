package utils

import "regexp"

// validColorRe matches safe CSS color values: hex (#fff, #aabbcc),
// named colors (red, currentColor), rgb/hsl functions, percentages.
// Rejects anything that could be used for CSS injection (semicolons,
// quotes, url(), etc.).
var validColorRe = regexp.MustCompile(`^[a-zA-Z0-9#(),. %_-]+$`)

// SanitizeColor returns the color string if it contains only safe
// CSS color characters, otherwise returns a neutral gray fallback.
func SanitizeColor(c string) string {
	if c != "" && validColorRe.MatchString(c) {
		return c
	}
	return "#888"
}
