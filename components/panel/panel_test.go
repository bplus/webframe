package panel

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

func TestPanel_Section(t *testing.T) {
	out := render(t, Panel(Props{Title: "Members"}))
	for _, want := range []string{"<section", "<h3", "Members"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "<details") {
		t.Errorf("unexpected <details in %s", out)
	}
}

func TestPanel_CollapsibleOpen(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Collapsible: true, Open: true}))
	for _, want := range []string{"<details open", "<summary"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestPanel_CollapsibleClosed(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Collapsible: true}))
	if !strings.Contains(out, "<details") {
		t.Errorf("missing <details in %s", out)
	}
	if strings.Contains(out, "<details open") {
		t.Errorf("unexpected open in %s", out)
	}
}

func TestPanel_Flush(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Flush: true}))
	if !strings.Contains(out, `class="p-0"`) {
		t.Errorf("missing class=p-0 in %s", out)
	}
	if strings.Contains(out, `class="p-4"`) {
		t.Errorf("unexpected class=p-4 in %s", out)
	}
}

func TestPanel_DangerVariant(t *testing.T) {
	out := render(t, Panel(Props{Title: "T", Variant: VariantDanger}))
	if !strings.Contains(out, "border-destructive") {
		t.Errorf("missing border-destructive in %s", out)
	}
}
