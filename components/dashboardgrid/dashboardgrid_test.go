package dashboardgrid_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/dashboardgrid"
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
		Gap:        24,
		PageMargin: 16,
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

func TestGridSlot_Composition(t *testing.T) {
	childContent := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<p>Child content here</p>`)
		return err
	})

	slotContent := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<span>Slot child</span>`)
		return err
	})

	// Render Slot with children
	var slotBuf bytes.Buffer
	slotComp := dashboardgrid.Slot(dashboardgrid.SlotProps{ColSpan: 6})
	if err := slotComp.Render(templ.WithChildren(context.Background(), slotContent), &slotBuf); err != nil {
		t.Fatalf("render slot: %v", err)
	}
	slotHTML := slotBuf.String()
	if !strings.Contains(slotHTML, "Slot child") {
		t.Errorf("expected slot children to appear, got: %s", slotHTML)
	}

	// Render Grid with children
	var gridBuf bytes.Buffer
	gridComp := dashboardgrid.Grid(dashboardgrid.Props{Cols: 12})
	if err := gridComp.Render(templ.WithChildren(context.Background(), childContent), &gridBuf); err != nil {
		t.Fatalf("render grid: %v", err)
	}
	gridHTML := gridBuf.String()
	if !strings.Contains(gridHTML, "Child content here") {
		t.Errorf("expected grid children to appear, got: %s", gridHTML)
	}
}

func TestGrid_ColsClamped(t *testing.T) {
	tests := []struct {
		name     string
		cols     int
		expected string
	}{
		{"negative clamped to 1", -5, "repeat(1,1fr)"},
		{"zero gets default 12", 0, "repeat(12,1fr)"},
		{"over 24 clamped to 24", 30, "repeat(24,1fr)"},
		{"valid passes through", 6, "repeat(6,1fr)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			comp := dashboardgrid.Grid(dashboardgrid.Props{Cols: tt.cols})
			if err := comp.Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			if !strings.Contains(buf.String(), tt.expected) {
				t.Errorf("expected %s, got: %s", tt.expected, buf.String())
			}
		})
	}
}

func TestSlot_SpanClamped(t *testing.T) {
	tests := []struct {
		name        string
		colSpan     int
		rowSpan     int
		expectedCol string
		expectedRow string
	}{
		{"negative clamped to 1", -3, -2, "grid-column:span 1", "grid-row:span 1"},
		{"over 12 clamped", 20, 15, "grid-column:span 12", "grid-row:span 12"},
		{"valid passes through", 4, 2, "grid-column:span 4", "grid-row:span 2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			comp := dashboardgrid.Slot(dashboardgrid.SlotProps{
				ColSpan: tt.colSpan,
				RowSpan: tt.rowSpan,
			})
			if err := comp.Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := buf.String()
			if !strings.Contains(html, tt.expectedCol) {
				t.Errorf("expected %s, got: %s", tt.expectedCol, html)
			}
			if !strings.Contains(html, tt.expectedRow) {
				t.Errorf("expected %s, got: %s", tt.expectedRow, html)
			}
		})
	}
}
