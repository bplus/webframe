package barchart

import (
	"regexp"
	"strconv"
	"strings"
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

// axisText matches a vertical chart's axis label (centered, muted).
var axisText = regexp.MustCompile(`<text x="(-?\d+)" y="(-?\d+)" text-anchor="middle" class="text-xs fill-muted-foreground"[^>]*>([^<]*)</text>`)

// manyBars returns n bars labelled "Week 1".."Week n" with 3-digit values.
func manyBars(n int) []Bar {
	bars := make([]Bar, n)
	for i := range bars {
		bars[i] = Bar{Label: "Week " + strconv.Itoa(i+1), Value: float64(900 + i)}
	}
	return bars
}

// assertNoOverlap fails when two centered 10px labels share horizontal
// space or one runs past the SVG's left or right edge.
func assertNoOverlap(t *testing.T, kind string, labels [][3]string, w int) {
	t.Helper()
	prevEnd := -1.0
	for _, l := range labels {
		x, _ := strconv.Atoi(l[0])
		half := float64(len([]rune(l[2]))) * valueCharW / 2
		start, end := float64(x)-half, float64(x)+half
		if start < 0 || end > float64(w) {
			t.Errorf("%s label %q spans %.0f..%.0f, outside the SVG width %d", kind, l[2], start, end, w)
		}
		if start < prevEnd {
			t.Errorf("%s label %q starts at %.0f, before the previous label ends at %.0f", kind, l[2], start, prevEnd)
		}
		prevEnd = end
	}
}

// Narrow columns cannot hold their value labels side by side, so the chart
// leaves them out instead of drawing them on top of each other, and each
// bar carries its label and value in a hover <title>.
func TestShowValues_VerticalNarrowBarsDoNotOverlap(t *testing.T) {
	html := render(t, Props{ShowValues: true, Orientation: Vertical, Bars: manyBars(24)})
	labels, w, _ := valueLabels(t, html)
	assertNoOverlap(t, "value", labels, w)
	if !strings.Contains(html, "<title>Week 1: 900</title>") {
		t.Errorf("hidden value labels should move into a hover title per bar, got %s", html)
	}
}

// Wide columns keep every value label and need no hover title.
func TestShowValues_VerticalWideBarsKeepLabels(t *testing.T) {
	html := render(t, Props{ShowValues: true, ShowLabels: true, Orientation: Vertical, Bars: manyBars(4)})
	labels, w, _ := valueLabels(t, html)
	if len(labels) != 4 {
		t.Fatalf("want 4 value labels, got %d", len(labels))
	}
	assertNoOverlap(t, "value", labels, w)
	if strings.Contains(html, "<title>") {
		t.Errorf("no title expected when every label fits, got %s", html)
	}
}

// Axis labels under narrow columns are thinned to every k-th one, so the
// shown ones never collide or cross the SVG edge.
func TestShowLabels_VerticalNarrowBarsThinned(t *testing.T) {
	html := render(t, Props{ShowLabels: true, Orientation: Vertical, Bars: manyBars(24)})
	var labels [][3]string
	for _, m := range axisText.FindAllStringSubmatch(html, -1) {
		labels = append(labels, [3]string{m[1], m[2], m[3]})
	}
	_, w, _ := valueLabels(t, html)
	if len(labels) < 3 || len(labels) > 12 {
		t.Fatalf("want a thinned set of axis labels, got %v", labels)
	}
	assertNoOverlap(t, "axis", labels, w)
	if titles := strings.Count(html, "<title>Week "); titles+len(labels) != 24 {
		t.Errorf("each of the 24 bars should show its label or carry it in a hover title: %d labels, %d titles", len(labels), titles)
	}
}
