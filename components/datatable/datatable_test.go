package datatable

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/emptystate"
	"github.com/bplus/webframe/listquery"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := DataTable(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func wantAll(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func wantNone(t *testing.T, out string, bads ...string) {
	t.Helper()
	for _, b := range bads {
		if strings.Contains(out, b) {
			t.Errorf("unexpected %q in:\n%s", b, out)
		}
	}
}

var itemColumns = []Column{
	{Key: "name", Label: "NAMECOL", Sortable: true},
	{Key: "type", Label: "TYPECOL", Sortable: true, Hidden: true},
	{Key: "status", Label: "STATUSCOL", Shrink: true},
	{Key: "updated", Label: "UPDATEDCOL", Mono: true, Align: AlignRight},
}

func itemRows() []Row {
	return []Row{
		{ID: "1", Href: "/items/1", Cells: []Cell{{Text: "Alpha"}, {Text: "weapon"}, {Text: "published"}, {Text: "2026-10-01"}}},
		{ID: "2", Cells: []Cell{{Text: "Beta"}, {Text: "armor"}, {Text: "draft", Muted: true}, {Text: "2026-10-02", Mono: true}}},
	}
}

func TestDataTable_HeaderAndCells(t *testing.T) {
	out := render(t, Props{ID: "tbl", Class: "mt-2", Attributes: templ.Attributes{"data-x": "1"},
		Columns: itemColumns, Rows: itemRows(), Base: "/items", Caption: "All items"})
	wantAll(t, out,
		`id="tbl"`, `mt-2`, `data-x="1"`,
		`<table`, `<thead`, `<tbody`, `<th`, `<td`,
		"NAMECOL", "TYPECOL", "STATUSCOL", "UPDATEDCOL",
		"Alpha", "weapon", "published", "2026-10-01", "Beta", "armor", "draft", "2026-10-02",
		`<caption class="sr-only">All items</caption>`,
		`id="tbl-row-1"`, `id="tbl-row-2"`,
		"text-right", "font-mono", "text-muted-foreground", "whitespace-nowrap",
	)
	if strings.Count(out, "<tr") != 3 {
		t.Errorf("want 3 rows (header + 2), got %d", strings.Count(out, "<tr"))
	}
	// Non-sortable columns carry no aria-sort and no link.
	i := strings.Index(out, "STATUSCOL")
	th := out[strings.LastIndex(out[:i], "<th"):i]
	wantNone(t, th, "aria-sort", "<a ")
}

func TestDataTable_SortLinks(t *testing.T) {
	spec := listquery.Spec{Sort: "name", Dir: listquery.Asc, Page: 3}
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Spec: spec, Base: "/items"})
	wantAll(t, out,
		`aria-sort="ascending"`,
		`href="/items?dir=desc&amp;sort=name"`,
		`href="/items?dir=asc&amp;sort=type"`,
		`aria-sort="none"`,
	)
	if strings.Count(out, "aria-sort=") != 2 {
		t.Errorf("want aria-sort on exactly the two sortable columns, got %d", strings.Count(out, "aria-sort="))
	}
	// Sorted column shows a single-direction chevron; the other sortable
	// column shows the faint two-way chevron so the header width is stable.
	wantAll(t, out, `data-sort-icon="ascending"`, `data-sort-icon="none"`)
	wantNone(t, out, `data-sort-icon="descending"`, "hx-get", "hx-push-url")
	if got := strings.Count(out, "data-sort-icon="); got != 2 {
		t.Errorf("want one chevron per sortable column (2), got %d", got)
	}

	// Descending flips the icon and the aria value.
	spec.Dir = listquery.Desc
	out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Spec: spec, Base: "/items"})
	wantAll(t, out, `aria-sort="descending"`, `data-sort-icon="descending"`, `href="/items?dir=asc&amp;sort=name"`)
	wantNone(t, out, `data-sort-icon="ascending"`)

	// No Base and no request path: links are relative to the current URL.
	out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Spec: spec})
	wantAll(t, out, `href="?dir=asc&amp;sort=name"`)
}

func TestDataTable_HiddenColumn(t *testing.T) {
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items"})
	// One hidden column: its th plus one td per row carry md:table-cell.
	if got := strings.Count(out, "md:table-cell"); got != len(itemRows())+1 {
		t.Errorf("md:table-cell count = %d, want %d", got, len(itemRows())+1)
	}
	// The th and the td are hidden below md.
	i := strings.Index(out, "TYPECOL")
	th := out[strings.LastIndex(out[:i], "<th"):i]
	wantAll(t, th, "hidden", "md:table-cell")
	j := strings.Index(out, "weapon")
	td := out[strings.LastIndex(out[:j], "<td"):j]
	wantAll(t, td, "hidden", "md:table-cell")
}

func TestDataTable_RowHref(t *testing.T) {
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items"})
	wantAll(t, out, `<a href="/items/1"`, "cursor-pointer")
	// The linked text is the first cell's text, and only the first cell.
	i := strings.Index(out, `<a href="/items/1"`)
	a := out[i : strings.Index(out[i:], "</a>")+i]
	wantAll(t, a, "Alpha")
	wantNone(t, a, "weapon")
	// A row without Href has no link and no pointer.
	j := strings.Index(out, `id="tbl-row-2"`)
	if j >= 0 {
		t.Fatal("unexpected table id prefix without Props.ID")
	}
	j = strings.Index(out, `id="row-2"`)
	if j < 0 {
		t.Fatalf("missing row-2 in %s", out)
	}
	row2 := out[j : strings.Index(out[j:], "</tr>")+j]
	wantNone(t, row2, "<a ", "cursor-pointer")
	wantAll(t, row2, "Beta")
}

func TestDataTable_RowActionsTwoRenderings(t *testing.T) {
	rows := itemRows()
	rows[0].Actions = []action.Action{
		{Label: "EDITACT", Href: "/items/1/edit"},
		{Label: "DELACT", HxDelete: "/items/1", HxTarget: "#list-results", Confirm: &action.Confirm{Message: "Sure?"}},
	}
	out := render(t, Props{Columns: itemColumns, Rows: rows, Base: "/items"})
	wantAll(t, out, `aria-label="Row actions"`, "Actions")
	if got := strings.Count(out, "EDITACT"); got != 2 {
		t.Errorf("EDITACT count = %d, want 2 (inline + menu)", got)
	}
	if got := strings.Count(out, "DELACT"); got != 2 {
		t.Errorf("DELACT count = %d, want 2 (inline + menu)", got)
	}
	// Inline group is desktop-only; the menu is phone-only.
	wantAll(t, out, "md:flex", "md:hidden")
	// The menu item carries the action's hx attributes and is not a nested button.
	menu := out[strings.Index(out, `aria-label="Row actions"`):]
	wantAll(t, menu, `role="menuitem"`, `hx-delete="/items/1"`, `hx-target="#list-results"`, `hx-confirm="Sure?"`, `href="/items/1/edit"`)
	// Row 2 has no actions but still gets the actions cell so columns line up.
	j := strings.Index(out, `id="row-2"`)
	row2 := out[j : strings.Index(out[j:], "</tr>")+j]
	if strings.Count(row2, "<td") != len(itemColumns)+1 {
		t.Errorf("row-2 td count = %d, want %d", strings.Count(row2, "<td"), len(itemColumns)+1)
	}
	// Header has a column for actions too.
	head := out[:strings.Index(out, "</thead>")]
	if strings.Count(head, "<th ") != len(itemColumns)+1 {
		t.Errorf("th count = %d, want %d", strings.Count(head, "<th "), len(itemColumns)+1)
	}
}

func TestDataTable_Selection(t *testing.T) {
	out := render(t, Props{ID: "tbl", Columns: itemColumns, Rows: itemRows(), Base: "/items",
		Selection: true, SelectionForm: "bulk"})
	wantAll(t, out,
		`data-tui-checkbox-group="tbl-selection"`,
		`data-tui-checkbox-parent="true"`,
		`name="ids"`, `value="1"`, `value="2"`,
		`form="bulk"`,
		`aria-label="Select all rows"`, `aria-label="Select row"`,
	)
	// The selection column renders at every width: only the fixture's
	// hidden type column collapses on a phone.
	if got := strings.Count(out, `md:table-cell`); got != 3 {
		t.Errorf("md:table-cell count = %d, want 3 (type column: header + 2 rows)", got)
	}
	if got := strings.Count(out, `type="checkbox"`); got != 3 {
		t.Errorf("checkbox count = %d, want 3 (header + 2 rows)", got)
	}
	// The header box has no name: it must never be submitted.
	head := out[:strings.Index(out, "</thead>")]
	wantNone(t, head, `name="ids"`)
	// Custom name.
	out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Selection: true, SelectionName: "item"})
	wantAll(t, out, `name="item"`, `data-tui-checkbox-group="item"`)
	wantNone(t, out, `name="ids"`)
}

func TestDataTable_EmptyKeepsHeader(t *testing.T) {
	out := render(t, Props{Columns: itemColumns, Base: "/items"})
	wantAll(t, out, `<thead`, "NAMECOL", `colspan="4"`, "Nothing here yet")
	if strings.Count(out, "<tr") != 2 {
		t.Errorf("want header row + one empty row, got %d", strings.Count(out, "<tr"))
	}
	// Selection adds a column to the span and a custom empty state is used.
	out = render(t, Props{Columns: itemColumns, Base: "/items", Selection: true,
		Empty: emptystate.Props{Title: "No items match", Description: "Clear the filters"}})
	wantAll(t, out, `colspan="5"`, "No items match", "Clear the filters")
	wantNone(t, out, "Nothing here yet")
	// Never a checkbox in an empty table.
	wantNone(t, out, `type="checkbox"`)
}

func TestDataTable_Badge(t *testing.T) {
	rows := itemRows()
	rows[0].Cells[2] = Cell{Text: "published", Badge: &badge.Props{Variant: badge.VariantSuccess}}
	rows[1].Cells[3] = Cell{Component: templ.Raw("<b>CUSTOMCOMP</b>")}
	out := render(t, Props{Columns: itemColumns, Rows: rows, Base: "/items"})
	wantAll(t, out, "text-success", "published", "<b>CUSTOMCOMP</b>")
	// The badge variant classes wrap the text.
	i := strings.Index(out, "text-success")
	after := out[i : strings.Index(out[i:], "</span>")+i]
	wantAll(t, after, "published")
}

func TestDataTable_HxTarget(t *testing.T) {
	spec := listquery.Spec{Sort: "name", Dir: listquery.Asc}
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Spec: spec, Base: "/items", HxTarget: "#list-results"})
	wantAll(t, out,
		`href="/items?dir=desc&amp;sort=name"`,
		`hx-get="/items?dir=desc&amp;sort=name"`,
		`hx-target="#list-results"`,
		`hx-push-url="true"`,
	)
	if got := strings.Count(out, "hx-get="); got != 2 {
		t.Errorf("hx-get count = %d, want one per sortable column (2)", got)
	}
}

func TestDataTable_DenseAndSticky(t *testing.T) {
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", Dense: true, StickyHeader: true})
	// Page-level sticking offsets by the shell's sticky-top variable (the
	// topbar height) so the header is not hidden under the bar.
	wantAll(t, out, "py-1.5", "sticky", "top-[var(--wf-sticky-top,0px)]", "z-10", "bg-card")
	wantNone(t, out, "top-0")
	// The sticky classes sit on the thead, not the wrapper.
	head := out[strings.Index(out, "<thead"):strings.Index(out, "</thead>")]
	wantAll(t, head[:strings.Index(head, ">")], "sticky", "top-[var(--wf-sticky-top,0px)]", "z-10", "bg-card")
	out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items"})
	wantNone(t, out, "py-1.5", "sticky", "md:overflow-visible", "max-h-")
}

// wrapper returns the opening tag of table.Table's wrapper div.
func wrapper(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, "<div")
	if i < 0 {
		t.Fatalf("no wrapper div in:\n%s", out)
	}
	return out[i : strings.Index(out[i:], ">")+i]
}

func TestDataTable_StickyHeaderWithoutMaxHeightSticksToPage(t *testing.T) {
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", StickyHeader: true})
	w := wrapper(t, out)
	// md+: the wrapper stops clipping so the thead's scrolling ancestor is
	// the page. Below md the wrapper keeps its own scroll for overflow.
	wantAll(t, w, "md:overflow-visible", "overflow-auto", "relative")
	wantNone(t, w, "max-h-")
}

func TestDataTable_MaxHeight(t *testing.T) {
	cases := []struct {
		h    Height
		want string
	}{
		{HeightSm, "max-h-[240px]"},
		{HeightMd, "max-h-[400px]"},
		{HeightLg, "max-h-[600px]"},
	}
	for _, c := range cases {
		// With StickyHeader: the capped wrapper is the scroll box at every
		// width, so it must keep overflow-auto and never go page-level.
		out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", StickyHeader: true, MaxHeight: c.h})
		w := wrapper(t, out)
		wantAll(t, w, c.want, "overflow-auto")
		wantNone(t, w, "md:overflow-visible")
		// Inside the capped box the header sticks to the box's own top.
		wantAll(t, out, "sticky", "top-0", "bg-card")
		wantNone(t, out, "wf-sticky-top")
		// Without StickyHeader the cap alone is a plain scroll box.
		out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", MaxHeight: c.h})
		w = wrapper(t, out)
		wantAll(t, w, c.want, "overflow-auto")
		wantNone(t, w, "md:overflow-visible", "sticky")
	}
	// Exactly one max-h class, and only one wrapper-level cap.
	out := render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", MaxHeight: HeightMd})
	if got := strings.Count(out, "max-h-"); got != 1 {
		t.Errorf("max-h- count = %d, want 1", got)
	}
	// HeightNone and an unknown value set no cap.
	for _, h := range []Height{HeightNone, Height("huge")} {
		out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", MaxHeight: h})
		wantNone(t, wrapper(t, out), "max-h-")
	}
	// The caller's Class is still merged onto the wrapper alongside the cap.
	out = render(t, Props{Columns: itemColumns, Rows: itemRows(), Base: "/items", MaxHeight: HeightSm, StickyHeader: true, Class: "mt-2"})
	wantAll(t, wrapper(t, out), "mt-2", "max-h-[240px]")
}

func TestDataTable_ShortRow(t *testing.T) {
	// Fewer cells than columns still renders one td per column.
	out := render(t, Props{Columns: itemColumns, Rows: []Row{{ID: "x", Cells: []Cell{{Text: "Only"}}}}, Base: "/items"})
	if got := strings.Count(out, "<td"); got != len(itemColumns) {
		t.Errorf("td count = %d, want %d", got, len(itemColumns))
	}
}

// BadgeCell picks the variant from kind; text is the badge's text.
func TestDataTable_BadgeCell(t *testing.T) {
	c := BadgeCell("Published", "published")
	if c.Badge == nil || c.Badge.Variant != badge.VariantSuccess || c.Text != "Published" {
		t.Fatalf("BadgeCell = %+v", c)
	}
	if BadgeCell("x", "whatever").Badge.Variant != badge.VariantOutline {
		t.Errorf("unknown kind should fall back to outline")
	}
	rows := itemRows()
	rows[0].Cells[2] = BadgeCell(badge.StatusLabel("in-progress"), "in-progress")
	out := render(t, Props{Columns: itemColumns, Rows: rows, Base: "/items"})
	wantAll(t, out, "text-info", "In progress")
	i := strings.Index(out, "text-info")
	after := out[i : strings.Index(out[i:], "</span>")+i]
	wantAll(t, after, "In progress")
}

func TestDataTable_RowActionsMenuDestructiveConfirm(t *testing.T) {
	rows := itemRows()
	rows[0].Actions = []action.Action{
		{Label: "DELACT", HxDelete: "/items/1", HxTarget: "#list-results", Variant: action.VariantDestructive,
			Confirm: &action.Confirm{Title: "Delete Alpha", Message: "Sure?", ConfirmLabel: "Delete"}},
	}
	out := render(t, Props{Columns: itemColumns, Rows: rows, Base: "/items"})
	menu := out[strings.Index(out, `aria-label="Row actions"`):]
	wantAll(t, menu, `role="menuitem"`, `hx-delete="/items/1"`, `hx-confirm="Sure?"`, `data-wf-confirm-title="Delete Alpha"`, `data-wf-confirm-label="Delete"`, `data-wf-confirm-destructive="true"`)
}

// tdClasses returns the class set of the <td> whose content starts with text.
func tdClasses(t *testing.T, out, text string) map[string]bool {
	t.Helper()
	j := strings.Index(out, ">"+text+"<")
	if j < 0 {
		t.Fatalf("no cell %q in:\n%s", text, out)
	}
	td := out[strings.LastIndex(out[:j], "<td"):j]
	m := regexp.MustCompile(`class="([^"]*)"`).FindStringSubmatch(td)
	if m == nil {
		t.Fatalf("td for %q has no class: %s", text, td)
	}
	set := map[string]bool{}
	for _, f := range strings.Fields(m[1]) {
		set[f] = true
	}
	return set
}

// TestDataTable_CellAlignOverridesColumn checks a cell's Align wins over its
// column's, in both directions, and a zero Align inherits the column's.
func TestDataTable_CellAlignOverridesColumn(t *testing.T) {
	cols := []Column{{Key: "a", Label: "A"}, {Key: "n", Label: "N", Align: AlignRight}}
	out := render(t, Props{Columns: cols, Rows: []Row{
		{ID: "1", Cells: []Cell{{Text: "inheritLeft"}, {Text: "inheritRight"}}},
		{ID: "2", Cells: []Cell{{Text: "toCenter", Align: AlignCenter}, {Text: "toLeft", Align: AlignLeft}}},
	}})
	cases := []struct {
		text, want string
		not        []string
	}{
		{"inheritLeft", "", []string{"text-right", "text-center", "text-left"}},
		{"inheritRight", "text-right", []string{"text-center", "text-left"}},
		{"toCenter", "text-center", []string{"text-right", "text-left"}},
		{"toLeft", "text-left", []string{"text-right", "text-center"}},
	}
	for _, c := range cases {
		got := tdClasses(t, out, c.text)
		if c.want != "" && !got[c.want] {
			t.Errorf("%s: want %s, got %v", c.text, c.want, got)
		}
		for _, n := range c.not {
			if got[n] {
				t.Errorf("%s: unexpected %s in %v", c.text, n, got)
			}
		}
	}
}
