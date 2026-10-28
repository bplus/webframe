package autocomplete

import (
	"bytes"
	"context"
	"regexp"
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
	mustContain(t, out, "No matches", `role="option"`, `aria-disabled="true"`, "data-wf-ac-empty")
	if strings.Contains(out, "data-wf-ac-option") {
		t.Errorf("No matches row must not be pickable: %s", out)
	}
}

func TestAutocompleteMissingHxGetRendersError(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw"}))
	mustContain(t, out, `Error: Autocomplete missing required 'HxGet'`, "text-destructive")
	if strings.Contains(render(t, Autocomplete(Props{ID: "fw", HxGet: "/s"})), "Error:") {
		t.Errorf("unexpected error with HxGet set")
	}
}

func TestAutocompleteListLabelledByLabel(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", HxGet: "/s", Label: "Framework"}))
	mustContain(t, out, `id="fw-label"`, `aria-labelledby="fw-label"`)
	if strings.Contains(render(t, Autocomplete(Props{ID: "fw", HxGet: "/s"})), "aria-labelledby") {
		t.Errorf("unexpected aria-labelledby without Label")
	}
}

func TestScriptTag(t *testing.T) {
	out := render(t, Script())
	mustContain(t, out, "/static/js/autocomplete.js")
}

// boolAttr matches a bare boolean attribute (not a Tailwind "disabled:" variant).
func boolAttr(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s` + name + `[\s/>]`)
}

func TestAutocompleteRequiredDisabledLabel(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", HxGet: "/api/search", Label: "Framework", Required: true, Disabled: true}))
	mustContain(t, out, `for="fw"`, `Framework</label>`, `aria-expanded="false"`, `data-wf-ac-input`)
	for _, name := range []string{"required", "disabled"} {
		if !boolAttr(name).MatchString(out) {
			t.Errorf("missing %s attribute in %s", name, out)
		}
	}
}

func TestAutocompleteDefaultsOmitRequiredDisabledLabel(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", HxGet: "/api/search"}))
	for _, name := range []string{"required", "disabled"} {
		if boolAttr(name).MatchString(out) {
			t.Errorf("unexpected %s attribute in %s", name, out)
		}
	}
	if strings.Contains(out, "<label") {
		t.Errorf("unexpected <label in %s", out)
	}
}

func TestAutocomplete_TextName(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", HxGet: "/api/search", TextName: "title"}))
	mustContain(t, out, `name="title"`)
	if strings.Contains(out, `name="q"`) {
		t.Errorf("TextName should replace q: %s", out)
	}
}

func TestAutocomplete_DisabledHidden(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", Name: "framework", HxGet: "/api/search", Disabled: true}))
	hidden := out[strings.Index(out, `<input type="hidden"`):]
	hidden = hidden[:strings.Index(hidden, ">")+1]
	if !boolAttr("disabled").MatchString(hidden) {
		t.Errorf("hidden value input should be disabled: %s", hidden)
	}
	enabled := render(t, Autocomplete(Props{ID: "fw", Name: "framework", HxGet: "/api/search"}))
	h2 := enabled[strings.Index(enabled, `<input type="hidden"`):]
	h2 = h2[:strings.Index(h2, ">")+1]
	if boolAttr("disabled").MatchString(h2) {
		t.Errorf("hidden value input should not be disabled by default: %s", h2)
	}
}

func TestAutocomplete_RequiredMarker(t *testing.T) {
	out := render(t, Autocomplete(Props{ID: "fw", Name: "framework", HxGet: "/api/search", Required: true}))
	mustContain(t, out, "data-wf-ac-required")
	for _, p := range []Props{
		{ID: "fw", HxGet: "/api/search", Required: true},    // no hidden value: free text, browser required suffices
		{ID: "fw", Name: "framework", HxGet: "/api/search"}, // not required
	} {
		if strings.Contains(render(t, Autocomplete(p)), "data-wf-ac-required") {
			t.Errorf("unexpected data-wf-ac-required for %+v", p)
		}
	}
}
