package x

import (
	"fmt"

	content "github.com/bplus/webframe/components/prose"
)

// Content handed to prose is not app markup: the argument of a
// content.HTML conversion and the HTML value of a content.Props literal are
// exempt from html-from-go-string under whatever name the file imports the
// package. A literal anywhere else, and one under a name the file does not
// import prose as, is still reported.
var specimen = content.HTML(`<h2>Specimen</h2>
<table><tr><td>cell</td></tr></table>`)

var note = content.Props{
	ID:   "note",
	HTML: "<p class=\"lead\">A <strong>note</strong>.</p>",
}

func help(name string) content.Props {
	return content.Props{HTML: content.HTML("<div class=\"x\">" + fmt.Sprint(name) + "</div>"), Text: "<div class=\"text\">"}
}

var raw = "<div class=\"x\">"

var wrong = prose.HTML("<div class=\"y\">")
