package autocomplete

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

func TestAutocompleteRendersComboboxWiring(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", Name: "framework", HxGet: "/api/search"}))
	mustContain(t, out,
		`role="combobox"`,
		`aria-controls="fw-list"`,
		`hx-get="/api/search"`,
		`hx-target="#fw-list"`,
		`hx-trigger="input changed delay:300ms"`,
		`name="q"`,
		`name="framework"`,
		`type="hidden"`,
		`id="fw-list"`,
		`role="listbox"`,
		`data-wf-ac-min="1"`,
	)
}

func TestAutocompleteNoNameNoHidden(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", HxGet: "/api/search"}))
	if strings.Contains(out, `type="hidden"`) {
		t.Errorf("unexpected hidden field in %s", out)
	}
}

func TestAutocompleteInitialValueAndText(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", Name: "framework", HxGet: "/api/search", Value: "svelte", Text: "Svelte"}))
	mustContain(t, out, `value="svelte"`, `value="Svelte"`)
}

func TestAutocompleteMinCharsAndDelay(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", HxGet: "/api/search", MinChars: 2, DelayMs: 500}))
	mustContain(t, out, `data-wf-ac-min="2"`, `delay:500ms`)
}

func TestResultsRendersOptions(t *testing.T) {
	out := render(t, Results([]Item{
		{Label: "Vue", Value: "vue", Description: "Progressive framework"},
		{Label: "Svelte", Value: "svelte"},
	}))
	if n := strings.Count(out, `role="option"`); n != 2 {
		t.Errorf("want 2 options, got %d in %s", n, out)
	}
	mustContain(t, out, `data-wf-ac-item-value="vue"`, `Progressive framework</span>`)
	if n := strings.Count(out, `text-muted-foreground`); n != 1 {
		t.Errorf("want 1 description span, got %d in %s", n, out)
	}
}

func TestResultsEmpty(t *testing.T) {
	out := render(t, Results(nil))
	mustContain(t, out, "No matches")
	if strings.Contains(out, `role="option"`) {
		t.Errorf("unexpected option in %s", out)
	}
}

func TestScriptTag(t *testing.T) {
	out := render(t, Script())
	mustContain(t, out, "/static/js/autocomplete.js")
}
