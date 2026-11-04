package dashboardgrid

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// clamp restricts v to [lo, hi].
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// colsKey is the context key under which Grid stores its column count for
// the Slots inside it.
type colsKey struct{}

// defaultCols is the column count of a Grid with Cols 0, and what a Slot
// outside any Grid assumes.
const defaultCols = 12

// withCols renders its children with the Grid's column count in the
// context, so a Slot can cap its span at the real width of the grid.
func withCols(cols int) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children := templ.GetChildren(ctx)
		return children.Render(context.WithValue(templ.ClearChildren(ctx), colsKey{}, cols), w)
	})
}

// gridCols is the column count of the nearest enclosing Grid, or
// defaultCols outside one.
func gridCols(ctx context.Context) int {
	if n, ok := ctx.Value(colsKey{}).(int); ok && n > 0 {
		return n
	}
	return defaultCols
}
