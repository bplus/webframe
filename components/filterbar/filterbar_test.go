package filterbar_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/filterbar"
)

// TestFilterBar_NoID_OmitsIDAttribute: a bar rendered without an explicit ID
// emits no id attribute on the root <div>.
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

// richProps uses every control type and a reset button, but no Hidden and no
// Value, to show those options leave other callers' markup untouched.
func richProps() filterbar.Props {
	return filterbar.Props{
		ID:        "bar",
		Class:     "mt-2",
		HxGet:     "/items",
		HxTarget:  "#results",
		ShowReset: true,
		Filters: []filterbar.FilterConfig{
			{ID: "type", Label: "Type", Placeholder: "Any type", Type: filterbar.TypeSelect,
				Options: []filterbar.FilterOption{{Label: "Weapon", Value: "weapon"}, {Label: "Armor", Value: "armor", Selected: true}}},
			{ID: "status", Label: "Status", Type: filterbar.TypeToggle,
				Options: []filterbar.FilterOption{{Label: "Published", Value: "published", Selected: true}, {Label: "Draft", Value: "draft"}}},
			{ID: "q", Label: "Search", Placeholder: "Search items", Type: filterbar.TypeSearch},
		},
	}
}

func render(t *testing.T, p filterbar.Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := filterbar.FilterBar(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// TestFilterBar_Hidden: Props.Hidden renders one hidden input per entry, in
// sorted key order, each carrying the control marker so filterbar.min.js
// collectParams sends it along with every filter change.
func TestFilterBar_Hidden(t *testing.T) {
	html := render(t, filterbar.Props{
		Hidden: map[string]string{"sort": "name", "dir": "asc"},
	})
	dir := `<input type="hidden" name="dir" value="asc" data-tui-filterbar-control>`
	sort := `<input type="hidden" name="sort" value="name" data-tui-filterbar-control>`
	di, si := strings.Index(html, dir), strings.Index(html, sort)
	if di < 0 {
		t.Fatalf("expected hidden dir control, got: %s", html)
	}
	if si < 0 {
		t.Fatalf("expected hidden sort control, got: %s", html)
	}
	if di > si {
		t.Fatalf("expected hidden inputs in sorted key order (dir before sort), got: %s", html)
	}
	if strings.Count(html, `type="hidden"`) != 2 {
		t.Fatalf("expected exactly two hidden inputs, got: %s", html)
	}
}

// TestFilterBar_HiddenIDs: with Props.ID set, each hidden input carries
// HiddenID(ID, name); a prefixed name's "." becomes "-" so the id works as
// an htmx oob selector. Without an ID no hidden input has an id.
func TestFilterBar_HiddenIDs(t *testing.T) {
	html := render(t, filterbar.Props{
		ID:     "bar",
		Hidden: map[string]string{"sort": "name", "items.dir": "asc"},
	})
	for _, want := range []string{
		`<input type="hidden" id="bar-hidden-items-dir" name="items.dir" value="asc" data-tui-filterbar-control>`,
		`<input type="hidden" id="bar-hidden-sort" name="sort" value="name" data-tui-filterbar-control>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in: %s", want, html)
		}
	}
	if strings.Contains(html, "hx-swap-oob") {
		t.Errorf("FilterBar must not render oob inputs: %s", html)
	}
	html = render(t, filterbar.Props{Hidden: map[string]string{"sort": "name"}})
	if strings.Contains(html, ` id=`) {
		t.Errorf("no ids without Props.ID, got: %s", html)
	}
	for _, c := range []struct{ bar, name, want string }{
		{"bar", "sort", "bar-hidden-sort"},
		{"list-filterbar", "items.sort", "list-filterbar-hidden-items-sort"},
		{"bar", "a b:c[d]", "bar-hidden-a-b-c-d-"},
		{"", "sort", ""},
	} {
		if got := filterbar.HiddenID(c.bar, c.name); got != c.want {
			t.Errorf("HiddenID(%q, %q) = %q, want %q", c.bar, c.name, got, c.want)
		}
	}
}

// TestHiddenOOB: the same inputs as the bar renders, each with
// hx-swap-oob="true"; nothing without a bar id or without entries.
func TestHiddenOOB(t *testing.T) {
	renderOOB := func(id string, hidden map[string]string) string {
		var buf bytes.Buffer
		if err := filterbar.HiddenOOB(id, hidden).Render(context.Background(), &buf); err != nil {
			t.Fatalf("render: %v", err)
		}
		return buf.String()
	}
	html := renderOOB("bar", map[string]string{"sort": "name", "dir": "asc", "per": ""})
	want := `<input type="hidden" id="bar-hidden-dir" name="dir" value="asc" data-tui-filterbar-control hx-swap-oob="true">` +
		`<input type="hidden" id="bar-hidden-per" name="per" value="" data-tui-filterbar-control hx-swap-oob="true">` +
		`<input type="hidden" id="bar-hidden-sort" name="sort" value="name" data-tui-filterbar-control hx-swap-oob="true">`
	if html != want {
		t.Errorf("HiddenOOB =\n%s\nwant\n%s", html, want)
	}
	if got := renderOOB("", map[string]string{"sort": "name"}); got != "" {
		t.Errorf("HiddenOOB without a bar id = %q, want nothing", got)
	}
	if got := renderOOB("bar", nil); got != "" {
		t.Errorf("HiddenOOB with no entries = %q, want nothing", got)
	}
}

// TestFilterBar_Drop: Props.Drop renders comma-joined as
// data-tui-filterbar-drop; callers that do not set it get no attribute.
func TestFilterBar_Drop(t *testing.T) {
	html := render(t, filterbar.Props{Drop: []string{"page", "items.page"}})
	if !strings.Contains(html, `data-tui-filterbar-drop="page,items.page"`) {
		t.Errorf("missing drop attribute in: %s", html)
	}
	if html := render(t, richProps()); strings.Contains(html, "data-tui-filterbar-drop") {
		t.Errorf("drop attribute rendered without Drop: %s", html)
	}
}

// TestFilterBar_NilHidden_NoHiddenInputs: callers that do not set Hidden get
// no hidden inputs at all.
func TestFilterBar_NilHidden_NoHiddenInputs(t *testing.T) {
	html := render(t, richProps())
	if strings.Contains(html, `type="hidden"`) {
		t.Fatalf("expected no hidden inputs when Hidden is nil, got: %s", html)
	}
}

// TestFilterBar_SearchValue: FilterConfig.Value renders as the search input's
// initial value, and an empty Value emits no value attribute.
func TestFilterBar_SearchValue(t *testing.T) {
	html := render(t, filterbar.Props{Filters: []filterbar.FilterConfig{
		{ID: "q", Type: filterbar.TypeSearch, Value: `swo"rd`},
	}})
	if !strings.Contains(html, `type="search" name="q" data-tui-filterbar-control value="swo&#34;rd"`) {
		t.Fatalf("expected escaped search value, got: %s", html)
	}
	html = render(t, filterbar.Props{Filters: []filterbar.FilterConfig{
		{ID: "q", Type: filterbar.TypeSearch},
	}})
	if strings.Contains(html, `value=`) {
		t.Fatalf("expected no value attribute for an empty Value, got: %s", html)
	}
}

// TestWithSelected: Selected and Value are marked from the query values; the
// input slice and its Options are never mutated.
func TestWithSelected(t *testing.T) {
	in := richProps().Filters
	// Start from a clean slate: nothing selected in the input.
	for i := range in {
		for j := range in[i].Options {
			in[i].Options[j].Selected = false
		}
	}
	out := filterbar.WithSelected(in, map[string][]string{
		"type":   {"weapon"},
		"status": {"published", "draft"},
		"q":      {"sword"},
	})

	if len(out) != len(in) {
		t.Fatalf("expected %d filters, got %d", len(in), len(out))
	}
	if !out[0].Options[0].Selected || out[0].Options[1].Selected {
		t.Fatalf("select: expected only weapon selected, got %+v", out[0].Options)
	}
	if !out[1].Options[0].Selected || !out[1].Options[1].Selected {
		t.Fatalf("toggle: expected published and draft selected, got %+v", out[1].Options)
	}
	if out[2].Value != "sword" {
		t.Fatalf("search: expected Value sword, got %q", out[2].Value)
	}

	// Original untouched.
	for i := range in {
		if in[i].Value != "" {
			t.Fatalf("input slice mutated: Value set on %q", in[i].ID)
		}
		for _, o := range in[i].Options {
			if o.Selected {
				t.Fatalf("input slice mutated: %q option %q selected", in[i].ID, o.Value)
			}
		}
	}

	// A toggle that was Selected in the input but absent from values is
	// cleared in the copy (values are the whole truth) while the input keeps it.
	in[1].Options[1].Selected = true
	out = filterbar.WithSelected(in, map[string][]string{"status": {"published"}})
	if out[1].Options[1].Selected {
		t.Fatalf("expected draft cleared when absent from values, got %+v", out[1].Options)
	}
	if !in[1].Options[1].Selected {
		t.Fatalf("input slice mutated: draft deselected")
	}
}

// TestWithSelected_CommaJoinedToggle: filterbar.min.js submits multi-selected
// toggle values comma-joined in one parameter; WithSelected understands that
// form for toggles as well as repeated values.
func TestWithSelected_CommaJoinedToggle(t *testing.T) {
	in := richProps().Filters
	out := filterbar.WithSelected(in, map[string][]string{"status": {"published,draft"}})
	if !out[1].Options[0].Selected || !out[1].Options[1].Selected {
		t.Fatalf("expected both toggles selected from comma-joined value, got %+v", out[1].Options)
	}
}

// TestWithSelected_NilValues: nil values yields a copy with nothing selected.
func TestWithSelected_NilValues(t *testing.T) {
	in := richProps().Filters
	out := filterbar.WithSelected(in, nil)
	for _, f := range out {
		if f.Value != "" {
			t.Fatalf("expected empty Value, got %q", f.Value)
		}
		for _, o := range f.Options {
			if o.Selected {
				t.Fatalf("expected nothing selected, got %+v", f.Options)
			}
		}
	}
	if !in[0].Options[1].Selected {
		t.Fatalf("input slice mutated")
	}
}

// TestFilterBar_OptionAttributes: FilterOption.Attributes spread onto the
// <option>, after selected; options without them render as before.
func TestFilterBar_OptionAttributes(t *testing.T) {
	html := render(t, filterbar.Props{Filters: []filterbar.FilterConfig{{
		ID: "sort", Type: filterbar.TypeSelect,
		Options: []filterbar.FilterOption{
			{Label: "Name ↑", Value: "name", Attributes: templ.Attributes{"data-tui-filterbar-set-dir": "asc"}},
			{Label: "Name ↓", Value: "name", Selected: true, Attributes: templ.Attributes{"data-tui-filterbar-set-dir": "desc"}},
			{Label: "Plain", Value: "plain"},
		},
	}}})
	for _, want := range []string{
		`<option value="name" data-tui-filterbar-set-dir="asc">Name ↑</option>`,
		`<option value="name" selected data-tui-filterbar-set-dir="desc">Name ↓</option>`,
		`<option value="plain">Plain</option>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in: %s", want, html)
		}
	}
}

// TestFilterBar_PhoneOnly: a PhoneOnly control's wrapper is md:hidden; the
// control itself is unchanged, so the script still reads it on md and up.
func TestFilterBar_PhoneOnly(t *testing.T) {
	html := render(t, filterbar.Props{Filters: []filterbar.FilterConfig{
		{ID: "sort", Label: "Sort", Type: filterbar.TypeSelect, PhoneOnly: true, Options: []filterbar.FilterOption{{Label: "A", Value: "a"}}},
		{ID: "q", Label: "Search", Type: filterbar.TypeSearch},
	}})
	i := strings.Index(html, ">Sort</label>")
	wrapper := html[strings.LastIndex(html[:i], "<div"):i]
	if !strings.Contains(wrapper, "md:hidden") {
		t.Errorf("PhoneOnly wrapper lacks md:hidden: %s", wrapper)
	}
	j := strings.Index(html, ">Search</label>")
	other := html[strings.LastIndex(html[:j], "<div"):j]
	if strings.Contains(other, "md:hidden") {
		t.Errorf("non-PhoneOnly wrapper has md:hidden: %s", other)
	}
	if !strings.Contains(html, `<select name="sort" data-tui-filterbar-control class=`) {
		t.Errorf("PhoneOnly must not change the control itself: %s", html)
	}
}

// TestFilterBar_StateControl: a Hidden key naming a TypeSelect or
// TypeSearch control renders no hidden input for it; the control takes the
// input's id and the state marker, its label points at it, and the map
// value is ignored. A TypeToggle key is not promoted.
func TestFilterBar_StateControl(t *testing.T) {
	html := render(t, filterbar.Props{
		ID:     "bar",
		Hidden: map[string]string{"sort": "ignored", "dir": "asc", "q": "ignored", "status": "x"},
		Filters: []filterbar.FilterConfig{
			{ID: "sort", Label: "Sort", Type: filterbar.TypeSelect, Options: []filterbar.FilterOption{{Label: "Name", Value: "name", Selected: true}}},
			{ID: "q", Label: "Search", Type: filterbar.TypeSearch, Value: "blade"},
			{ID: "status", Label: "Status", Type: filterbar.TypeToggle, Options: []filterbar.FilterOption{{Label: "Draft", Value: "draft"}}},
		},
	})
	for _, want := range []string{
		`<input type="hidden" id="bar-hidden-dir" name="dir" value="asc" data-tui-filterbar-control>`,
		`<input type="hidden" id="bar-hidden-status" name="status" value="x" data-tui-filterbar-control>`,
		`<label for="bar-hidden-sort" class=`,
		`<select id="bar-hidden-sort" name="sort" data-tui-filterbar-control data-tui-filterbar-state class=`,
		`<option value="name" selected>Name</option>`,
		`<label for="bar-hidden-q" class=`,
		`<input type="search" id="bar-hidden-q" name="q" data-tui-filterbar-control data-tui-filterbar-state value="blade"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in: %s", want, html)
		}
	}
	for _, bad := range []string{`type="hidden" id="bar-hidden-sort"`, `type="hidden" id="bar-hidden-q"`, "ignored"} {
		if strings.Contains(html, bad) {
			t.Errorf("unexpected %q in: %s", bad, html)
		}
	}
	if n := strings.Count(html, `type="hidden"`); n != 2 {
		t.Errorf("%d hidden inputs, want 2 (dir, status): %s", n, html)
	}
	for _, name := range []string{`name="sort"`, `name="q"`} {
		if n := strings.Count(html, name); n != 1 {
			t.Errorf("%s appears %d times, want 1: %s", name, n, html)
		}
	}
	// Without a bar id the promoted control has no id and no label for.
	html = render(t, filterbar.Props{
		Hidden:  map[string]string{"sort": ""},
		Filters: []filterbar.FilterConfig{{ID: "sort", Type: filterbar.TypeSelect}},
	})
	if strings.Contains(html, ` id=`) || strings.Contains(html, ` for=`) {
		t.Errorf("no id without Props.ID: %s", html)
	}
	if !strings.Contains(html, `<select name="sort" data-tui-filterbar-control data-tui-filterbar-state`) || strings.Contains(html, `type="hidden"`) {
		t.Errorf("control not promoted without Props.ID: %s", html)
	}
}

// TestHiddenOOB_Controls: a key that names one of the passed controls
// renders that bare control (no wrapper, no label) with the input's id and
// hx-swap-oob="true" in the input's place, in sorted key order with the
// rest; controls whose ID is not a key are ignored.
func TestHiddenOOB_Controls(t *testing.T) {
	var buf bytes.Buffer
	sort := filterbar.FilterConfig{ID: "sort", Label: "Sort", Type: filterbar.TypeSelect, PhoneOnly: true,
		Options: []filterbar.FilterOption{{Label: "Name ↓", Value: "name", Selected: true, Attributes: templ.Attributes{"data-tui-filterbar-set-dir": "desc"}}}}
	q := filterbar.FilterConfig{ID: "q", Type: filterbar.TypeSearch, Value: "blade"}
	extra := filterbar.FilterConfig{ID: "type", Type: filterbar.TypeSelect}
	err := filterbar.HiddenOOB("bar", map[string]string{"sort": "", "dir": "desc", "q": ""}, sort, q, extra).Render(context.Background(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	want := `<input type="hidden" id="bar-hidden-dir" name="dir" value="desc" data-tui-filterbar-control hx-swap-oob="true">` +
		`<input type="search" id="bar-hidden-q" name="q" data-tui-filterbar-control data-tui-filterbar-state hx-swap-oob="true" value="blade" data-tui-filterbar-debounce="400" placeholder="" class="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring w-48">` +
		`<select id="bar-hidden-sort" name="sort" data-tui-filterbar-control data-tui-filterbar-state hx-swap-oob="true" class="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring">` +
		`<option value="name" selected data-tui-filterbar-set-dir="desc">Name ↓</option></select>`
	if got := buf.String(); got != want {
		t.Errorf("HiddenOOB =\n%s\nwant\n%s", got, want)
	}
}

// v030NoID is what FilterBar rendered in v0.3.0 for richProps without an ID:
// every control type of that version, a reset button, no Hidden. A bar that
// sets none of the fields added since must still render exactly this.
const v030NoID = `<div data-tui-filterbar data-tui-filterbar-hx-get="/items" data-tui-filterbar-hx-target="#results" data-tui-filterbar-hx-swap="innerHTML" class="flex flex-wrap items-center gap-3 p-3 bg-card border border-border rounded-lg mt-2">` +
	`<div class="flex items-center gap-2"><label class="text-xs font-medium text-muted-foreground whitespace-nowrap">Type</label>` +
	`<select name="type" data-tui-filterbar-control class="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"><option value="">Any type</option> <option value="weapon">Weapon</option><option value="armor" selected>Armor</option></select></div>` +
	`<div class="flex items-center gap-2"><span class="text-xs font-medium text-muted-foreground whitespace-nowrap">Status</span><div class="flex gap-1">` +
	`<button type="button" name="status" value="published" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="true" class="px-2.5 py-1 rounded-full text-xs font-medium border transition-colors border-primary bg-primary text-primary-foreground">Published</button>` +
	`<button type="button" name="status" value="draft" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="false" class="px-2.5 py-1 rounded-full text-xs font-medium border transition-colors border-border bg-background text-muted-foreground hover:bg-accent hover:text-accent-foreground">Draft</button></div></div>` +
	`<div class="flex items-center gap-2"><label class="text-xs font-medium text-muted-foreground whitespace-nowrap">Search</label>` +
	`<input type="search" name="q" data-tui-filterbar-control data-tui-filterbar-debounce="400" placeholder="Search items" class="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring w-48"></div>` +
	`<button type="button" data-tui-filterbar-reset class="ml-auto text-xs text-muted-foreground hover:text-foreground transition-colors">Clear filters</button></div>`

// TestFilterBar_UnchangedFromV030: the zero value of every field added
// after v0.3.0 (FilterOption.Count, TypeChoice) changes nothing: a bar of
// that version's props renders byte for byte what it rendered then. With an
// ID the one difference is the ids: the root's, and since issue 74 one per
// control (ControlID), with the label's for.
func TestFilterBar_UnchangedFromV030(t *testing.T) {
	p := richProps()
	p.ID = ""
	if got := render(t, p); got != v030NoID {
		t.Errorf("FilterBar without an ID =\n%s\nwant the v0.3.0 render\n%s", got, v030NoID)
	}
	want := strings.NewReplacer(
		`<div data-tui-filterbar `, `<div id="bar" data-tui-filterbar `,
		`<label class="text-xs font-medium text-muted-foreground whitespace-nowrap">Type</label><select name="type"`, `<label for="bar-control-type" class="text-xs font-medium text-muted-foreground whitespace-nowrap">Type</label><select id="bar-control-type" name="type"`,
		`<div class="flex gap-1">`, `<div id="bar-control-status" class="flex gap-1">`,
		`<label class="text-xs font-medium text-muted-foreground whitespace-nowrap">Search</label><input type="search" name="q"`, `<label for="bar-control-q" class="text-xs font-medium text-muted-foreground whitespace-nowrap">Search</label><input type="search" id="bar-control-q" name="q"`,
	).Replace(v030NoID)
	if got := render(t, richProps()); got != want {
		t.Errorf("FilterBar with an ID =\n%s\nwant the v0.3.0 render plus the ids\n%s", got, want)
	}
}

// TestFilterBar_ControlIDs: with a bar ID every control has an id: a
// select and a search box on the control itself (the label points at it),
// a toggle and a choice group on the element that holds the buttons. A
// state control keeps its HiddenID. Without a bar ID no control has an id.
func TestFilterBar_ControlIDs(t *testing.T) {
	p := richProps()
	p.Filters = append(p.Filters, choiceProps().Filters[0],
		filterbar.FilterConfig{ID: "items.sort", Label: "Sort", Type: filterbar.TypeSelect})
	p.Hidden = map[string]string{"items.sort": ""}
	html := render(t, p)
	for _, want := range []string{
		`<label for="bar-control-type" class=`, `<select id="bar-control-type" name="type" data-tui-filterbar-control class=`,
		`<label for="bar-control-q" class=`, `<input type="search" id="bar-control-q" name="q" data-tui-filterbar-control data-tui-filterbar-debounce=`,
		`<div id="bar-control-status" class="flex gap-1"><button type="button" name="status"`,
		`<div id="bar-control-school" role="group" aria-label="School" data-tui-filterbar-choice class=`,
		`<label for="bar-hidden-items-sort" class=`, `<select id="bar-hidden-items-sort" name="items.sort" data-tui-filterbar-control data-tui-filterbar-state class=`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in: %s", want, html)
		}
	}
	if n := strings.Count(html, ` id="`); n != 6 {
		t.Errorf("%d ids, want 6 (the bar and five controls): %s", n, html)
	}
	if strings.Contains(html, "hx-swap-oob") {
		t.Errorf("FilterBar must not render oob controls: %s", html)
	}
	p.ID = ""
	html = render(t, p)
	if strings.Contains(html, ` id=`) || strings.Contains(html, ` for=`) {
		t.Errorf("no id and no for without Props.ID: %s", html)
	}
	for _, c := range []struct{ bar, name, want string }{
		{"bar", "type", "bar-control-type"},
		{"list-filterbar", "items.status", "list-filterbar-control-items-status"},
		{"", "type", ""},
	} {
		if got := filterbar.ControlID(c.bar, c.name); got != c.want {
			t.Errorf("ControlID(%q, %q) = %q, want %q", c.bar, c.name, got, c.want)
		}
	}
}

// TestControlsOOB: each select, toggle and choice group renders bare (no
// wrapper, no label) with its ControlID, hx-swap-oob="true" and the options
// it was given; a search control renders nothing, and nothing renders
// without a bar id.
func TestControlsOOB(t *testing.T) {
	renderOOB := func(id string, controls ...filterbar.FilterConfig) string {
		var buf bytes.Buffer
		if err := filterbar.ControlsOOB(id, controls...).Render(context.Background(), &buf); err != nil {
			t.Fatalf("render: %v", err)
		}
		return buf.String()
	}
	typ := filterbar.FilterConfig{ID: "type", Label: "Type", Placeholder: "Any type", Type: filterbar.TypeSelect,
		Options: []filterbar.FilterOption{{Label: "Weapon", Value: "weapon", Count: filterbar.Count(2)}, {Label: "Armor", Value: "armor", Selected: true, Count: filterbar.Count(0)}}}
	status := filterbar.FilterConfig{ID: "items.status", Label: "Status", Type: filterbar.TypeToggle,
		Options: []filterbar.FilterOption{{Label: "Draft", Value: "draft", Selected: true, Count: filterbar.Count(3)}}}
	school := choiceProps().Filters[0]
	school.Options[0].Count = filterbar.Count(7)
	school.Options[0].Selected = true
	q := filterbar.FilterConfig{ID: "q", Label: "Search", Type: filterbar.TypeSearch, Value: "blade"}

	html := renderOOB("bar", typ, q, status, school)
	want := `<select id="bar-control-type" name="type" data-tui-filterbar-control hx-swap-oob="true" class="h-8 rounded-md border border-input bg-background px-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring">` +
		`<option value="">Any type</option> <option value="weapon">Weapon (2)</option><option value="armor" selected>Armor (0)</option></select>` +
		`<div id="bar-control-items-status" hx-swap-oob="true" class="flex gap-1">` +
		`<button type="button" name="items.status" value="draft" data-tui-filterbar-toggle data-tui-filterbar-toggle-selected="true" class="px-2.5 py-1 rounded-full text-xs font-medium border transition-colors border-primary bg-primary text-primary-foreground">Draft <span data-tui-filterbar-count class="ml-0.5 tabular-nums opacity-70">3</span></button></div>` +
		`<div id="bar-control-school" role="group" aria-label="School" data-tui-filterbar-choice hx-swap-oob="true" class="flex gap-1 min-w-0 overflow-x-auto p-0.5">`
	if !strings.HasPrefix(html, want) {
		t.Errorf("ControlsOOB =\n%s\nwant it to start with\n%s", html, want)
	}
	if evo := button(t, html, "evocation"); !strings.Contains(evo, `aria-pressed="true"`) || !strings.Contains(evo, `>7</span>`) {
		t.Errorf("the choice group must carry the new count and selection: %s", evo)
	}
	if n := strings.Count(html, `hx-swap-oob="true"`); n != 3 {
		t.Errorf("%d oob elements, want 3 (no search box): %s", n, html)
	}
	for _, bad := range []string{`type="search"`, "blade", "<label", ">Status<", ">School<"} {
		if strings.Contains(html, bad) {
			t.Errorf("unexpected %q in: %s", bad, html)
		}
	}
	if got := renderOOB("bar", q); got != "" {
		t.Errorf("ControlsOOB of a search control = %q, want nothing", got)
	}
	if got := renderOOB("", typ, status, school); got != "" {
		t.Errorf("ControlsOOB without a bar id = %q, want nothing", got)
	}
	if got := renderOOB("bar"); got != "" {
		t.Errorf("ControlsOOB with no controls = %q, want nothing", got)
	}
}

// choiceProps is a bar with one TypeChoice group: a Placeholder button, a
// counted option, an option counted 0 and an option with no count.
func choiceProps() filterbar.Props {
	return filterbar.Props{Filters: []filterbar.FilterConfig{{
		ID: "school", Label: "School", Placeholder: "All", Type: filterbar.TypeChoice,
		Options: []filterbar.FilterOption{
			{Label: "Evocation", Value: "evocation", Count: filterbar.Count(54)},
			{Label: "Illusion", Value: "illusion", Count: filterbar.Count(0)},
			{Label: "Other", Value: "other"},
		},
	}}}
}

// button returns the <button> element in html whose value attribute is value.
func button(t *testing.T, html, value string) string {
	t.Helper()
	i := strings.Index(html, `value="`+value+`"`)
	if i < 0 {
		t.Fatalf("no button with value %q in: %s", value, html)
	}
	start := strings.LastIndex(html[:i], "<button")
	return html[start : i+strings.Index(html[i:], "</button>")+len("</button>")]
}

// TestFilterBar_Choice: a TypeChoice group is a role="group" named by the
// label, with one button per option plus the Placeholder button, each with
// aria-pressed and the toggle markers the script reads; the count renders
// when set (0 as "0") and no count element renders when it is nil.
func TestFilterBar_Choice(t *testing.T) {
	html := render(t, choiceProps())
	if !strings.Contains(html, `<div role="group" aria-label="School" data-tui-filterbar-choice class="flex gap-1 min-w-0 overflow-x-auto p-0.5">`) {
		t.Errorf("missing the group element in: %s", html)
	}
	if n := strings.Count(html, "<button"); n != 4 {
		t.Errorf("%d buttons, want 4 (All and three options): %s", n, html)
	}
	if n := strings.Count(html, `name="school"`); n != 4 {
		t.Errorf("%d buttons named school, want 4: %s", n, html)
	}
	if n := strings.Count(html, "aria-pressed="); n != 4 {
		t.Errorf("%d aria-pressed attributes, want one per button: %s", n, html)
	}
	// Nothing selected: the Placeholder button is, and it has no value.
	all := button(t, html, "")
	for _, want := range []string{`data-tui-filterbar-choice-none`, `data-tui-filterbar-toggle-selected="true"`, `aria-pressed="true"`, `bg-primary`, `>All</button>`} {
		if !strings.Contains(all, want) {
			t.Errorf("Placeholder button lacks %q: %s", want, all)
		}
	}
	evo := button(t, html, "evocation")
	for _, want := range []string{`type="button"`, `data-tui-filterbar-toggle `, `data-tui-filterbar-toggle-selected="false"`, `aria-pressed="false"`, `shrink-0 whitespace-nowrap`, `>Evocation <span data-tui-filterbar-count class="ml-0.5 tabular-nums opacity-70">54</span></button>`} {
		if !strings.Contains(evo, want) {
			t.Errorf("counted option lacks %q: %s", want, evo)
		}
	}
	if strings.Contains(evo, "data-tui-filterbar-choice-none") {
		t.Errorf("an option must not carry the Placeholder marker: %s", evo)
	}
	if ill := button(t, html, "illusion"); !strings.Contains(ill, `>0</span>`) {
		t.Errorf("a count of 0 must render: %s", ill)
	}
	if other := button(t, html, "other"); !strings.Contains(other, `>Other</button>`) || strings.Contains(other, "data-tui-filterbar-count") {
		t.Errorf("an option with a nil Count must render no count element: %s", other)
	}
	if n := strings.Count(html, "data-tui-filterbar-count"); n != 2 {
		t.Errorf("%d count elements, want 2: %s", n, html)
	}
}

// TestFilterBar_Choice_Selection: the first Selected option alone renders
// selected, the Placeholder button is then not; without a Placeholder there
// is no extra button, and without a Label no aria-label.
func TestFilterBar_Choice_Selection(t *testing.T) {
	p := choiceProps()
	p.Filters[0].Options[1].Selected = true
	p.Filters[0].Options[2].Selected = true
	html := render(t, p)
	if n := strings.Count(html, `aria-pressed="true"`); n != 1 {
		t.Errorf("%d pressed buttons, want 1: %s", n, html)
	}
	if ill := button(t, html, "illusion"); !strings.Contains(ill, `aria-pressed="true"`) || !strings.Contains(ill, `data-tui-filterbar-toggle-selected="true"`) {
		t.Errorf("the first Selected option must render selected: %s", ill)
	}
	if all := button(t, html, ""); !strings.Contains(all, `aria-pressed="false"`) {
		t.Errorf("the Placeholder button must not be selected beside an option: %s", all)
	}
	p.Filters[0].Placeholder, p.Filters[0].Label = "", ""
	html = render(t, p)
	if strings.Contains(html, "data-tui-filterbar-choice-none") || strings.Count(html, "<button") != 3 {
		t.Errorf("no Placeholder, no extra button: %s", html)
	}
	if strings.Contains(html, "aria-label") {
		t.Errorf("no Label, no aria-label: %s", html)
	}
}

// TestWithSelected_Choice: a TypeChoice option is selected from the query
// value; of several values the first non-empty one counts, a comma-joined
// value is one value, and no value deselects every option.
func TestWithSelected_Choice(t *testing.T) {
	in := choiceProps().Filters
	in[0].Options[2].Selected = true
	out := filterbar.WithSelected(in, map[string][]string{"school": {"", "illusion", "evocation"}})
	if o := out[0].Options; o[0].Selected || !o[1].Selected || o[2].Selected {
		t.Errorf("expected illusion alone selected, got %+v", o)
	}
	if !in[0].Options[2].Selected || in[0].Options[1].Selected {
		t.Errorf("input slice mutated: %+v", in[0].Options)
	}
	html := render(t, filterbar.Props{Filters: out})
	if ill := button(t, html, "illusion"); !strings.Contains(ill, `aria-pressed="true"`) {
		t.Errorf("the option of the query value must render selected: %s", ill)
	}
	out = filterbar.WithSelected(in, map[string][]string{"school": {"illusion,evocation"}})
	for _, o := range out[0].Options {
		if o.Selected {
			t.Errorf("a comma-joined value is not an option value, got %+v", out[0].Options)
		}
	}
	out = filterbar.WithSelected(in, nil)
	for _, o := range out[0].Options {
		if o.Selected {
			t.Errorf("expected nothing selected without a value, got %+v", out[0].Options)
		}
	}
}

// TestFilterBar_Counts: FilterOption.Count renders in a toggle pill after
// the label and in a select option in parentheses; a nil Count leaves both
// as they were.
func TestFilterBar_Counts(t *testing.T) {
	html := render(t, filterbar.Props{Filters: []filterbar.FilterConfig{
		{ID: "status", Type: filterbar.TypeToggle, Options: []filterbar.FilterOption{
			{Label: "Published", Value: "published", Count: filterbar.Count(11)},
			{Label: "Draft", Value: "draft"},
		}},
		{ID: "type", Type: filterbar.TypeSelect, Options: []filterbar.FilterOption{
			{Label: "Weapon", Value: "weapon", Count: filterbar.Count(5)},
			{Label: "Armor", Value: "armor", Count: filterbar.Count(0)},
			{Label: "Other", Value: "other"},
		}},
	}})
	for _, want := range []string{
		`>Published <span data-tui-filterbar-count class="ml-0.5 tabular-nums opacity-70">11</span></button>`,
		`>Draft</button>`,
		`<option value="weapon">Weapon (5)</option>`,
		`<option value="armor">Armor (0)</option>`,
		`<option value="other">Other</option>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in: %s", want, html)
		}
	}
}

// classOf returns the class tokens of the first element in html that
// starts with open (for example `<input type="search"`).
func classOf(t *testing.T, html, open string) []string {
	t.Helper()
	i := strings.Index(html, open)
	if i < 0 {
		t.Fatalf("no %q in: %s", open, html)
	}
	tag := html[i : i+strings.Index(html[i:], ">")]
	j := strings.Index(tag, `class="`)
	if j < 0 {
		t.Fatalf("no class on %q: %s", open, tag)
	}
	rest := tag[j+len(`class="`):]
	return strings.Fields(rest[:strings.Index(rest, `"`)])
}

func hasClass(classes []string, c string) bool {
	for _, x := range classes {
		if x == c {
			return true
		}
	}
	return false
}

// TestFilterBar_Grow: a search control with Grow has a growing wrapper and
// an input that fills it with a 192px floor, and no fixed w-48; without
// Grow the input is w-48 as before. Grow on another type changes nothing.
func TestFilterBar_Grow(t *testing.T) {
	html := render(t, filterbar.Props{Filters: []filterbar.FilterConfig{
		{ID: "q", Label: "Search", Type: filterbar.TypeSearch, Grow: true},
	}})
	input := classOf(t, html, `<input type="search"`)
	if !hasClass(input, "flex-1") || !hasClass(input, "min-w-48") || hasClass(input, "w-48") {
		t.Errorf("Grow input classes = %v, want flex-1 and min-w-48 and no w-48", input)
	}
	if !strings.Contains(html, `<div class="flex items-center gap-2 flex-1"><label`) {
		t.Errorf("Grow wrapper must be flex-1 with the label inside: %s", html)
	}
	html = render(t, filterbar.Props{Filters: []filterbar.FilterConfig{
		{ID: "q", Label: "Search", Type: filterbar.TypeSearch},
		{ID: "type", Label: "Type", Type: filterbar.TypeSelect, Grow: true},
	}})
	input = classOf(t, html, `<input type="search"`)
	if !hasClass(input, "w-48") || hasClass(input, "flex-1") || hasClass(input, "min-w-48") {
		t.Errorf("input classes without Grow = %v, want w-48 alone", input)
	}
	if strings.Contains(html, "flex-1") {
		t.Errorf("Grow must not change a select, and no wrapper grows without it: %s", html)
	}
}

// TestFilterBar_Bare: Bare drops the card (padding, background, border,
// rounded corners) and keeps the layout classes; Class still merges.
// Without Bare the root class list is the one of v0.3.0.
func TestFilterBar_Bare(t *testing.T) {
	html := render(t, filterbar.Props{Bare: true, Class: "mt-2", Filters: richProps().Filters})
	root := classOf(t, html, `<div data-tui-filterbar`)
	if got := strings.Join(root, " "); got != "flex flex-wrap items-center gap-3 mt-2" {
		t.Errorf("Bare root classes = %q", got)
	}
	for _, bad := range []string{"p-3", "bg-card", "border", "border-border", "rounded-lg"} {
		if hasClass(root, bad) {
			t.Errorf("Bare root must not have %q: %v", bad, root)
		}
	}
	// The controls are untouched: only the root's class list differs.
	card := render(t, filterbar.Props{Class: "mt-2", Filters: richProps().Filters})
	if want := strings.Replace(card, ` p-3 bg-card border border-border rounded-lg mt-2">`, ` mt-2">`, 1); html != want {
		t.Errorf("Bare must change the root class list only:\n%s\nwant\n%s", html, want)
	}
	root = classOf(t, card, `<div data-tui-filterbar`)
	if got := strings.Join(root, " "); got != "flex flex-wrap items-center gap-3 p-3 bg-card border border-border rounded-lg mt-2" {
		t.Errorf("root classes without Bare = %q, want the v0.3.0 list", got)
	}
}

// TestHiddenOOB_GrowSearch: HiddenOOB renders a promoted search control
// with the width classes the bar gave it.
func TestHiddenOOB_GrowSearch(t *testing.T) {
	q := filterbar.FilterConfig{ID: "q", Label: "Search", Type: filterbar.TypeSearch, Value: "blade", Grow: true}
	bar := render(t, filterbar.Props{ID: "bar", Hidden: map[string]string{"q": ""}, Filters: []filterbar.FilterConfig{q}})
	var buf bytes.Buffer
	if err := filterbar.HiddenOOB("bar", map[string]string{"q": ""}, q).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	oob := buf.String()
	if !strings.HasPrefix(oob, `<input type="search" id="bar-hidden-q" name="q" data-tui-filterbar-control data-tui-filterbar-state hx-swap-oob="true" value="blade"`) {
		t.Errorf("HiddenOOB = %s", oob)
	}
	inBar, inOOB := classOf(t, bar, `<input type="search"`), classOf(t, oob, `<input type="search"`)
	if strings.Join(inBar, " ") != strings.Join(inOOB, " ") {
		t.Errorf("HiddenOOB search classes = %v, the bar's = %v", inOOB, inBar)
	}
	if !hasClass(inOOB, "flex-1") || !hasClass(inOOB, "min-w-48") || hasClass(inOOB, "w-48") {
		t.Errorf("HiddenOOB search classes = %v, want the Grow classes", inOOB)
	}
}
