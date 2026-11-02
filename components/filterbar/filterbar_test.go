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
