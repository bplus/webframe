package appshell

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func renderScripts(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Scripts().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// The bundle carries each first-party script exactly once.
func TestScripts_LoadsEachOnce(t *testing.T) {
	got := renderScripts(t)
	for _, src := range []string{
		"/static/js/confirm.js",
		"/static/js/htmx-errors.js",
		"/static/js/splitpane.js",
		"/static/js/tabs.min.js",
		"/static/js/theme-toggle.js",
	} {
		if n := strings.Count(got, src); n != 1 {
			t.Errorf("%s loaded %d times, want 1:\n%s", src, n, got)
		}
	}
}

// Document renders the bundle, so a full page carries the guard too.
func TestDocument_LoadsErrorGuard(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, nil)
	for _, src := range []string{"/static/js/confirm.js", "/static/js/htmx-errors.js"} {
		if !strings.Contains(got, src) {
			t.Errorf("document missing %s:\n%s", src, got)
		}
	}
}

// The shell's polled elements must be marked so htmx-errors.js never swaps
// an error body into them.
func TestShell_LiveElementsSwallowErrors(t *testing.T) {
	nav := []NavItem{{Label: "Inbox", Href: "/inbox", Badge: &LiveBadge{HxGet: "/inbox/count", Every: "10s"}}}
	got := renderShell(t, Props{AppName: "A", Nav: nav, LiveDot: &LiveDot{HxGet: "/health"}}, nil)

	badge := strings.Index(got, `hx-get="/inbox/count"`)
	if badge < 0 {
		t.Fatalf("badge span missing in:\n%s", got)
	}
	badgeTag := got[badge : badge+strings.Index(got[badge:], ">")]
	if !strings.Contains(badgeTag, "data-wf-swallow-errors") {
		t.Errorf("badge span lacks data-wf-swallow-errors: %s", badgeTag)
	}
	if !strings.Contains(badgeTag, `hx-trigger="load, every 10s"`) {
		t.Errorf("badge span trigger changed: %s", badgeTag)
	}

	dot := strings.Index(got, `class="wf-live-dot"`)
	if dot < 0 {
		t.Fatalf("live dot missing in:\n%s", got)
	}
	dotTag := got[dot : dot+strings.Index(got[dot:], ">")]
	if !strings.Contains(dotTag, "data-wf-swallow-errors") {
		t.Errorf("live dot lacks data-wf-swallow-errors: %s", dotTag)
	}

	// Sidebar layout renders the same badge span through navLink.
	got = renderShell(t, Props{AppName: "A", Layout: LayoutSidebar, Nav: nav}, nil)
	badge = strings.Index(got, `hx-get="/inbox/count"`)
	if badge < 0 {
		t.Fatalf("sidebar badge span missing in:\n%s", got)
	}
	badgeTag = got[badge : badge+strings.Index(got[badge:], ">")]
	if !strings.Contains(badgeTag, "data-wf-swallow-errors") {
		t.Errorf("sidebar badge span lacks data-wf-swallow-errors: %s", badgeTag)
	}
}
