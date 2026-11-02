package timeline

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/emptystate"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Timeline(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

var (
	utc    = time.UTC
	denver = mustZone("America/Denver")
	tokyo  = mustZone("Asia/Tokyo")
)

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func TestTimeline_GroupsByDayNewestFirst(t *testing.T) {
	items := []Item{
		{At: time.Date(2026, 10, 29, 9, 0, 0, 0, utc), Title: "Oldest"},
		{At: time.Date(2026, 10, 31, 14, 5, 0, 0, utc), Title: "Newest"},
		{At: time.Date(2026, 10, 31, 8, 30, 0, 0, utc), Title: "Same day earlier"},
	}
	out := render(t, Props{ID: "tl", Items: items})

	if !strings.Contains(out, `<ol id="tl"`) {
		t.Errorf("missing root ol with id: %s", out)
	}
	for _, want := range []string{
		`<time datetime="2026-10-31"`, "Saturday, October 31, 2026",
		`<time datetime="2026-10-29"`, "Thursday, October 29, 2026",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if got := strings.Count(out, `<time datetime="2026-10-31"`) + strings.Count(out, `<time datetime="2026-10-29"`); got != 2 {
		t.Errorf("want 2 day headings, got %d in %s", got, out)
	}
	// Newest day first, newest item first within the day, oldest day last.
	iNewest, iSame, iOldest := strings.Index(out, "Newest"), strings.Index(out, "Same day earlier"), strings.Index(out, "Oldest")
	if !(iNewest < iSame && iSame < iOldest) {
		t.Errorf("order wrong (newest=%d same=%d oldest=%d): %s", iNewest, iSame, iOldest, out)
	}
	// The day heading precedes its items.
	if strings.Index(out, `datetime="2026-10-31"`) > iNewest {
		t.Errorf("day heading after its item: %s", out)
	}
	// Input order is not the display order.
	groups := Props{Items: items}.Groups()
	if len(groups) != 2 || len(groups[0].Items) != 2 || groups[0].Items[0].Title != "Newest" || groups[1].Items[0].Title != "Oldest" {
		t.Errorf("Groups() = %+v", groups)
	}
}

func TestTimeline_OldestFirst(t *testing.T) {
	items := []Item{
		{At: time.Date(2026, 10, 31, 14, 5, 0, 0, utc), Title: "Newest"},
		{At: time.Date(2026, 10, 29, 9, 0, 0, 0, utc), Title: "Oldest"},
	}
	out := render(t, Props{Items: items, OldestFirst: true})
	if strings.Index(out, "Oldest") > strings.Index(out, "Newest") {
		t.Errorf("OldestFirst did not flip the order: %s", out)
	}
	if strings.Index(out, `datetime="2026-10-29"`) > strings.Index(out, `datetime="2026-10-31"`) {
		t.Errorf("OldestFirst did not flip the day groups: %s", out)
	}
}

func TestTimeline_GroupsByItemOwnLocation(t *testing.T) {
	// 2026-10-31 23:30 in Denver is 2026-11-01 05:30 UTC; with no
	// Props.Location the item's own zone decides the day and the clock.
	items := []Item{{At: time.Date(2026, 10, 31, 23, 30, 0, 0, denver), Title: "Late"}}
	out := render(t, Props{Items: items})
	if !strings.Contains(out, `<time datetime="2026-10-31"`) {
		t.Errorf("want the item's own day 2026-10-31: %s", out)
	}
	if !strings.Contains(out, `datetime="2026-10-31T23:30:00-06:00"`) {
		t.Errorf("want RFC3339 in the item's own zone: %s", out)
	}
	if !strings.Contains(out, ">23:30</time>") {
		t.Errorf("want HH:MM in the item's own zone: %s", out)
	}
}

func TestTimeline_GroupsByPropsLocation(t *testing.T) {
	// The same instant seen from Tokyo is 2026-11-01 14:30.
	items := []Item{{At: time.Date(2026, 10, 31, 23, 30, 0, 0, denver), Title: "Late"}}
	out := render(t, Props{Items: items, Location: tokyo})
	if !strings.Contains(out, `<time datetime="2026-11-01"`) {
		t.Errorf("want the day in Props.Location: %s", out)
	}
	if strings.Contains(out, `datetime="2026-10-31"`) {
		t.Errorf("item's own day leaked despite Props.Location: %s", out)
	}
	if !strings.Contains(out, `datetime="2026-11-01T14:30:00+09:00"`) {
		t.Errorf("want RFC3339 in Props.Location: %s", out)
	}
	if !strings.Contains(out, ">14:30</time>") {
		t.Errorf("want HH:MM in Props.Location: %s", out)
	}
	// Two items that share a UTC day but straddle midnight in Tokyo land in
	// two groups.
	straddle := []Item{
		{At: time.Date(2026, 10, 31, 14, 0, 0, 0, utc), Title: "A"}, // 23:00 Tokyo, Oct 31
		{At: time.Date(2026, 10, 31, 16, 0, 0, 0, utc), Title: "B"}, // 01:00 Tokyo, Nov 1
	}
	if got := len((Props{Items: straddle, Location: tokyo}).Groups()); got != 2 {
		t.Errorf("want 2 groups across Tokyo midnight, got %d", got)
	}
	if got := len((Props{Items: straddle}).Groups()); got != 1 {
		t.Errorf("want 1 group in UTC, got %d", got)
	}
}

func TestTimeline_NoGroups(t *testing.T) {
	items := []Item{
		{At: time.Date(2026, 10, 29, 9, 0, 0, 0, utc), Title: "Oldest"},
		{At: time.Date(2026, 10, 31, 14, 5, 0, 0, utc), Title: "Newest"},
	}
	out := render(t, Props{Items: items, NoGroups: true})
	if strings.Contains(out, `datetime="2026-10-31"`) || strings.Contains(out, "October") {
		t.Errorf("day heading rendered with NoGroups: %s", out)
	}
	if strings.Index(out, "Newest") > strings.Index(out, "Oldest") {
		t.Errorf("NoGroups still sorts newest first: %s", out)
	}
	if got := strings.Count(out, "<li"); got != 3 { // one group li + two items
		t.Errorf("want 3 <li>, got %d in %s", got, out)
	}
}

func TestTimeline_ISOAttributes(t *testing.T) {
	at := time.Date(2026, 10, 31, 14, 5, 9, 0, utc)
	out := render(t, Props{Items: []Item{{At: at, Title: "T"}}})
	if !strings.Contains(out, `<time datetime="2026-10-31T14:05:09Z"`) {
		t.Errorf("missing RFC3339 datetime: %s", out)
	}
	if !strings.Contains(out, ">14:05</time>") {
		t.Errorf("missing HH:MM text: %s", out)
	}
	if !strings.Contains(out, `<time datetime="2026-10-31" `) {
		t.Errorf("missing ISO day datetime: %s", out)
	}
}

func TestTimeline_TonePerVariant(t *testing.T) {
	cases := []struct {
		v    badge.Variant
		want string
	}{
		{badge.VariantDefault, "bg-primary"},
		{"", "bg-primary"},
		{badge.VariantDestructive, "bg-destructive"},
		{badge.VariantSuccess, "bg-success"},
		{badge.VariantWarning, "bg-warning"},
		{badge.VariantInfo, "bg-info"},
		{badge.VariantSecondary, "bg-muted-foreground"},
		{badge.VariantOutline, "border-border"},
	}
	for _, c := range cases {
		if got := Tone(c.v); !strings.Contains(got, c.want) {
			t.Errorf("Tone(%q) = %q, want %q", c.v, got, c.want)
		}
		out := render(t, Props{Items: []Item{{At: time.Now(), Title: "x", Variant: c.v}}})
		if !strings.Contains(out, c.want) {
			t.Errorf("variant %q: dot class %q missing in %s", c.v, c.want, out)
		}
		if !strings.Contains(out, "rounded-full") {
			t.Errorf("variant %q: no dot rendered: %s", c.v, out)
		}
	}
	// Each variant's token appears on exactly one dot in a one-item list.
	out := render(t, Props{Items: []Item{{At: time.Now(), Title: "x", Variant: badge.VariantSuccess}}})
	if strings.Contains(out, "bg-primary") {
		t.Errorf("default tone leaked onto a success item: %s", out)
	}
}

func TestTimeline_IconReplacesDot(t *testing.T) {
	ic := templ.Raw(`<svg data-test="glyph"></svg>`)
	out := render(t, Props{Items: []Item{{At: time.Now(), Title: "x", Icon: ic, Variant: badge.VariantWarning}}})
	if !strings.Contains(out, `data-test="glyph"`) {
		t.Errorf("icon not rendered: %s", out)
	}
	if strings.Contains(out, "rounded-full") {
		t.Errorf("dot rendered alongside the icon: %s", out)
	}
	if !strings.Contains(out, "text-warning") {
		t.Errorf("icon not toned by the variant: %s", out)
	}
}

func TestTimeline_LinkBodyMeta(t *testing.T) {
	out := render(t, Props{Items: []Item{{
		At: time.Now(), Title: "Deployed", Href: "/deploys/9", Body: "Build 42 went live.", Meta: "by ada",
	}}})
	if !strings.Contains(out, `<a href="/deploys/9"`) || !strings.Contains(out, ">Deployed</a>") {
		t.Errorf("title not a link: %s", out)
	}
	if got := strings.Count(out, "<a "); got != 1 {
		t.Errorf("want exactly one link, got %d: %s", got, out)
	}
	for _, want := range []string{"Build 42 went live.", "by ada", "text-muted-foreground"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q: %s", want, out)
		}
	}
	plain := render(t, Props{Items: []Item{{At: time.Now(), Title: "Plain"}}})
	if strings.Contains(plain, "<a ") {
		t.Errorf("link rendered without Href: %s", plain)
	}
	if !strings.Contains(plain, ">Plain</p>") {
		t.Errorf("title not rendered as text: %s", plain)
	}
}

func TestTimeline_UnsafeHrefSanitized(t *testing.T) {
	out := render(t, Props{Items: []Item{{At: time.Now(), Title: "x", Href: "javascript:alert(1)"}}})
	if strings.Contains(out, "javascript:alert(1)") {
		t.Errorf("unsanitized href: %s", out)
	}
}

func TestTimeline_Escaping(t *testing.T) {
	out := render(t, Props{Items: []Item{{
		At:    time.Now(),
		Title: `<script>alert("t")</script>`,
		Body:  `<b>body</b>`,
		Meta:  `<i>meta</i> & more`,
	}}})
	for _, bad := range []string{"<script>", "<b>", "<i>", "</script>"} {
		if strings.Contains(out, bad) {
			t.Errorf("unescaped %q in %s", bad, out)
		}
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;b&gt;body&lt;/b&gt;", "&amp; more"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing escaped %q in %s", want, out)
		}
	}
}

func TestTimeline_EmptyState(t *testing.T) {
	out := render(t, Props{ID: "tl", Class: "mt-4", Attributes: templ.Attributes{"data-x": "1"}})
	if strings.Contains(out, "<ol") || strings.Contains(out, "<li") {
		t.Errorf("list rendered with no items: %s", out)
	}
	for _, want := range []string{`id="tl"`, "mt-4", `data-x="1"`, "Nothing here yet"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	custom := render(t, Props{Empty: emptystate.Props{Title: "No activity", Description: "Changes show up here."}})
	for _, want := range []string{"No activity", "Changes show up here."} {
		if !strings.Contains(custom, want) {
			t.Errorf("missing %q in %s", want, custom)
		}
	}
	if strings.Contains(custom, "Nothing here yet") {
		t.Errorf("default title leaked over a custom Empty: %s", custom)
	}
}

func TestTimeline_RootAttributes(t *testing.T) {
	out := render(t, Props{
		ID: "tl", Class: "mt-4", Attributes: templ.Attributes{"data-x": "1"},
		Items: []Item{{At: time.Now(), Title: "x"}},
	})
	for _, want := range []string{`<ol id="tl"`, "mt-4", `data-x="1"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestTimeline_FixedGutterOnEveryRow(t *testing.T) {
	// Rail alignment relies on every row (heading and item) using the same
	// fixed-gutter grid.
	out := render(t, Props{Items: []Item{
		{At: time.Date(2026, 10, 31, 1, 0, 0, 0, utc), Title: "a"},
		{At: time.Date(2026, 10, 30, 1, 0, 0, 0, utc), Title: "b"},
	}})
	if got := strings.Count(out, "grid-cols-[1rem_minmax(0,1fr)]"); got != 4 { // 2 headings + 2 items
		t.Errorf("want 4 fixed-gutter rows, got %d in %s", got, out)
	}
	// The rail runs between rows but not below the last item: the first
	// heading has no rail above it; item a, heading b carry one; item b
	// (last) does not.
	if got := strings.Count(out, "bg-border"); got != 2 {
		t.Errorf("want 2 rail segments, got %d in %s", got, out)
	}
	if strings.Contains(out, "float") || strings.Contains(out, "absolute") {
		t.Errorf("layout uses floats or absolute positioning: %s", out)
	}
}

func TestTimeline_ZeroPropsRendersEmptyState(t *testing.T) {
	var buf bytes.Buffer
	if err := Timeline().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "Nothing here yet") {
		t.Errorf("zero props did not render the empty state: %s", buf.String())
	}
}
