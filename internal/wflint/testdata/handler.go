package x

import "github.com/a-h/templ"

// Handler builds HTML from a Go string, which the lint forbids.
func Handler() templ.Component {
	return templ.Raw("<b>")
}
