package stattile

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/statcomparison"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestStatTile_Basics(t *testing.T) {
	out := render(t, StatTile(Props{
		ID:      "items",
		Label:   "Items",
		Value:   "14",
		Sub:     "11 published · 3 drafts",
		Caption: "Everything in the catalog, drafts included.",
	}))
	for _, want := range []string{
		`id="items"`,
		"Items", "14", "11 published · 3 drafts", "Everything in the catalog, drafts included.",
		"uppercase", "font-mono", "tabular-nums", "bg-card", "p-4", "min-h-24",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if !strings.HasPrefix(out, "<div") {
		t.Errorf("root should be <div without Href, got %s", out)
	}
	for _, unwanted := range []string{"<a ", "<svg", "border-destructive"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q in %s", unwanted, out)
		}
	}
}

func TestStatTile_Spark(t *testing.T) {
	out := render(t, StatTile(Props{Label: "Builds", Value: "9", Spark: []float64{1, 4, 2, 8}}))
	for _, want := range []string{"<svg", "<polyline", "<polygon", `width="110"`, `height="30"`, "var(--color-primary)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestStatTile_Ring(t *testing.T) {
	out := render(t, StatTile(Props{Label: "Coverage", Value: "72%", Ring: &Ratio{Value: 72, Label: "72"}}))
	for _, want := range []string{"stroke-dasharray", "width: 56px; height: 56px", ">72</span>"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.Contains(out, "<polyline") {
		t.Errorf("ring-only tile must not render a sparkline: %s", out)
	}

	both := render(t, StatTile(Props{Label: "Coverage", Value: "72%", Ring: &Ratio{Value: 72}, Spark: []float64{1, 2, 3}}))
	if !strings.Contains(both, "<polyline") {
		t.Errorf("Spark should win when both set, missing sparkline in %s", both)
	}
	if strings.Contains(both, "stroke-dasharray") {
		t.Errorf("Spark should win when both set, ring still rendered in %s", both)
	}
}

func TestStatTile_Href(t *testing.T) {
	out := render(t, StatTile(Props{Label: "Items", Value: "14", Href: "/items?sort=name&dir=asc"}))
	if !strings.HasPrefix(out, `<a href="/items?sort=name&amp;dir=asc"`) {
		t.Errorf("root should be <a href=...>, got %s", out)
	}
	for _, want := range []string{"hover:border-primary", "</a>"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if strings.HasPrefix(out, "<div") {
		t.Errorf("root should not be <div when Href set: %s", out)
	}
}

func TestStatTile_Delta(t *testing.T) {
	up := render(t, StatTile(Props{Label: "Items", Value: "14", Delta: &Delta{Value: "+12%", Direction: statcomparison.DirectionUp}}))
	for _, want := range []string{"+12%", "text-green-500", "M8 3v10M8 3l4 4M8 3L4 7"} {
		if !strings.Contains(up, want) {
			t.Errorf("up: missing %q in %s", want, up)
		}
	}
	down := render(t, StatTile(Props{Label: "Items", Value: "14", Delta: &Delta{Value: "-3", Direction: statcomparison.DirectionDown}}))
	for _, want := range []string{"-3", "text-red-500", "M8 13V3M8 13l4-4M8 13L4 9"} {
		if !strings.Contains(down, want) {
			t.Errorf("down: missing %q in %s", want, down)
		}
	}
	flat := render(t, StatTile(Props{Label: "Items", Value: "14", Delta: &Delta{Value: "0"}}))
	for _, want := range []string{"text-muted-foreground", "M2 8h12"} {
		if !strings.Contains(flat, want) {
			t.Errorf("flat: missing %q in %s", want, flat)
		}
	}
	none := render(t, StatTile(Props{Label: "Items", Value: "14"}))
	if strings.Contains(none, "text-green-500") || strings.Contains(none, "text-red-500") {
		t.Errorf("no Delta should render no delta row: %s", none)
	}
}

func TestStatTile_Variants(t *testing.T) {
	danger := render(t, StatTile(Props{Label: "Errors", Value: "3", Variant: VariantDanger}))
	if !strings.Contains(danger, "border-destructive") {
		t.Errorf("danger: missing border-destructive in %s", danger)
	}
	muted := render(t, StatTile(Props{Label: "Archived", Value: "3", Variant: VariantMuted}))
	if !strings.Contains(muted, "bg-muted") {
		t.Errorf("muted: missing bg-muted in %s", muted)
	}
	if strings.Contains(muted, "bg-card") {
		t.Errorf("muted: should not keep bg-card in %s", muted)
	}
}

func TestStatTile_AttributesAndClass(t *testing.T) {
	out := render(t, StatTile(Props{
		Label:      "Items",
		Value:      "14",
		Class:      "w-48",
		Attributes: templ.Attributes{"data-tile": "items"},
	}))
	for _, want := range []string{"w-48", `data-tile="items"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestStatTile_ZeroValue(t *testing.T) {
	out := render(t, StatTile())
	if !strings.HasPrefix(out, "<div") {
		t.Errorf("zero-value tile should still render a <div root: %s", out)
	}
}
