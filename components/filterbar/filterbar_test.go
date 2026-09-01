package filterbar_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/filterbar"
)

// TestFilterBar_NoID_OmitsIDAttribute locks the contract that a bar rendered
// without an explicit ID emits no id attribute on the root <div>. The removed
// utils.RandomID fallback was computed but never rendered, so the emitted
// markup must be unchanged by that cleanup: no explicit ID, no id attribute.
func TestFilterBar_NoID_OmitsIDAttribute(t *testing.T) {
	var buf bytes.Buffer
	comp := filterbar.FilterBar()
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `data-tui-filterbar`) {
		t.Fatalf("expected filterbar marker, got: %s", html)
	}
	if strings.Contains(html, `id="`) {
		t.Fatalf("expected no id attribute when Props.ID is empty, got: %s", html)
	}
}

// TestFilterBar_ExplicitID verifies that when an ID is supplied the root <div>
// carries it, and that controls render keyed by their FilterConfig.ID.
func TestFilterBar_ExplicitID(t *testing.T) {
	var buf bytes.Buffer
	comp := filterbar.FilterBar(filterbar.Props{
		ID:       "region-bar",
		HxGet:    "/filters",
		HxTarget: "#results",
		Filters: []filterbar.FilterConfig{{
			ID:    "status",
			Label: "Status",
			Type:  filterbar.TypeSelect,
			Options: []filterbar.FilterOption{
				{Label: "All", Value: ""},
				{Label: "Errors", Value: "error"},
			},
		}},
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `id="region-bar"`) {
		t.Fatalf("expected root id when Props.ID is set, got: %s", html)
	}
	if !strings.Contains(html, `name="status"`) {
		t.Fatalf("expected control keyed by FilterConfig.ID, got: %s", html)
	}
	if !strings.Contains(html, `data-tui-filterbar-hx-get="/filters"`) {
		t.Fatalf("expected hx-get attribute, got: %s", html)
	}
}
