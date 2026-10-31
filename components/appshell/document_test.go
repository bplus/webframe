package appshell

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func renderDoc(t *testing.T, p DocProps, children templ.Component) string {
	t.Helper()
	ctx := context.Background()
	if children != nil {
		ctx = templ.WithChildren(ctx, children)
	}
	var buf bytes.Buffer
	if err := Document(p).Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestDocument_TitleFormat(t *testing.T) {
	got := renderDoc(t, DocProps{Title: "Movies", AppName: "Acme"}, nil)
	if !strings.Contains(got, "<title>Movies — Acme</title>") {
		t.Errorf("missing full title in:\n%s", got)
	}
	got = renderDoc(t, DocProps{AppName: "Acme"}, nil)
	if !strings.Contains(got, "<title>Acme</title>") {
		t.Errorf("missing app-only title in:\n%s", got)
	}
}

func TestDocument_EscapesTitle(t *testing.T) {
	got := renderDoc(t, DocProps{Title: "<script>x", AppName: "A"}, nil)
	if strings.Contains(got, "<script>x") {
		t.Errorf("title not escaped:\n%s", got)
	}
	if !strings.Contains(got, "&lt;script&gt;x") {
		t.Errorf("expected escaped title in:\n%s", got)
	}
}

func TestDocument_LoadsBundle(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, nil)
	for _, want := range []string{"output.css", "htmx.min.js", "alpine.min.js", "tabs.min.js", "theme-toggle.js", `id="toast-container"`, "data-theme"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestDocument_RendersChildren(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, templ.Raw("<p>hi</p>"))
	start := strings.Index(got, "<body")
	end := strings.Index(got, "</body>")
	idx := strings.Index(got, "<p>hi</p>")
	if start < 0 || end < 0 || idx < start || idx > end {
		t.Errorf("children not inside body:\n%s", got)
	}
}

func TestDocument_ConfirmHostOnceAfterToastContainer(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, templ.Raw("<p>hi</p>"))
	const host = `<div id="wf-confirm"></div>`
	if n := strings.Count(got, `id="wf-confirm"`); n != 1 {
		t.Fatalf("want exactly one #wf-confirm, got %d in:\n%s", n, got)
	}
	if !strings.Contains(got, host) {
		t.Errorf("confirm host is not empty in:\n%s", got)
	}
	toast := strings.Index(got, `id="toast-container"`)
	confirm := strings.Index(got, host)
	scripts := strings.Index(got, "confirm.js")
	child := strings.Index(got, "<p>hi</p>")
	end := strings.Index(got, "</body>")
	if !(child < toast && toast < confirm && confirm < scripts && scripts < end) {
		t.Errorf("want children < toast-container < wf-confirm < scripts < </body>; got %d %d %d %d %d", child, toast, confirm, scripts, end)
	}
}

func TestDocument_ConfirmScriptFirstInBundleAfterHTMX(t *testing.T) {
	got := renderDoc(t, DocProps{AppName: "A"}, nil)
	htmx := strings.Index(got, "htmx.min.js")
	confirm := strings.Index(got, "/static/js/confirm.js")
	tabs := strings.Index(got, "tabs.min.js")
	if htmx < 0 || confirm < 0 || tabs < 0 {
		t.Fatalf("missing htmx/confirm/tabs script in:\n%s", got)
	}
	if !(htmx < confirm && confirm < tabs) {
		t.Errorf("want htmx < confirm.js < component scripts; got %d %d %d", htmx, confirm, tabs)
	}
}
