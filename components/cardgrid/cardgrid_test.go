package cardgrid

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/aspectratio"
	"github.com/bplus/webframe/components/badge"
	"github.com/bplus/webframe/components/emptystate"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := CardGrid(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func wantAll(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func wantNone(t *testing.T, out string, bads ...string) {
	t.Helper()
	for _, b := range bads {
		if strings.Contains(out, b) {
			t.Errorf("unexpected %q in:\n%s", b, out)
		}
	}
}

// hasClass reports whether cls appears as a whole class token (so "p-4"
// does not match inside "gap-4").
func hasClass(out, cls string) bool {
	return regexp.MustCompile(`(^|[\s"])` + regexp.QuoteMeta(cls) + `([\s"]|$)`).MatchString(out)
}

func wantClasses(t *testing.T, out string, want bool, classes ...string) {
	t.Helper()
	for _, c := range classes {
		if hasClass(out, c) != want {
			t.Errorf("class %q present=%v, want %v in:\n%s", c, !want, want, out)
		}
	}
}

func one() []Card {
	return []Card{{Title: "Alpha", Href: "/items/1"}}
}

func TestColumns(t *testing.T) {
	cases := []struct {
		columns  int
		want     []string
		dontWant []string
	}{
		{0, []string{"grid-cols-1", "sm:grid-cols-2", "lg:grid-cols-3", "xl:grid-cols-4"}, nil},
		{4, []string{"grid-cols-1", "sm:grid-cols-2", "lg:grid-cols-3", "xl:grid-cols-4"}, nil},
		{3, []string{"grid-cols-1", "sm:grid-cols-2", "lg:grid-cols-3"}, []string{"xl:grid-cols-4"}},
		{2, []string{"grid-cols-1", "sm:grid-cols-2"}, []string{"lg:grid-cols-3", "xl:grid-cols-4"}},
		{7, []string{"xl:grid-cols-4"}, nil}, // out of range falls back to 4
	}
	for _, c := range cases {
		out := render(t, Props{Items: one(), Columns: c.columns})
		wantAll(t, out, c.want...)
		wantNone(t, out, c.dontWant...)
		wantAll(t, out, "gap-4")
	}
}

func TestRootIDClassAttributes(t *testing.T) {
	out := render(t, Props{ID: "catalog", Class: "mt-8", Attributes: templ.Attributes{"data-x": "1"}, Items: one()})
	wantAll(t, out, `id="catalog"`, "mt-8", `data-x="1"`, "grid")
}

// The title link is the card's one focusable element for navigation: the
// title is wrapped exactly once, with the whole-card overlay classes, and
// a card without Href has no anchor at all.
func TestTitleLinkOnce(t *testing.T) {
	out := render(t, Props{Items: []Card{{ID: "c1", Title: "Alpha", Subtitle: "First", Href: "/items/1"}}})
	if n := strings.Count(out, "<a "); n != 1 {
		t.Fatalf("want exactly one <a>, got %d in:\n%s", n, out)
	}
	wantAll(t, out,
		`href="/items/1"`, "after:absolute", "after:inset-0",
		`id="c1"`, "relative", "<h3", ">Alpha</a>", "First", "text-muted-foreground",
	)

	plain := render(t, Props{Items: []Card{{Title: "Alpha"}}})
	wantNone(t, plain, "<a ", "after:absolute")
	wantAll(t, plain, "Alpha")
}

// A card without Art renders no frame or picture; one with Art renders
// the aspect frame (16:9 by default) and a cropped picture.
func TestArt(t *testing.T) {
	out := render(t, Props{Items: []Card{{Title: "No art"}}})
	wantNone(t, out, "aspect-", "<img")

	out = render(t, Props{Items: []Card{{Title: "Art", Art: &Art{Src: "/a.png", Alt: "A"}}}})
	wantAll(t, out, "aspect-video", `src="/a.png"`, `alt="A"`, "object-cover", "h-full", "w-full")

	out = render(t, Props{Items: []Card{{Title: "Art", Art: &Art{Src: "/a.png", Ratio: aspectratio.RatioSquare}}}})
	wantAll(t, out, "aspect-square", `alt=""`)
	wantNone(t, out, "aspect-video")
}

func TestEmpty(t *testing.T) {
	out := render(t, Props{})
	wantAll(t, out, "col-span-full", "Nothing here yet")
	wantNone(t, out, "bg-card")

	out = render(t, Props{Empty: emptystate.Props{Title: "No mods yet", Description: "Publish one to see it here."}})
	wantAll(t, out, "No mods yet", "Publish one to see it here.")
	wantNone(t, out, "Nothing here yet")
}

func TestBadgesAndMetaEscaped(t *testing.T) {
	out := render(t, Props{Items: []Card{{
		Title:  "<b>T</b>",
		Badges: []Badge{{Label: "<i>Draft</i>", Props: badge.Props{Variant: badge.VariantSecondary}}, StatusBadge("", "published")},
		Meta:   []string{"<script>x</script>", "3 files"},
	}}})
	wantNone(t, out, "<b>T</b>", "<i>Draft</i>", "<script>x</script>")
	wantAll(t, out,
		"&lt;b&gt;T&lt;/b&gt;", "&lt;i&gt;Draft&lt;/i&gt;", "&lt;script&gt;x&lt;/script&gt; · 3 files",
		"bg-secondary", "Published", "text-success",
	)
}

func TestFooterActions(t *testing.T) {
	out := render(t, Props{Items: []Card{{Title: "A", Href: "/a", Actions: []action.Action{
		{Label: "Open", Href: "/a/open"},
		{Label: "Delete", HxDelete: "/a", Variant: action.VariantDestructive},
	}}}})
	wantAll(t, out, "z-10", "flex-wrap", "pt-0", `href="/a/open"`, "Open", `hx-delete="/a"`, "Delete", "bg-destructive")
	// the plain action got the ghost treatment and the small size
	wantAll(t, out, "hover:bg-accent", "h-8")
	if n := strings.Count(out, "<a "); n != 2 {
		t.Fatalf("want title link + one action link, got %d anchors in:\n%s", n, out)
	}

	noFooter := render(t, Props{Items: one()})
	wantNone(t, noFooter, "z-10", "pt-0")

	footer := render(t, Props{Items: []Card{{Title: "A", Footer: templ.Raw(`<span data-f="1">by Acme</span>`)}}})
	wantAll(t, footer, `data-f="1"`, "by Acme", "z-10")
}

func TestDense(t *testing.T) {
	out := render(t, Props{Items: one(), Dense: true})
	wantClasses(t, out, true, "gap-3", "p-4")
	wantClasses(t, out, false, "gap-4", "p-6")

	out = render(t, Props{Items: one()})
	wantClasses(t, out, true, "gap-4", "p-6")
	wantClasses(t, out, false, "gap-3", "p-4")
}

func TestStatusBadge(t *testing.T) {
	b := StatusBadge("", "in-progress")
	if b.Label != "In progress" || b.Props.Variant != badge.VariantInfo {
		t.Fatalf("StatusBadge = %+v", b)
	}
	if StatusBadge("Custom", "failed").Label != "Custom" {
		t.Fatal("explicit label should win")
	}
}
