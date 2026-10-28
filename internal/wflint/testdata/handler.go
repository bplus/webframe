package x

import "github.com/a-h/templ"

// Handler builds HTML from a Go string, which the lint forbids.
func Handler() templ.Component {
	return templ.Raw("<b>")
}

// Props built in Go and then passed to a component are still styling.
var p = card.Props{
	Title: "t",
	Class: "x",
}

// Other structs may have a Class field.
var o = other.Options{Class: "x"}

var a = map[string]any{"style": "x"}

// Any *Props literal counts, not only one named Props.
var c = dropdown.ContentProps{Class: "x"}
