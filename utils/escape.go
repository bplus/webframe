package utils

import "strings"

// EscapeJS escapes a string for safe inclusion in a JavaScript string literal.
// Replaces \, ', ", newlines, and closing script tags to prevent XSS.
// The `</` → `<\/` rewrite is the piece that stops a value from terminating
// the surrounding <script> element. strings.NewReplacer makes a single pass
// and never re-scans replacement output, so the \ → \\ rule cannot cascade
// into the other rules' output. Scope limit: this escapes for
// a JS string-literal context only (it does not make arbitrary HTML-attribute
// or URL contexts safe), and it does not escape U+2028/U+2029, which are
// invalid inside pre-ES2019 string literals.
//
// @pure
// @complexity O(len(s))
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
