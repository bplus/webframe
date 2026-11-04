package demo

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bplus/webframe/components/action"
)

// TestRenderFailed_PanicsOnInvalidAction: an invalid action declaration
// panics in the demo; any other render error does not.
func TestRenderFailed_PanicsOnInvalidAction(t *testing.T) {
	func() {
		defer func() {
			if recover() == nil {
				t.Errorf("renderFailed did not panic on action.ErrInvalid")
			}
		}()
		renderFailed(fmt.Errorf("wrapped: %w", action.ErrInvalid), "demo: render %s", "x")
	}()
	renderFailed(errors.New("disk full"), "demo: render %s", "y")
}
