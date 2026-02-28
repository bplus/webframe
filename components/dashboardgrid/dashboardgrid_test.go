package dashboardgrid_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"webframe/components/dashboardgrid"
)

func TestGrid_DefaultProps(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardgrid.Grid()
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "repeat(12,1fr)") {
		t.Errorf("expected 12-column grid, got: %s", html)
	}
	if !strings.Contains(html, "gap:16px") {
		t.Errorf("expected 16px gap, got: %s", html)
	}
	if !strings.Contains(html, "padding:32px") {
		t.Errorf("expected 32px padding, got: %s", html)
	}
}

func TestGrid_CustomProps(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardgrid.Grid(dashboardgrid.Props{
		Cols:       6,
		Gap:        "24px",
		PageMargin: "16px",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "repeat(6,1fr)") {
		t.Errorf("expected 6-column grid, got: %s", html)
	}
	if !strings.Contains(html, "gap:24px") {
		t.Errorf("expected 24px gap, got: %s", html)
	}
}

func TestSlot_DefaultSpan(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardgrid.Slot()
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "span 12") {
		t.Errorf("expected default col-span 12, got: %s", html)
	}
}

func TestSlot_CustomSpan(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardgrid.Slot(dashboardgrid.SlotProps{
		ColSpan: 4,
		RowSpan: 2,
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "grid-column:span 4") {
		t.Errorf("expected col-span 4, got: %s", html)
	}
	if !strings.Contains(html, "grid-row:span 2") {
		t.Errorf("expected row-span 2, got: %s", html)
	}
}

func TestSlot_WithID(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardgrid.Slot(dashboardgrid.SlotProps{
		ID:      "kpi-row",
		ColSpan: 12,
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `id="kpi-row"`) {
		t.Errorf("expected id attribute, got: %s", html)
	}
}
