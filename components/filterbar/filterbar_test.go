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

// richProps is a representative existing-caller configuration: every control
// type, a reset button, no Hidden, no Value. Used to show the batch-2
// additions leave such callers' markup untouched.
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
