package x

import "log"

// Mentions of markup, not markup: a log format, SQL, usage text and a hint.
// None of these builds HTML, so none is html-from-go-string.
const q = "select * from t where style= ?"

var (
	usage = "<input> is not allowed; <Label> neither"
	msg   = "<div> → layout.Stack"
)

func logClass(c string) {
	log.Printf("class=%s", c)
}
