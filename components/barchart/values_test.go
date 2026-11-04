package barchart

import (
	"regexp"
	"strconv"
	"testing"
)

// valueText matches a value label: the only <text> drawn in fill-foreground.
var valueText = regexp.MustCompile(`<text x="(-?\d+)" y="(-?\d+)"(?: text-anchor="middle")? class="text-xs fill-foreground"[^>]*>([^<]*)</text>`)

var viewBox = regexp.MustCompile(`viewBox="0 0 (\d+) (\d+)"`)

// 6px per glyph is the width of a 10px value digit.

// valueLabels returns the x, y and text of every value label plus the
// viewBox width and height.
func valueLabels(t *testing.T, html string) (labels [][3]string, w, h int) {
	t.Helper()
	vb := viewBox.FindStringSubmatch(html)
	if vb == nil {
		t.Fatalf("no viewBox in %s", html)
	}
	w, _ = strconv.Atoi(vb[1])
	h, _ = strconv.Atoi(vb[2])
	for _, m := range valueText.FindAllStringSubmatch(html, -1) {
		labels = append(labels, [3]string{m[1], m[2], m[3]})
	}
	return labels, w, h
}

// The largest bar's value label must fit inside the SVG: a horizontal
// label starts after the bar end and needs room for its text before the
// right edge.
func TestShowValues_HorizontalLargestLabelFits(t *testing.T) {
	html := render(t, Props{ShowValues: true, ShowLabels: true, Bars: []Bar{
		{Label: "Weapon", Value: 42}, {Label: "Armor", Value: 31}, {Label: "Ring", Value: 1250},
	}})
	labels, w, _ := valueLabels(t, html)
	if len(labels) != 3 {
		t.Fatalf("want 3 value labels, got %d in %s", len(labels), html)
	}
	for _, l := range labels {
		x, _ := strconv.Atoi(l[0])
		end := x + int(float64(len(l[2]))*6+0.5)
		if end > w {
			t.Errorf("value %q runs from x=%d to %d, past the SVG width %d", l[2], x, end, w)
		}
	}
}

// A vertical chart's tallest bar has its value above the bar; the text
// baseline must sit low enough that the 10px glyphs are inside the SVG.
func TestShowValues_VerticalTopLabelFits(t *testing.T) {
	html := render(t, Props{ShowValues: true, ShowLabels: true, Orientation: Vertical, Bars: []Bar{
		{Label: "Jan", Value: 240}, {Label: "Feb", Value: 190},
	}})
	labels, _, _ := valueLabels(t, html)
	if len(labels) != 2 {
		t.Fatalf("want 2 value labels, got %d in %s", len(labels), html)
	}
	for _, l := range labels {
		y, _ := strconv.Atoi(l[1])
		if y < 10 {
			t.Errorf("value %q has baseline y=%d, so its top is cut off by the SVG edge", l[2], y)
		}
	}
}
