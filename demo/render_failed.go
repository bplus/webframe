package demo

import (
	"errors"
	"fmt"
	"log"

	"github.com/bplus/webframe/components/action"
)

// renderFailed reports an error from a page type's Respond. An invalid
// action declaration (action.ErrInvalid) is a bug in the demo's own
// code, so it panics: appshell.Middleware logs the stack and answers with
// the 500 page, and every demo test that loads the page fails. Any other
// error is logged, as a real app would.
func renderFailed(err error, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if errors.Is(err, action.ErrInvalid) {
		panic(fmt.Sprintf("%s: %v", msg, err))
	}
	log.Printf("%s: %v", msg, err)
}
