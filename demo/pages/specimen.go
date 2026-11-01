package pages

import _ "embed"

// proseSpecimen is the typography specimen the showcase renders through
// prose.HTML. It is content, not app markup, so it lives in its own file
// (as a help page or a CMS body would) rather than in a Go string literal,
// which the UI lint reads as HTML built in app code.
//
//go:embed prose_specimen.html
var proseSpecimen string
