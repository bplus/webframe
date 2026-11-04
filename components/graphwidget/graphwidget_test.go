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
	out := render(t, Props{ID: "g2", DataURL: "/d", Width: 300, Height: 200})
	for _, want := range []string{`id="g2"`, `data-tui-graph-data-url="/d"`, "width:300px;height:200px", `data-tui-graph-color-map="{}"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestColorMapJSON_Escapes: a quote or backslash in a type must still
// yield valid JSON that decodes back to the same entries.
func TestColorMapJSON_Escapes(t *testing.T) {
	entries := []NodeColorEntry{
		{Type: `say "hi"`, Color: `#fff`},
		{Type: `back\slash`, Color: `rgb(1,2,3)`},
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
	if want := `{"say \"hi\"":"#fff","back\\slash":"rgb(1,2,3)","plain":"red"}`; got != want {
		t.Errorf("colorMapJSON = %s, want %s", got, want)
	}
	if got := colorMapJSON(nil); got != "{}" {
		t.Errorf("colorMapJSON(nil) = %s, want {}", got)
	}
}

// TestColors_Sanitized: node colors and the background pass through
// utils.SanitizeColor, so a value that could break out of a style (a
// semicolon, a quote, url(...)) reaches the script as the "#888" fallback.
func TestColors_Sanitized(t *testing.T) {
	entries := []NodeColorEntry{
		{Type: "a", Color: "red;background:url(http://x)"},
		{Type: "b", Color: `#fff"`},
		{Type: "c", Color: "rgb(1, 2, 3)"},
	}
	if got, want := colorMapJSON(entries), `{"a":"#888","b":"#888","c":"rgb(1, 2, 3)"}`; got != want {
		t.Errorf("colorMapJSON = %s, want %s", got, want)
	}
	var buf bytes.Buffer
	if err := GraphWidget(Props{ID: "g", BackgroundColor: "black;x:y"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if out := buf.String(); !strings.Contains(out, `data-tui-graph-bg="#888"`) || strings.Contains(out, "black;x:y") {
		t.Errorf("background not sanitized: %s", out)
	}
	buf.Reset()
	if err := GraphWidget(Props{ID: "g", BackgroundColor: "#101010"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), `data-tui-graph-bg="#101010"`) {
		t.Errorf("safe background dropped: %s", buf.String())
	}
}

// TestScriptHooks: graphwidget.min.js draws into data-tui-graph-mount (the
// library empties its element, so the spinner and error box must be
// siblings, not children) and reports failures in data-tui-graph-error,
// which starts hidden and is an alert.
func TestScriptHooks(t *testing.T) {
	out := render(t, Props{ID: "g3", DataURL: "/d"})
	mount := strings.Index(out, `<div data-tui-graph-mount class="absolute inset-0"></div>`)
	loading := strings.Index(out, "data-tui-graph-loading")
	errBox := strings.Index(out, `<div data-tui-graph-error role="alert" hidden`)
	if mount < 0 || loading < 0 || errBox < 0 {
		t.Fatalf("mount/loading/error hooks missing:\n%s", out)
	}
	if !(mount < loading && loading < errBox) {
		t.Errorf("want mount, then loading, then error box in that order:\n%s", out)
	}
}

// TestScriptLoadsLibraryThenGlue: the library tag comes first (both are
// defer, so they run in document order) and nothing else is loaded.
func TestScriptLoadsLibraryThenGlue(t *testing.T) {
	var buf bytes.Buffer
	if err := Script().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	lib := strings.Index(out, "/static/js/3d-force-graph.min.js")
	glue := strings.Index(out, "/static/js/graphwidget.min.js")
	if lib < 0 || glue < 0 || lib > glue {
		t.Errorf("want 3d-force-graph.min.js before graphwidget.min.js:\n%s", out)
	}
	if n := strings.Count(out, "<script"); n != 2 {
		t.Errorf("Script renders %d script tags, want 2:\n%s", n, out)
	}
}
