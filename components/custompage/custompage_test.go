package custompage

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

func TestCustomPage_EmitsKindAndReason(t *testing.T) {
	out := render(t, CustomPage(Props{Reason: "landing page"}))
	for _, want := range []string{`data-wf-custom="page"`, `data-wf-reason="landing page"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "data-wf-missing-reason") {
		t.Errorf("unexpected missing-reason flag in %s", out)
	}
}

func TestCustomPage_MissingReasonFlagged(t *testing.T) {
	out := render(t, CustomPage())
	if !strings.Contains(out, `data-wf-missing-reason="true"`) {
		t.Errorf("expected missing-reason flag in %s", out)
	}
}

func TestCustomPage_RendersChildren(t *testing.T) {
	child := templ.Raw("<span>inner</span>")
	var buf bytes.Buffer
	ctx := templ.WithChildren(context.Background(), child)
	if err := CustomPage(Props{ID: "x", Reason: "r"}).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<span>inner</span>") || !strings.Contains(out, `id="x"`) {
		t.Errorf("children/id missing in %s", out)
	}
}
