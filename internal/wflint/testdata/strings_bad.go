package x

// Markup, not mentions: a quoted class, a tag directly inside another tag,
// a quoted style, a class whose value is concatenated from outside the
// literal. Each literal is one html-from-go-string.
func markup(q string) []string {
	a := "<div class=\"x\">"
	b := `<table><tr><td>`
	c := "<p style='x'>"
	d := "<div class=" + q + ">"
	return []string{a, b, c, d}
}
