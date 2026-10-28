package hovercard

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/popover"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte("<span>child</span>"))
		return err
	})
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), child)
	if err := Hover(p).Render(ctx, &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestHoverDefaultsToBottomStart(t *testing.T) {
	out := render(t, Props{ID: "x", Title: "T"})
	if !strings.Contains(out, `data-tui-popover-placement="bottom-start"`) {
		t.Errorf("want bottom-start placement, got:\n%s", out)
	}
}

func TestHoverHonorsPlacement(t *testing.T) {
	out := render(t, Props{ID: "x", Title: "T", Placement: popover.PlacementTopEnd})
	if !strings.Contains(out, `data-tui-popover-placement="top-end"`) {
		t.Errorf("want top-end placement, got:\n%s", out)
	}
}

func TestHoverLinkIsSanitized(t *testing.T) {
	out := render(t, Props{ID: "x", Link: "javascript:alert(1)"})
	if !strings.Contains(out, `href="about:invalid`) || strings.Contains(out, "javascript:") {
		t.Errorf("unsafe link not sanitized:\n%s", out)
	}
	out = render(t, Props{ID: "x", Link: "/items/1"})
	if !strings.Contains(out, `href="/items/1"`) {
		t.Errorf("safe link altered:\n%s", out)
	}
}

// TestHover_CardIsScrollableCapped pins batch 2 input #13: the card body is
// capped to the height popover.min.js reports as available (--wf-pop-max,
// falling back to 70vh before the script has positioned it) and scrolls
// inside, so a tall card beside a mid-screen trigger at 390x844 never
// extends past the viewport.
func TestHover_CardIsScrollableCapped(t *testing.T) {
	out := render(t, Props{ID: "x", Title: "T"})
	if !strings.Contains(out, "overflow-y:auto") {
		t.Errorf("want inner scroll (overflow-y:auto) on the card shell, got:\n%s", out)
	}
	if !strings.Contains(out, "max-height:") || !strings.Contains(out, "var(--wf-pop-max, 70vh)") {
		t.Errorf("want max-height bound to --wf-pop-max with a 70vh fallback, got:\n%s", out)
	}
}
