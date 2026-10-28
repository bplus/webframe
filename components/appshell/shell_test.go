package appshell

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
)

func renderShell(t *testing.T, p Props, children templ.Component) string {
	t.Helper()
	ctx := context.Background()
	if children != nil {
		ctx = templ.WithChildren(ctx, children)
	}
	var buf bytes.Buffer
	if err := Shell(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		item NavItem
		path string
		want bool
	}{
		{NavItem{Href: "/"}, "/", true},
		{NavItem{Href: "/"}, "/x", false},
		{NavItem{Href: "/factory", Match: "/factory"}, "/factory/123", true},
		{NavItem{Href: "/inbox"}, "/inboxes", false},
	}
	for _, tt := range tests {
		if got := isActive(tt.item, tt.path); got != tt.want {
			t.Errorf("isActive(%+v, %q) = %v, want %v", tt.item, tt.path, got, tt.want)
		}
	}
}

func TestShell_TopbarMoreDisclosure(t *testing.T) {
	nav := []NavItem{
		{Label: "Alpha", Href: "/a"},
		{Label: "Bravo", Href: "/b"},
		{Label: "Charlie", Href: "/c"},
		{Label: "Delta", Href: "/d"},
		{Label: "Echo", Href: "/e"},
		{Label: "Foxtrot", Href: "/f"},
	}
	got := renderShell(t, Props{AppName: "A", Nav: nav, MaxVisible: 4}, nil)
	more := strings.Index(got, "<summary>More</summary>")
	if more < 0 {
		t.Fatalf("missing More summary in:\n%s", got)
	}
	if i := strings.Index(got, "Alpha"); i < 0 || i > more {
		t.Errorf("first item should appear before More (idx %d, more %d)", i, more)
	}
	for _, label := range []string{"Echo", "Foxtrot"} {
		if i := strings.Index(got, label); i < more {
			t.Errorf("%s should appear only after More (idx %d, more %d)", label, i, more)
		}
	}
}

func TestShell_ActiveItemMarked(t *testing.T) {
	nav := []NavItem{
		{Label: "Home", Href: "/home"},
		{Label: "Inbox", Href: "/inbox"},
	}
	got := renderShell(t, Props{AppName: "A", Nav: nav, CurrentPath: "/inbox"}, nil)
	i := strings.Index(got, `href="/inbox"`)
	if i < 0 {
		t.Fatalf("missing inbox link in:\n%s", got)
	}
	start := strings.LastIndex(got[:i], "<a")
	end := i + strings.Index(got[i:], ">")
	tag := got[start : end+1]
	if !strings.Contains(tag, `aria-current="page"`) || !strings.Contains(tag, "wf-nav-link--active") {
		t.Errorf("inbox link not marked active: %s", tag)
	}
	j := strings.Index(got, `href="/home"`)
	hs := strings.LastIndex(got[:j], "<a")
	he := j + strings.Index(got[j:], ">")
	if strings.Contains(got[hs:he+1], "aria-current") {
		t.Errorf("home link wrongly marked active: %s", got[hs:he+1])
	}
}

func TestShell_SidebarGroups(t *testing.T) {
	got := renderShell(t, Props{
		AppName: "A",
		Layout:  LayoutSidebar,
		Groups: []NavGroup{
			{Title: "Mine", Items: []NavItem{{Label: "Inbox", Href: "/inbox"}}},
			{Title: "Discover"},
		},
	}, nil)
	for _, want := range []string{`<aside class="wf-sidebar`, "Mine", "Discover", "wf-shell--sidebar"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestShell_UserMenu(t *testing.T) {
	got := renderShell(t, Props{
		AppName: "A",
		User: &User{Name: "Ada", Menu: []action.Action{
			{Label: "Sign out", Href: "/logout"},
		}},
	}, nil)
	for _, want := range []string{"Sign out", "/logout"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestShell_WidthClass(t *testing.T) {
	got := renderShell(t, Props{AppName: "A", Width: WidthNarrow}, nil)
	if !strings.Contains(got, "wf-content--narrow") {
		t.Errorf("missing wf-content--narrow in:\n%s", got)
	}
}

func TestShell_RendersChildrenInMain(t *testing.T) {
	got := renderShell(t, Props{AppName: "A"}, templ.Raw("<p>body-here</p>"))
	mainStart := strings.Index(got, "<main")
	mainEnd := strings.Index(got, "</main>")
	flash := strings.Index(got, `id="wf-flash"`)
	child := strings.Index(got, "<p>body-here</p>")
	if mainStart < 0 || mainEnd < 0 || flash < 0 || child < 0 {
		t.Fatalf("missing main/flash/child in:\n%s", got)
	}
	if !(mainStart < flash && flash < child && child < mainEnd) {
		t.Errorf("want <main < wf-flash < child < </main>; got %d %d %d %d", mainStart, flash, child, mainEnd)
	}
}

func TestShell_ThemeToggleInMoreList(t *testing.T) {
	// Two nav items: nothing overflows, but the phone-width toggle still
	// needs the More list, so it renders as a phone-only disclosure.
	nav := []NavItem{{Label: "Alpha", Href: "/a"}, {Label: "Bravo", Href: "/b"}}
	got := renderShell(t, Props{AppName: "A", Nav: nav, ThemeToggle: true}, nil)
	if !strings.Contains(got, `class="wf-nav-more wf-nav-more--phone-only"`) {
		t.Errorf("More should render phone-only when only the toggle needs it:\n%s", got)
	}
	inMenu := `<button type="button" data-theme-toggle class="wf-theme-toggle wf-theme-toggle--in-menu">Dark Mode</button>`
	i := strings.Index(got, inMenu)
	if i < 0 {
		t.Fatalf("missing in-menu theme toggle:\n%s", got)
	}
	if end := strings.Index(got, "</details>"); end < 0 || i > end {
		t.Errorf("in-menu toggle should sit inside the More disclosure (idx %d, </details> %d)", i, end)
	}
	if n := strings.Count(got, "data-theme-toggle"); n != 2 {
		t.Errorf("want 2 theme toggles (topbar + More list), got %d", n)
	}

	got = renderShell(t, Props{AppName: "A", Nav: nav}, nil)
	if strings.Contains(got, "data-theme-toggle") || strings.Contains(got, "wf-nav-more") {
		t.Errorf("no toggle and no overflow should render neither toggle nor More:\n%s", got)
	}
}

func TestShell_NavHrefsSanitized(t *testing.T) {
	bad := "javascript:alert(1)"
	for _, layout := range []Layout{LayoutTopbar, LayoutSidebar} {
		out := renderShell(t, Props{
			AppName:  "Demo",
			HomeHref: bad,
			Layout:   layout,
			Nav:      []NavItem{{Label: "Evil", Href: bad}},
			Groups:   []NavGroup{{Title: "G", Items: []NavItem{{Label: "Evil2", Href: bad}}}},
		}, nil)
		if strings.Contains(out, "javascript:") {
			t.Errorf("layout %v: javascript: URL not sanitized: %s", layout, out)
		}
	}
}
