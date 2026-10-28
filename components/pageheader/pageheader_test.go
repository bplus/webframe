package pageheader

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/breadcrumbs"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := PageHeader(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestPageHeader_TitleOnly(t *testing.T) {
	out := render(t, Props{Title: "Invoices"})
	if !strings.Contains(out, "<h1") || !strings.Contains(out, "Invoices") {
		t.Errorf("missing h1 title: %s", out)
	}
	if strings.Contains(out, "<a") {
		t.Errorf("unexpected <a in title-only header: %s", out)
	}
}

func TestPageHeader_AllParts(t *testing.T) {
	out := render(t, Props{
		Title:       "Invoice 42",
		Subtitle:    "Due Friday",
		Breadcrumbs: []breadcrumbs.Item{{Label: "Billing", Href: "/billing"}, {Label: "Invoice 42"}},
		Back:        &Link{Label: "All invoices", Href: "/invoices"},
		Status:      &badge.Props{Variant: badge.VariantSecondary},
		StatusText:  "Overdue",
		Secondary:   []action.Action{{Label: "Export", Href: "/export"}},
		Primary:     &action.Action{Label: "Pay now", Href: "/pay", Variant: action.VariantPrimary},
	})
	for _, want := range []string{"Billing", "← All invoices", "Overdue", "Due Friday", "Export", "Pay now"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Index(out, "Pay now") < strings.Index(out, "Export") {
		t.Errorf("primary must appear after secondary: %s", out)
	}
}

func TestPageHeader_BackHrefSanitized(t *testing.T) {
	out := render(t, Props{Title: "X", Back: &Link{Label: "Back", Href: "javascript:alert(1)"}})
	if strings.Contains(out, "javascript:") {
		t.Errorf("javascript: URL not sanitized: %s", out)
	}
}

func TestPageHeader_ActionsWrapperCanShrink(t *testing.T) {
	out := render(t, Props{Title: "T", Primary: &action.Action{Label: "A"}})
	if strings.Contains(out, `<div class="shrink-0">`) {
		t.Errorf("actions wrapper must not be shrink-0: %s", out)
	}
	if !strings.Contains(out, `<div class="min-w-0 max-w-full">`) {
		t.Errorf("actions wrapper missing min-w-0 max-w-full: %s", out)
	}
}
