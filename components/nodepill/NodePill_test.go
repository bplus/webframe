package nodepill

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/badge"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := NodePill(p).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

var attrRe = regexp.MustCompile(`([a-zA-Z-]+)(?:="([^"]*)")?`)

// tag is one parsed opening tag: its name and attributes.
type tag struct {
	name  string
	attrs map[string]string
}

func (g tag) classes() map[string]bool {
	set := map[string]bool{}
	for _, f := range strings.Fields(g.attrs["class"]) {
		set[f] = true
	}
	return set
}

// openTags parses every opening tag in out, in document order.
func openTags(out string) []tag {
	var tags []tag
	for _, m := range regexp.MustCompile(`<([a-z]+)([^>]*)>`).FindAllStringSubmatch(out, -1) {
		g := tag{name: m[1], attrs: map[string]string{}}
		for _, a := range attrRe.FindAllStringSubmatch(m[2], -1) {
			g.attrs[a[1]] = a[2]
		}
		tags = append(tags, g)
	}
	return tags
}

// find returns the only tag carrying attribute marker.
func find(t *testing.T, out, marker string) tag {
	t.Helper()
	var hit []tag
	for _, g := range openTags(out) {
		if _, ok := g.attrs[marker]; ok {
			hit = append(hit, g)
		}
	}
	if len(hit) != 1 {
		t.Fatalf("want one tag with %s, got %d in %s", marker, len(hit), out)
	}
	return hit[0]
}

var allVariants = []badge.Variant{"", badge.VariantDefault, badge.VariantSecondary, badge.VariantDestructive, badge.VariantOutline, badge.VariantSuccess, badge.VariantWarning, badge.VariantInfo}

// TestVariantPicksTheTimelineDot checks the dot carries every class
// badge.DotClass gives the same variant, so a pill and a timeline dot of one
// variant always match.
func TestVariantPicksTheTimelineDot(t *testing.T) {
	for _, v := range allVariants {
		t.Run(string(v), func(t *testing.T) {
			got := find(t, render(t, Props{Title: "x", Variant: v}), "data-nodepill-dot").classes()
			for _, c := range strings.Fields(badge.DotClass(v)) {
				if !got[c] {
					t.Errorf("dot for %q lacks %q: %v", v, c, got)
				}
			}
		})
	}
}

// TestVariantsAreDistinct guards against the variants collapsing to one dot.
// The zero value and VariantDefault are the same primary dot by design.
func TestVariantsAreDistinct(t *testing.T) {
	seen := map[string]badge.Variant{}
	for _, v := range allVariants[1:] {
		c := find(t, render(t, Props{Title: "x", Variant: v}), "data-nodepill-dot").attrs["class"]
		if prev, dup := seen[c]; dup {
			t.Errorf("variants %q and %q render the same dot %q", prev, v, c)
		}
		seen[c] = v
	}
}

// TestTitleWrapsByDefault checks the default title is never cut: the full
// text is rendered, the title may wrap anywhere and does not truncate, and
// the root aligns items to the start so the dot sits beside line one.
func TestTitleWrapsByDefault(t *testing.T) {
	long := strings.Repeat("a long title that goes on ", 4) + "unbroken_identifier_with_no_spaces_at_all"
	out := render(t, Props{Title: long})
	if !strings.Contains(out, ">"+long+"</span>") {
		t.Errorf("the full title is not rendered: %s", out)
	}
	title := find(t, out, "data-nodepill-title")
	tc := title.classes()
	if !tc["wrap-anywhere"] || tc["truncate"] {
		t.Errorf("default title should wrap, not truncate: %v", tc)
	}
	if _, ok := title.attrs["title"]; ok {
		t.Error("a wrapping title needs no tooltip")
	}
	root := openTags(out)[0]
	rc := root.classes()
	if root.name != "span" || !rc["items-start"] || rc["items-center"] || !rc["max-w-full"] {
		t.Errorf("root should be a span capped at its container with the dot on line one: %s %v", root.name, rc)
	}
}

// TestTruncateKeepsOneLine checks Truncate swaps wrapping for an ellipsis
// with a width cap and puts the full text in a tooltip.
func TestTruncateKeepsOneLine(t *testing.T) {
	long := strings.Repeat("word ", 30)
	out := render(t, Props{Title: long, Truncate: true})
	title := find(t, out, "data-nodepill-title")
	tc := title.classes()
	if !tc["truncate"] || tc["wrap-anywhere"] {
		t.Errorf("truncated title classes = %v", tc)
	}
	if title.attrs["title"] != long {
		t.Errorf("tooltip = %q, want the full title", title.attrs["title"])
	}
	if !openTags(out)[0].classes()["max-w-[min(100%,15rem)]"] {
		t.Errorf("truncated root lacks its width cap: %s", out)
	}
}

func TestHrefMakesALink(t *testing.T) {
	out := render(t, Props{Title: "n", Href: "/nodes/1", Variant: badge.VariantInfo})
	root := openTags(out)[0]
	if root.name != "a" || root.attrs["href"] != "/nodes/1" || !root.classes()["cursor-pointer"] {
		t.Errorf("want an <a> root with the href and hover affordance: %s", out)
	}
	find(t, out, "data-nodepill-dot")
	find(t, out, "data-nodepill-title")
	if plain := render(t, Props{Title: "n"}); strings.Contains(plain, "cursor-pointer") {
		t.Errorf("non-link pill has the link affordance: %s", plain)
	}
}

func TestNodePillHrefIsSanitized(t *testing.T) {
	if out := render(t, Props{Title: "n", Href: "javascript:alert(1)"}); !strings.Contains(out, "about:invalid") || strings.Contains(out, "javascript:") {
		t.Errorf("unsafe href not sanitized: %s", out)
	}
	if out := render(t, Props{Title: "n", Href: "https://example.com/a"}); !strings.Contains(out, `href="https://example.com/a"`) {
		t.Errorf("safe href altered: %s", out)
	}
}
