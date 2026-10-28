package breadcrumbs

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Breadcrumbs(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestBreadcrumbs_LastIsCurrentNotLinked(t *testing.T) {
	out := render(t, Props{Items: []Item{
		{Label: "Home", Href: "/"},
		{Label: "Docs", Href: "/docs"},
		{Label: "Intro", Href: "/docs/intro"},
	}})
	if got := strings.Count(out, "<a href"); got != 2 {
		t.Errorf("want 2 links, got %d in %s", got, out)
	}
	i := strings.Index(out, `aria-current="page"`)
	if i < 0 {
		t.Fatalf("missing aria-current in %s", out)
	}
	if !strings.Contains(out[i:], "Intro") {
		t.Errorf("last label not inside aria-current span: %s", out)
	}
	if strings.Contains(out, ">Intro</a>") {
		t.Errorf("last label is linked: %s", out)
	}
}

func TestBreadcrumbs_Empty(t *testing.T) {
	out := render(t, Props{})
	if !strings.Contains(out, "<nav") {
		t.Errorf("missing nav: %s", out)
	}
	if strings.Contains(out, "<a") {
		t.Errorf("unexpected link: %s", out)
	}
}
