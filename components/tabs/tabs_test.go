package tabs_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/tabs"
)

func TestContentEmptyValueDoesNotEmitNopComponentText(t *testing.T) {
	var buf bytes.Buffer
	comp := tabs.Content(tabs.ContentProps{})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, "return templ.NopComponent") {
		t.Errorf("expected no literal NopComponent text, got: %s", html)
	}
	if !strings.Contains(html, "missing required 'Value' attribute") {
		t.Errorf("expected error message, got: %s", html)
	}
}

func renderTabs(t *testing.T, props ...tabs.Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tabs.Tabs(props...).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// rootClasses returns the class tokens of the first element in html as a
// set. Tokens, not substrings: utils.TwMerge does not keep a stable class
// order across processes (e.g. "flex-col gap-2 flex").
func rootClasses(t *testing.T, html string) map[string]bool {
	t.Helper()
	_, rest, ok := strings.Cut(html, `class="`)
	if !ok {
		t.Fatalf("no class attribute in: %s", html)
	}
	class, _, _ := strings.Cut(rest, `"`)
	set := map[string]bool{}
	for _, c := range strings.Fields(class) {
		set[c] = true
	}
	return set
}

func TestTabsDefaultsToTallestSizing(t *testing.T) {
	html := renderTabs(t)
	if !strings.Contains(html, `data-wf-tabs-sizing="tallest"`) {
		t.Errorf("expected tallest sizing attribute, got: %s", html)
	}
	classes := rootClasses(t, html)
	if !classes["grid"] {
		t.Errorf("expected root class to contain grid, got: %v", classes)
	}
	if classes["flex-col"] {
		t.Errorf("expected root class without flex-col, got: %v", classes)
	}
	if strings.Contains(html, "data-wf-tabs-height") {
		t.Errorf("expected no height attribute outside fixed sizing, got: %s", html)
	}
}

func TestTabsFixedSizingEmitsHeight(t *testing.T) {
	html := renderTabs(t, tabs.Props{Sizing: tabs.SizingFixed, Height: tabs.HeightS})
	if !strings.Contains(html, `data-wf-tabs-sizing="fixed"`) {
		t.Errorf("expected fixed sizing attribute, got: %s", html)
	}
	if !strings.Contains(html, `data-wf-tabs-height="s"`) {
		t.Errorf("expected height s, got: %s", html)
	}

	html = renderTabs(t, tabs.Props{Sizing: tabs.SizingFixed})
	if !strings.Contains(html, `data-wf-tabs-height="m"`) {
		t.Errorf("expected default height m, got: %s", html)
	}
}

func TestTabsFitSizingKeepsFlexColumn(t *testing.T) {
	html := renderTabs(t, tabs.Props{Sizing: tabs.SizingFit})
	if !strings.Contains(html, `data-wf-tabs-sizing="fit"`) {
		t.Errorf("expected fit sizing attribute, got: %s", html)
	}
	if classes := rootClasses(t, html); !classes["flex"] || !classes["flex-col"] {
		t.Errorf("expected root class to contain flex and flex-col, got: %v", classes)
	}
}

func TestTabsSizingCSSPresent(t *testing.T) {
	css, err := os.ReadFile("../../static/css/input.css")
	if err != nil {
		t.Fatalf("read input.css: %v", err)
	}
	for _, want := range []string{
		`[data-wf-tabs-sizing="tallest"] > [data-tui-tabs-content]`,
		`[data-wf-tabs-sizing="tallest"] > [data-tui-tabs-content][data-tui-tabs-state="inactive"]`,
		`[data-wf-tabs-sizing="fixed"] > [data-tui-tabs-content]`,
		`visibility: hidden`,
		`pointer-events: none`,
	} {
		if !strings.Contains(string(css), want) {
			t.Errorf("input.css missing %q", want)
		}
	}
}

func renderTabsWithPanel(t *testing.T, p tabs.Props) string {
	t.Helper()
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), tabs.Content(tabs.ContentProps{Value: "a", IsActive: true}))
	if err := tabs.Tabs(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestTabsFixedPanelsAreFocusable(t *testing.T) {
	if html := renderTabsWithPanel(t, tabs.Props{Sizing: tabs.SizingFixed}); !strings.Contains(html, `tabindex="0"`) {
		t.Errorf("expected fixed panel tabindex=0, got: %s", html)
	}
	for _, s := range []tabs.Sizing{tabs.SizingTallest, tabs.SizingFit} {
		if html := renderTabsWithPanel(t, tabs.Props{Sizing: s}); strings.Contains(html, "tabindex") {
			t.Errorf("sizing %q: unexpected tabindex in %s", s, html)
		}
	}
}
