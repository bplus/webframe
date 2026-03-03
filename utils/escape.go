package utils

import "strings"

// EscapeJS escapes a string for safe inclusion in a JavaScript string literal.
// Replaces \, ', ", newlines, and closing script tags to prevent XSS.
func EscapeJS(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"</", `<\/`,
	)
	return replacer.Replace(s)
}
