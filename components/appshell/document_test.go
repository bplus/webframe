package appshell

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/utils"
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
	const host = `id="wf-confirm"`
	if n := strings.Count(got, host); n != 1 {
		t.Fatalf("want exactly one #wf-confirm, got %d in:\n%s", n, got)
	}
	// The host is confirmdialog.Host: a closed dialog.Dialog instance named
	// wf-confirm (what confirm.js opens), not an empty div.
	for _, want := range []string{`data-dialog-instance="wf-confirm"`, `data-wf-confirm-part="cancel"`, `data-tui-dialog-hidden="true"`} {
		if !strings.Contains(got, want) {
			t.Errorf("confirm host missing %q in:\n%s", want, got)
		}
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

// TestDocument_HTMXConfigSwapsErrors: the htmx-config meta precedes the
// htmx script and tells htmx to swap 4xx/5xx responses, so a 422 form
// re-render, a 409 carrying only a toast and a 404 error page all reach
// the page.
func TestDocument_HTMXConfigSwapsErrors(t *testing.T) {
	got := renderDoc(t, DocProps{Title: "T", AppName: "A"}, nil)
	meta := strings.Index(got, `<meta name="htmx-config"`)
	script := strings.Index(got, "htmx.min.js")
	if meta < 0 || script < 0 || meta > script {
		t.Fatalf("htmx-config meta missing or after the htmx script (meta=%d script=%d)", meta, script)
	}
	for _, want := range []string{`[45]..`, `swap`, `error`, `204`} {
		if !strings.Contains(got, want) {
			t.Errorf("htmx-config lacks %q:\n%s", want, got[meta:meta+300])
		}
	}
}

// htmx and Alpine go through assets.URL like the stylesheet and the bundle,
// so a mount path or cache-busting override applies to them too.
func TestDocument_CoreScriptsUseAssetURL(t *testing.T) {
	orig := utils.ScriptURL
	t.Cleanup(func() { utils.ScriptURL = orig })
	utils.ScriptURL = func(path string) string { return "/mounted" + path + "?v=TEST" }
	got := renderDoc(t, DocProps{AppName: "A"}, nil)
	for _, want := range []string{
		`<script src="/mounted/static/js/htmx.min.js?v=TEST"></script>`,
		`<script defer src="/mounted/static/js/alpine.min.js?v=TEST"></script>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// A browser with JavaScript turned off shows a warning at the top of
// every page; the notice sits inside <noscript> so it costs nothing when
// JavaScript runs.
func TestDocument_NoscriptNotice(t *testing.T) {
	got := renderDoc(t, DocProps{Title: "Movies", AppName: "Acme"}, templ.Raw("<main>page</main>"))
	assertNoscriptNotice(t, got)
	if strings.Index(got, "<noscript>") > strings.Index(got, "<main>page</main>") {
		t.Errorf("noscript notice should come before the page content:\n%s", got)
	}
}

func TestRenderBare_NoscriptNotice(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/signin", nil)
	if err := RenderBare(rec, req, "Acme", "Sign in", templ.Raw("<main>form</main>")); err != nil {
		t.Fatalf("RenderBare: %v", err)
	}
	assertNoscriptNotice(t, rec.Body.String())
}

func assertNoscriptNotice(t *testing.T, html string) {
	t.Helper()
	start := strings.Index(html, "<noscript>")
	end := strings.Index(html, "</noscript>")
	if start < 0 || end < start {
		t.Fatalf("no <noscript> block in:\n%s", html)
	}
	block := html[start:end]
	for _, want := range []string{noscriptTitle, noscriptBody, `role="status"`, "border-warning"} {
		if !strings.Contains(block, want) {
			t.Errorf("noscript block missing %q:\n%s", want, block)
		}
	}
	if strings.Contains(noscriptBody, "—") || strings.Contains(noscriptTitle, "—") {
		t.Errorf("notice wording holds an em dash")
	}
}
