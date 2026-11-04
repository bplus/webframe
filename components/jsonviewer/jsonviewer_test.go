package jsonviewer_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/jsonviewer"
)

func render(t *testing.T, p jsonviewer.Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := jsonviewer.JSONViewer(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

type item struct {
	Name  string            `json:"name"`
	Count int               `json:"count"`
	Done  bool              `json:"done"`
	Note  *string           `json:"note"`
	Tags  []string          `json:"tags"`
	Meta  map[string]any    `json:"meta"`
	Empty map[string]string `json:"empty"`
}

func sample() item {
	return item{
		Name:  "widget",
		Count: 42,
		Tags:  []string{"a", "b", "c"},
		Meta:  map[string]any{"z": 1.5, "a": map[string]any{"deep": "x"}},
		Empty: map[string]string{},
	}
}

func TestNestedStructure(t *testing.T) {
	html := render(t, jsonviewer.Props{ID: "jv", Value: sample()})
	for _, want := range []string{
		`id="jv"`,
		`data-path="$"`, "{7}",
		`data-path="$.name"`, `&#34;name&#34;:`, `text-success`, `&#34;widget&#34;`,
		`data-path="$.count"`, `text-info`, ">42<",
		`data-path="$.done"`, `text-warning`, ">false<",
		`data-path="$.note"`, `italic`, ">null<",
		`data-path="$.tags"`, "[3]", `data-path="$.tags[0]"`, `data-path="$.tags[2]"`,
		`data-path="$.meta.a.deep"`,
		`data-path="$.empty"`, ">{}<",
		"overflow-x-auto", "font-mono", "break-all",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
	// Struct field order is kept (not sorted) by default.
	if strings.Index(html, `data-path="$.name"`) > strings.Index(html, `data-path="$.count"`) {
		t.Errorf("expected struct field order name before count")
	}
	// Every open container is a <details open>.
	if strings.Contains(html, "<details class=\"group/json\" data-path=\"$.tags\">") {
		t.Errorf("tags should be open with Collapsed 0")
	}
}

func TestEscaping(t *testing.T) {
	v := map[string]any{
		`<b>key</b>`: `<script>alert("x")</script>`,
	}
	html := render(t, jsonviewer.Props{Value: v})
	if strings.Contains(html, "<script>") || strings.Contains(html, "<b>") {
		t.Fatalf("unescaped markup in:\n%s", html)
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;b&gt;key&lt;/b&gt;"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
	// A non-identifier key is bracketed in the path.
	if !strings.Contains(html, `data-path="$[&#34;&lt;b&gt;key&lt;/b&gt;&#34;]"`) {
		t.Errorf("expected bracketed path for non-identifier key in:\n%s", html)
	}
}

func TestCollapsedDepth(t *testing.T) {
	v := map[string]any{"outer": map[string]any{"inner": map[string]any{"leaf": 1}}}
	html := render(t, jsonviewer.Props{Value: v, Collapsed: 2})
	open := func(path string) bool {
		return strings.Contains(html, `data-path="`+path+`" open>`)
	}
	if !open("$") {
		t.Errorf("root should be open")
	}
	if !open("$.outer") {
		t.Errorf("depth 1 should be open with Collapsed 2")
	}
	if open("$.outer.inner") {
		t.Errorf("depth 2 should be closed with Collapsed 2")
	}
	if !strings.Contains(html, `data-path="$.outer.inner">`) {
		t.Errorf("depth 2 should still render as <details> in:\n%s", html)
	}

	all := render(t, jsonviewer.Props{Value: v, Collapsed: 1})
	if !strings.Contains(all, `data-path="$" open>`) || strings.Contains(all, `data-path="$.outer" open>`) {
		t.Errorf("Collapsed 1 should open only the root in:\n%s", all)
	}
}

func TestMaxDepthElision(t *testing.T) {
	v := map[string]any{"outer": map[string]any{"inner": map[string]any{"leaf": 1, "more": 2}, "flat": "ok"}}
	html := render(t, jsonviewer.Props{Value: v, MaxDepth: 1})
	if !strings.Contains(html, `data-path="$.outer" open>`) {
		t.Errorf("depth 1 container should render in:\n%s", html)
	}
	if strings.Contains(html, `data-path="$.outer.inner.leaf"`) {
		t.Errorf("children of an elided node must not render in:\n%s", html)
	}
	if !strings.Contains(html, "… {2}") {
		t.Errorf("elided node should show an ellipsis with its size in:\n%s", html)
	}
	if !strings.Contains(html, `data-path="$.outer.flat"`) {
		t.Errorf("scalars inside the deepest rendered container still print in:\n%s", html)
	}
}

func TestSort(t *testing.T) {
	type s struct {
		Zeta  int `json:"zeta"`
		Alpha int `json:"alpha"`
		Mid   int `json:"mid"`
	}
	unsorted := render(t, jsonviewer.Props{Value: s{}})
	if !(strings.Index(unsorted, `"$.zeta"`) < strings.Index(unsorted, `"$.alpha"`)) {
		t.Errorf("without Sort the marshaller's order is kept")
	}
	sorted := render(t, jsonviewer.Props{Value: s{}, Sort: true})
	a, m, z := strings.Index(sorted, `"$.alpha"`), strings.Index(sorted, `"$.mid"`), strings.Index(sorted, `"$.zeta"`)
	if !(a < m && m < z) {
		t.Errorf("Sort should order keys alpha, mid, zeta: %d %d %d", a, m, z)
	}
}

func TestCopyPath(t *testing.T) {
	v := map[string]any{"a": []any{1}}
	plain := render(t, jsonviewer.Props{Value: v})
	if strings.Contains(plain, "clipboard") {
		t.Errorf("no copy buttons without CopyPath")
	}
	html := render(t, jsonviewer.Props{Value: v, CopyPath: true})
	for _, want := range []string{
		`aria-label="Copy path $.a[0]"`,
		`writeText(&#39;$.a&#39;)`,
		`x-on:click.prevent.stop`, // the summary's button must not toggle the node
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
}

func TestIndentCap(t *testing.T) {
	// Ten levels deep: the children wrapper gains pl-4 for the first six
	// levels only.
	var v any = "leaf"
	for i := 0; i < 10; i++ {
		v = map[string]any{"k": v}
	}
	html := render(t, jsonviewer.Props{Value: v})
	got := strings.Count(html, `<div class="flex flex-col pl-4">`)
	if got != 6 {
		t.Errorf("expected 6 indented levels, got %d", got)
	}
	if strings.Count(html, `<div class="flex flex-col">`) != 4 {
		t.Errorf("expected 4 flush levels past the cap in:\n%s", html)
	}
}

func TestNumbersKeepTheirLiteral(t *testing.T) {
	html := render(t, jsonviewer.Props{Value: map[string]any{"big": int64(1234567890123), "f": 0.5}})
	if !strings.Contains(html, ">1234567890123<") {
		t.Errorf("integers should not render in exponent form:\n%s", html)
	}
	if !strings.Contains(html, ">0.5<") {
		t.Errorf("missing 0.5 in:\n%s", html)
	}
}

func TestUnmarshallableValue(t *testing.T) {
	html := render(t, jsonviewer.Props{Value: make(chan int)})
	if !strings.Contains(html, "text-destructive") || !strings.Contains(html, "cannot render value") {
		t.Errorf("expected an error message in:\n%s", html)
	}
}

func TestZeroValue(t *testing.T) {
	html := render(t, jsonviewer.Props{})
	if !strings.Contains(html, ">null<") {
		t.Errorf("nil Value renders as null:\n%s", html)
	}
}

// TestCopyPathHiddenUntilRowFocus checks the reveal wiring: every copy
// button sits in an sr-only wrapper that the row's hover or focus (or the
// button's own focus) undoes, and leaf rows are tap-focusable without
// becoming Tab stops.
func TestCopyPathHiddenUntilRowFocus(t *testing.T) {
	v := map[string]any{"a": []any{1}, "e": map[string]any{}}
	html := render(t, jsonviewer.Props{Value: v, CopyPath: true})
	buttons := strings.Count(html, `aria-label="Copy path `)
	wrappers := strings.Count(html, `<span class="sr-only shrink-0 group-hover/row:not-sr-only group-focus-within/row:not-sr-only focus-within:not-sr-only">`)
	if buttons == 0 || wrappers != buttons {
		t.Errorf("want one hidden wrapper per copy button, got %d wrappers for %d buttons:\n%s", wrappers, buttons, html)
	}
	for _, want := range []string{
		`<summary class="group/row `,       // a summary row reveals its button on focus
		`data-path="$.a[0]" tabindex="-1"`, // a scalar row is tap-focusable
		`data-path="$.e" tabindex="-1"`,    // so is an empty container row
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in:\n%s", want, html)
		}
	}
	plain := render(t, jsonviewer.Props{Value: v})
	if strings.Contains(plain, "tabindex") || strings.Contains(plain, "group/row focus") {
		t.Errorf("without CopyPath rows are not focusable:\n%s", plain)
	}
}
