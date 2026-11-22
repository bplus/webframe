package commandpalette

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/icon"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func sample() Props {
	return Props{Items: []Item{
		{Label: "Home", Href: "/", Keywords: "start dashboard", Icon: icon.House()},
		{Label: "New movie", Href: "/movies/new", Group: "Movies", Shortcut: "N"},
		{Label: "All movies", Href: "/movies", Group: "Movies"},
		{Label: "Settings", Href: "/settings", Group: "Account"},
	}}
}

func TestPalette_DialogIDAndDefaults(t *testing.T) {
	got := render(t, Palette(sample()))
	for _, want := range []string{
		`id="wf-palette"`,
		`data-dialog-instance="wf-palette"`,
		`data-tui-dialog-hidden="true"`,
		`data-wf-palette=""`,
		`data-wf-palette-shortcut="mod+k"`,
		`placeholder="` + DefaultPlaceholder + `"`,
		`>` + DefaultEmpty + `</p>`,
		`data-wf-palette-empty`,
		`role="combobox"`,
		`aria-controls="wf-palette-list"`,
		`id="wf-palette-list"`,
		`role="listbox"`,
		`x-model="q"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `aria-label="Close"`) {
		t.Errorf("templUI X button should be hidden:\n%s", got)
	}
}

func TestPalette_CustomIDAndTexts(t *testing.T) {
	got := render(t, Palette(Props{ID: "jump", Placeholder: "Go to…", Empty: "Nothing here", Shortcut: "ctrl+shift+p"}))
	for _, want := range []string{`id="jump"`, `id="jump-input"`, `id="jump-list"`, `placeholder="Go to…"`, `>Nothing here</p>`, `data-wf-palette-shortcut="ctrl+shift+p"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestPalette_ItemsGroupsKeywordsShortcut(t *testing.T) {
	got := render(t, Palette(sample()))
	// Three groups: the untitled one, Movies, Account, in first-appearance order.
	if n := strings.Count(got, `role="group"`); n != 3 {
		t.Errorf("want 3 groups, got %d in:\n%s", n, got)
	}
	movies := strings.Index(got, ">Movies</div>")
	account := strings.Index(got, ">Account</div>")
	home := strings.Index(got, ">Home</span>")
	if home < 0 || movies < 0 || account < 0 || !(home < movies && movies < account) {
		t.Errorf("want Home < Movies heading < Account heading; got %d %d %d", home, movies, account)
	}
	if n := strings.Count(got, `role="option"`); n != 4 {
		t.Errorf("want 4 options, got %d in:\n%s", n, got)
	}
	for _, want := range []string{
		`data-wf-palette-item data-label="Home" data-keywords="start dashboard"`,
		`href="/movies/new"`,
		`id="wf-palette-option-0"`,
		`id="wf-palette-option-3"`,
		`aria-selected="false"`,
		`tabindex="-1"`,
		`<kbd class="ml-auto shrink-0 rounded border bg-muted px-1.5 py-0.5 font-sans text-xs text-muted-foreground">N</kbd>`,
		`aria-labelledby="wf-palette-group-1"`,
		`id="wf-palette-group-1" role="presentation"`,
		`<svg`, // the Home icon
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// The untitled group has no heading and no aria-labelledby.
	if strings.Contains(got, `aria-labelledby="wf-palette-group-0"`) || strings.Contains(got, `id="wf-palette-group-0"`) {
		t.Errorf("untitled group should have no heading:\n%s", got)
	}
}

func TestPalette_EscapesLabelsAndKeywords(t *testing.T) {
	got := render(t, Palette(Props{Items: []Item{{Label: `<b>x</b>`, Href: "/x", Keywords: `"><script>`}}}))
	if strings.Contains(got, "<b>x</b>") || strings.Contains(got, "<script>") {
		t.Errorf("label/keywords not escaped:\n%s", got)
	}
	for _, want := range []string{`data-label="&lt;b&gt;x&lt;/b&gt;"`, `data-keywords="&#34;&gt;&lt;script&gt;"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestPalette_PhoneTopSheetAndDesktopPanel(t *testing.T) {
	got := render(t, Palette(sample()))
	for _, want := range []string{"left-0", "right-0", "top-[var(--wf-palette-top,0px)]", "rounded-b-xl", "max-h-[var(--wf-palette-vh,60dvh)]", "sm:max-w-lg", "sm:left-[50%]", "sm:top-[50%]", "sm:rounded-lg", "sm:max-h-[60vh]", "overflow-y-auto"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// The dialog's own centering classes are overridden below sm, not duplicated.
	if strings.Contains(got, ` left-[50%]`) || strings.Contains(got, ` translate-x-[-50%]`) {
		t.Errorf("unprefixed centering class survived the merge:\n%s", got)
	}
}

func TestPalette_AlpineReadsTheDOM(t *testing.T) {
	got := render(t, Palette(sample()))
	for _, want := range []string{`x-data="{`, "window.wf.palette", "window.tui", "dataset.label", "dataset.keywords", "ArrowDown", "ArrowUp", "Home", "End", "Enter", "Escape", `x-on:keydown="key($event)"`, `x-on:pointermove="hover($event)"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// No item text is baked into the script: the x-data is identical for any items.
	other := render(t, Palette(Props{Items: []Item{{Label: "Zebra", Href: "/z"}}}))
	if xd(got) != xd(other) {
		t.Errorf("x-data differs between item sets:\n%s\n---\n%s", xd(got), xd(other))
	}
}

// xd extracts the x-data attribute value of the dialog root.
func xd(html string) string {
	i := strings.Index(html, `x-data="`)
	if i < 0 {
		return ""
	}
	rest := html[i+len(`x-data="`):]
	return rest[:strings.Index(rest, `"`)]
}

func TestTrigger(t *testing.T) {
	got := render(t, Trigger(Props{}))
	for _, want := range []string{
		"<button", `type="button"`,
		`data-wf-palette-trigger="wf-palette"`,
		`aria-controls="wf-palette"`,
		`aria-keyshortcuts="Meta+K Control+K"`,
		`aria-label="Search"`,
		`<span class="hidden md:inline">Search</span>`,
		`>⌘K</kbd>`,
		`data-wf-hint-mac="⌘K"`,
		`data-wf-hint-other="Ctrl+K"`,
		`? $el.dataset.wfHintMac : $el.dataset.wfHintOther`,
		"window.wf.palette.open($el.dataset.wfPaletteTrigger)",
		`<svg`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	got = render(t, Trigger(Props{ID: "jump", Shortcut: "ctrl+shift+p"}))
	for _, want := range []string{`data-wf-palette-trigger="jump"`, `aria-keyshortcuts="Control+Shift+P"`, `>Ctrl+⇧P</kbd>`, `data-wf-hint-other="Ctrl+Shift+P"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestHint(t *testing.T) {
	for in, want := range map[string]string{"mod+k": "⌘K", "ctrl+k": "Ctrl+K", "shift+alt+p": "⇧⌥P", "k": "K"} {
		if got := Hint(in); got != want {
			t.Errorf("Hint(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHintFor_OtherPlatforms(t *testing.T) {
	for in, want := range map[string]string{"mod+k": "Ctrl+K", "ctrl+k": "Ctrl+K", "shift+alt+p": "Shift+Alt+P", "meta+j": "Meta+J", "k": "K"} {
		if got := HintFor(in, false); got != want {
			t.Errorf("HintFor(%q, false) = %q, want %q", in, got, want)
		}
	}
	if HintFor("mod+k", true) != Hint("mod+k") {
		t.Errorf("HintFor(mod+k, true) = %q, want Hint's %q", HintFor("mod+k", true), Hint("mod+k"))
	}
}

// The hint and the listener share one platform test, so the hint names the
// chord the listener wants.
func TestTrigger_HintUsesTheListenersPlatformTest(t *testing.T) {
	trig := render(t, Trigger(Props{}))
	pal := render(t, Palette(sample()))
	esc := templ.EscapeString(isMacJS)
	if !strings.Contains(trig, esc) || !strings.Contains(pal, "isMac: "+esc) {
		t.Errorf("isMacJS missing from the trigger hint or the palette listener")
	}
}

// The phone sheet follows window.visualViewport: the x-data sets the two
// custom properties the sheet's classes read, and drops its listeners once
// the palette leaves the page.
func TestPalette_SheetFollowsVisualViewport(t *testing.T) {
	got := render(t, Palette(sample()))
	for _, want := range []string{"window.visualViewport", "--wf-palette-vh", "--wf-palette-top", "vv.offsetTop", "removeEventListener(&#39;resize&#39;, fit)"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestGroupItems(t *testing.T) {
	gs := groupItems([]Item{{Label: "a", Group: "X"}, {Label: "b"}, {Label: "c", Group: "X"}, {Label: "d", Group: "Y"}})
	if len(gs) != 3 || gs[0].Name != "X" || gs[1].Name != "" || gs[2].Name != "Y" {
		t.Fatalf("groups = %+v", gs)
	}
	if len(gs[0].Items) != 2 || gs[0].Items[1].Label != "c" {
		t.Errorf("group X = %+v", gs[0].Items)
	}
}

// TestPalette_UnsafeHrefNeutralised: an item's Href goes through templ's
// URL sanitizer, so a javascript: href renders about:invalid.
func TestPalette_UnsafeHrefNeutralised(t *testing.T) {
	got := render(t, Palette(Props{Items: []Item{{Label: "Evil", Href: "javascript:alert(1)"}, {Label: "Fine", Href: "/ok?x=1&y=2"}}}))
	if !strings.Contains(got, `href="about:invalid`) {
		t.Errorf("javascript: href must be neutralised to about:invalid:\n%s", got)
	}
	if strings.Contains(got, `javascript:`) {
		t.Errorf("javascript: leaked into href:\n%s", got)
	}
	if !strings.Contains(got, `href="/ok?x=1&amp;y=2"`) {
		t.Errorf("a plain href must survive:\n%s", got)
	}
}

// dialog.Content gives every dialog a height limit and lets the panel
// scroll. The palette has its own limit and scrolls its list, not its panel:
// its classes must win the merge, with none of the dialog's left beside them.
func TestPalette_KeepsItsOwnHeightLimitAndOverflow(t *testing.T) {
	got := render(t, Palette(sample()))
	i := strings.Index(got, `" data-tui-dialog-content data-dialog-instance=`)
	if i < 0 {
		t.Fatalf("no dialog content element in:\n%s", got)
	}
	start := strings.LastIndex(got[:i], `class="`) + len(`class="`)
	classes := strings.Fields(got[start : start+strings.Index(got[start:], `"`)])
	has := func(want string) bool {
		for _, c := range classes {
			if c == want {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"overflow-hidden", "max-h-[var(--wf-palette-vh,60dvh)]", "sm:max-h-[80vh]"} {
		if !has(want) {
			t.Errorf("missing %q in %v", want, classes)
		}
	}
	for _, not := range []string{"overflow-y-auto", "max-h-[calc(100dvh-2rem)]"} {
		if has(not) {
			t.Errorf("the dialog's %q survived the merge in %v", not, classes)
		}
	}
}
