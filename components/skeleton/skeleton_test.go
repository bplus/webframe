package skeleton

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

// bars counts the decorative pulsing bars in a render.
func bars(out string) int { return strings.Count(out, "animate-pulse") }

func TestLines_CountAndShape(t *testing.T) {
	out := render(t, Lines(4, Props{ID: "para", Class: "mt-2", Attributes: templ.Attributes{"data-x": "1"}}))
	if got := bars(out); got != 4 {
		t.Errorf("bars = %d, want 4", got)
	}
	mustContain(t, out, `id="para"`, `aria-busy="true"`, `data-x="1"`, "mt-2", "space-y-1", "h-4", "w-2/3", "bg-muted", `aria-hidden="true"`)
	if got := strings.Count(out, "w-2/3"); got != 1 {
		t.Errorf("last-line w-2/3 count = %d, want 1", got)
	}
	if got := strings.Count(out, `<span class="sr-only">Loading…</span>`); got != 1 {
		t.Errorf("sr-only Loading count = %d, want 1", got)
	}
	if got := strings.Count(out, `aria-hidden="true"`); got != 4 {
		t.Errorf("aria-hidden bars = %d, want 4", got)
	}
}

func TestLines_ZeroRendersOne(t *testing.T) {
	if got := bars(render(t, Lines(0))); got != 1 {
		t.Errorf("bars = %d, want 1", got)
	}
}

func TestCard_PanelShape(t *testing.T) {
	out := render(t, Card(Props{ID: "c"}))
	if got := bars(out); got != 4 {
		t.Errorf("bars = %d, want 4 (title + 3 lines)", got)
	}
	mustContain(t, out, `id="c"`, `aria-busy="true"`, "rounded-lg", "border-border", "bg-card", "px-4 py-3", "p-4", "space-y-2")
	if got := strings.Count(out, "sr-only"); got != 1 {
		t.Errorf("sr-only count = %d, want 1", got)
	}
}

func TestTable_RowsAndCols(t *testing.T) {
	out := render(t, Table(3, 4, Props{ID: "tbl", Attributes: templ.Attributes{"data-x": "1"}}))
	if got := strings.Count(out, "<th "); got != 4 {
		t.Errorf("header cells = %d, want 4", got)
	}
	if got := strings.Count(out, "<td"); got != 12 {
		t.Errorf("body cells = %d, want 12", got)
	}
	if got := strings.Count(out, "<tr"); got != 4 {
		t.Errorf("rows = %d, want 4 (1 header + 3 body)", got)
	}
	if got := bars(out); got != 16 {
		t.Errorf("bars = %d, want 16", got)
	}
	// datatable's non-dense paddings: table.Head h-10 px-3, table.Cell p-3, max-md:px-2.
	mustContain(t, out, `id="tbl"`, `aria-busy="true"`, `data-x="1"`, "h-10", "px-3", "p-3", "max-md:px-2", "h-5", "<caption class=\"sr-only\">Loading…</caption>", "hover:bg-transparent")
	if got := strings.Count(out, "Loading…"); got != 1 {
		t.Errorf("Loading… count = %d, want 1", got)
	}
	if got := strings.Count(out, `aria-hidden="true"`); got != 2 {
		t.Errorf("aria-hidden sections = %d, want 2 (thead, tbody)", got)
	}
}

func TestTable_ClampsToOne(t *testing.T) {
	out := render(t, Table(0, 0))
	if got := strings.Count(out, "<th "); got != 1 {
		t.Errorf("header cells = %d, want 1", got)
	}
	if got := strings.Count(out, "<td"); got != 1 {
		t.Errorf("body cells = %d, want 1", got)
	}
}

func TestTile_StatTileShape(t *testing.T) {
	out := render(t, Tile(Props{Class: "w-48"}))
	if got := bars(out); got != 4 {
		t.Errorf("bars = %d, want 4 (label, value, delta, sub)", got)
	}
	// stattile's surface: rounded-lg border p-4 min-h-24; value is 28px leading-none → h-7.
	mustContain(t, out, `aria-busy="true"`, "rounded-lg", "bg-card", "p-4", "min-h-24", "mb-1.5", "h-7", "mt-1.5", "w-48")
	if got := strings.Count(out, "sr-only"); got != 1 {
		t.Errorf("sr-only count = %d, want 1", got)
	}
}

func TestEveryRootIsBusy(t *testing.T) {
	for name, c := range map[string]templ.Component{
		"Lines": Lines(2), "Card": Card(), "Table": Table(1, 1), "Tile": Tile(),
	} {
		out := render(t, c)
		if strings.Count(out, `aria-busy="true"`) != 1 {
			t.Errorf("%s: aria-busy count != 1 in %s", name, out)
		}
		if !strings.Contains(out, "Loading…") {
			t.Errorf("%s: missing Loading… text", name)
		}
	}
}
