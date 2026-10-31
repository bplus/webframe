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

// The error-swap guard must load after confirm.js (the first tag of the
// bundle) and before any component script, so it is listening before the
// first swap a component script could trigger.
func TestScripts_ErrorGuardAfterConfirm(t *testing.T) {
	got := renderScripts(t)
	confirm := strings.Index(got, "/static/js/confirm.js")
	errors := strings.Index(got, "/static/js/htmx-errors.js")
	tabs := strings.Index(got, "/static/js/tabs.min.js")
	theme := strings.Index(got, "/static/js/theme-toggle.js")
	if confirm < 0 || errors < 0 || tabs < 0 || theme < 0 {
		t.Fatalf("bundle missing a script (confirm %d, errors %d, tabs %d, theme %d):\n%s", confirm, errors, tabs, theme, got)
	}
	if !(confirm < errors && errors < tabs && tabs < theme) {
		t.Errorf("want confirm.js < htmx-errors.js < tabs.min.js < theme-toggle.js, got %d, %d, %d, %d", confirm, errors, tabs, theme)
	}
	if strings.Count(got, "/static/js/htmx-errors.js") != 1 {
		t.Errorf("htmx-errors.js should be loaded exactly once:\n%s", got)
	}
}

// Document renders the bundle, so a full page carries the guard too.
func TestDocument_LoadsErrorGuard(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, nil)
	confirm := strings.Index(got, "/static/js/confirm.js")
	errors := strings.Index(got, "/static/js/htmx-errors.js")
	if confirm < 0 || errors < 0 || errors < confirm {
		t.Errorf("want htmx-errors.js after confirm.js in the document (confirm %d, errors %d):\n%s", confirm, errors, got)
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
	badgeTag := got[badge:badge+strings.Index(got[badge:], ">")]
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
	dotTag := got[dot:dot+strings.Index(got[dot:], ">")]
	if !strings.Contains(dotTag, "data-wf-swallow-errors") {
		t.Errorf("live dot lacks data-wf-swallow-errors: %s", dotTag)
	}

	// Sidebar layout renders the same badge span through navLink.
	got = renderShell(t, Props{AppName: "A", Layout: LayoutSidebar, Nav: nav}, nil)
	badge = strings.Index(got, `hx-get="/inbox/count"`)
	if badge < 0 {
		t.Fatalf("sidebar badge span missing in:\n%s", got)
	}
	badgeTag = got[badge:badge+strings.Index(got[badge:], ">")]
	if !strings.Contains(badgeTag, "data-wf-swallow-errors") {
		t.Errorf("sidebar badge span lacks data-wf-swallow-errors: %s", badgeTag)
	}
}
