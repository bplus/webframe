package graphwidget

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := GraphWidget(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// TestLoadingStateUsesSpinner: the loading placeholder is spinner.Spinner
// (role="status", SizeLg ring, animate-spin) inside the
// data-tui-graph-loading wrapper the client script hides, with the caption
// under it.
func TestLoadingStateUsesSpinner(t *testing.T) {
	out := render(t, Props{ID: "g1", DataURL: "/graph.json"})
	i := strings.Index(out, "data-tui-graph-loading")
	if i < 0 {
		t.Fatalf("loading wrapper missing:\n%s", out)
	}
	loading := out[i:]
	for _, want := range []string{`role="status"`, "animate-spin", "h-8", "w-8", `<span class="sr-only">Loading graph</span>`, "Loading graph..."} {
		if !strings.Contains(loading, want) {
			t.Errorf("loading state missing %q:\n%s", want, loading)
		}
	}
	// The caption is aria-hidden: the spinner's label is the announcement.
	if !strings.Contains(loading, `<span class="text-xs" aria-hidden="true">Loading graph...</span>`) {
		t.Errorf("caption should be aria-hidden:\n%s", loading)
	}
	// Exactly one spinning ring.
	if n := strings.Count(loading, "animate-spin"); n != 1 {
		t.Errorf("animate-spin appears %d times, want 1", n)
	}
	if strings.Contains(loading, "h-6") {
		t.Errorf("unexpected h-6 spinner size rendered:\n%s", loading)
	}
}

func TestContainerAttributes(t *testing.T) {
	out := render(t, Props{ID: "g2", DataURL: "/d", Width: 300, Height: 200, ShowLabels: true})
	for _, want := range []string{`id="g2"`, `data-tui-graph-data-url="/d"`, `data-tui-graph-labels="true"`, "width:300px;height:200px", `data-tui-graph-color-map="{}"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestColorMapJSON_Escapes: a quote or backslash in a type or color must
// still yield valid JSON that decodes back to the same entries.
func TestColorMapJSON_Escapes(t *testing.T) {
	entries := []NodeColorEntry{
		{Type: `say "hi"`, Color: `#fff`},
		{Type: `back\slash`, Color: `rgb(1,2,3)"`},
		{Type: "plain", Color: "red"},
	}
	got := colorMapJSON(entries)
	var m map[string]string
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("colorMapJSON = %s: invalid JSON: %v", got, err)
	}
	for _, e := range entries {
		if m[e.Type] != e.Color {
			t.Errorf("decoded[%q] = %q, want %q", e.Type, m[e.Type], e.Color)
		}
	}
	// Order is the entry order (deterministic markup).
	if want := `{"say \"hi\"":"#fff","back\\slash":"rgb(1,2,3)\"","plain":"red"}`; got != want {
		t.Errorf("colorMapJSON = %s, want %s", got, want)
	}
	if got := colorMapJSON(nil); got != "{}" {
		t.Errorf("colorMapJSON(nil) = %s, want {}", got)
	}
}
