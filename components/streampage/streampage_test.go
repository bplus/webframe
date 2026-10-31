package streampage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/layout"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/trafficstream"
)

const scriptSrc = "/static/js/trafficstream.min.js"

func text(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Page(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// inOrder fails unless each needle appears after the previous one.
func inOrder(t *testing.T, out string, needles ...string) {
	t.Helper()
	pos := 0
	for _, n := range needles {
		i := strings.Index(out[pos:], n)
		if i < 0 {
			t.Fatalf("missing %q after offset %d in:\n%s", n, pos, out)
		}
		pos += i + len(n)
	}
}

func trafficPage() Props {
	return Props{
		ID:     "traffic",
		Header: pageheader.Props{Title: "Traffic Stream", Subtitle: "Live request and response activity."},
		Intro:  "Events arrive every second or so.",
		Stream: trafficstream.Props{
			ID:              "traffic-panel",
			StreamURL:       "/api/traffic/stream",
			Height:          560,
			ProviderOptions: []string{"anthropic", "openai"},
			InitialRows: []trafficstream.Row{{
				Time: "12:00:01", Provider: "anthropic", Direction: "request",
				Event: "request.received", TraceID: "trace-demo-001", Summary: "POST /v1/messages",
			}},
		},
	}
}

func TestPage_CompositionOrder(t *testing.T) {
	p := trafficPage()
	out := render(t, p)
	inOrder(t, out,
		`id="traffic"`,
		"<h1", "Traffic Stream",
		"Events arrive every second or so.",
		`id="traffic-panel"`, "data-tui-trafficstream",
		`data-tui-trafficstream-url="/api/traffic/stream"`,
		"POST /v1/messages",
		scriptSrc,
	)
}

func TestPage_StreamPropsPassThrough(t *testing.T) {
	out := render(t, trafficPage())
	for _, want := range []string{"height:560px", `<option value="anthropic">`, `<option value="openai">`, "trace-demo-001"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// The panel keeps its own controls.
	for _, want := range []string{"data-tui-trafficstream-search", "data-tui-trafficstream-provider", "data-tui-trafficstream-toggle"} {
		if !strings.Contains(out, want) {
			t.Errorf("stream control %q missing:\n%s", want, out)
		}
	}
}

func TestPage_NoIntroNoFilters(t *testing.T) {
	p := trafficPage()
	p.Intro = ""
	out := render(t, p)
	if strings.Contains(out, "data-tui-filterbar") {
		t.Errorf("no Filters should render no filter bar:\n%s", out)
	}
	// The body opens straight onto the panel: no intro <p> before it.
	if !strings.Contains(out, `<div class="flex flex-col gap-4"><section`) {
		t.Errorf("empty Intro should render no intro line before the panel:\n%s", out)
	}
}

func TestPage_ScriptOnce(t *testing.T) {
	out := render(t, trafficPage())
	if n := strings.Count(out, scriptSrc); n != 1 {
		t.Errorf("want the trafficstream script exactly once, got %d:\n%s", n, out)
	}
	if n := strings.Count(out, "<script"); n != 1 {
		t.Errorf("want one <script> tag, got %d:\n%s", n, out)
	}
	// The script follows the panel, so the panel is in the DOM when it runs.
	inOrder(t, out, "data-tui-trafficstream-rows", scriptSrc)
}

func TestPage_NoScript(t *testing.T) {
	p := trafficPage()
	p.NoScript = true
	out := render(t, p)
	if strings.Contains(out, "<script") {
		t.Errorf("NoScript should render no script tag:\n%s", out)
	}
	if !strings.Contains(out, "data-tui-trafficstream") {
		t.Errorf("the panel must still render:\n%s", out)
	}
}

func TestPage_Aside(t *testing.T) {
	p := trafficPage()
	p.Aside = text(`<p id="rail">Legend</p>`)
	p.AsideWidth = layout.AsideL
	p.AsideSticky = true
	out := render(t, p)
	// trafficstream's own toolbar also uses md:flex-row, so the TwoColumn is
	// recognised by its <aside>, which nothing else on the page renders.
	if !strings.Contains(out, "<aside") || !strings.Contains(out, `id="rail"`) {
		t.Errorf("aside should wrap the body in a TwoColumn:\n%s", out)
	}
	if !strings.Contains(out, "md:sticky") {
		t.Errorf("AsideSticky should reach the aside:\n%s", out)
	}
	// Header stays above the two columns; the stream is in the main column
	// before the aside; the script comes after everything.
	inOrder(t, out, "<h1", "data-tui-trafficstream", "<aside", `id="rail"`, "</aside>", scriptSrc)
}

func TestPage_NoAsideNoTwoColumn(t *testing.T) {
	out := render(t, trafficPage())
	if strings.Contains(out, "<aside") {
		t.Errorf("no Aside should render no TwoColumn:\n%s", out)
	}
}

func TestPage_ClassAttributesAndIDs(t *testing.T) {
	p := trafficPage()
	p.Class = "max-w-5xl"
	p.Attributes = templ.Attributes{"data-page": "traffic"}
	out := render(t, p)
	for _, want := range []string{"max-w-5xl", `data-page="traffic"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestPage_ZeroValueRenders(t *testing.T) {
	out := render(t, Props{})
	if !strings.Contains(out, "data-tui-trafficstream") || !strings.Contains(out, scriptSrc) {
		t.Errorf("zero Props should still render a panel and its script:\n%s", out)
	}
}

func testApp() *appshell.App {
	return &appshell.App{Name: "Demo", Nav: []appshell.NavItem{{Label: "Traffic", Href: "/traffic"}}}
}

func serve(t *testing.T, app *appshell.App, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Respond(w, r, "Traffic", trafficPage()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	if app != nil {
		h = app.Middleware(h)
	}
	req := httptest.NewRequest("GET", "/traffic", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRespond_FullDocument(t *testing.T) {
	rec := serve(t, testApp(), nil)
	out := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, out)
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(out)), "<!doctype") {
		t.Errorf("want full document: %s", out[:min(len(out), 200)])
	}
	for _, want := range []string{"Traffic Stream", `data-tui-trafficstream-url="/api/traffic/stream"`, scriptSrc} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s", want)
		}
	}
	if n := strings.Count(out, scriptSrc); n != 1 {
		t.Errorf("full document should load the trafficstream script once, got %d", n)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestRespond_Fragment(t *testing.T) {
	rec := serve(t, testApp(), map[string]string{"HX-Request": "true"})
	out := rec.Body.String()
	if strings.Contains(strings.ToLower(out), "<!doctype") {
		t.Errorf("plain htmx request should get the bare body: %s", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "<h1") || !strings.Contains(out, "data-tui-trafficstream") {
		t.Errorf("fragment should carry the header and the panel: %s", out)
	}
}

func TestRespond_NoApp(t *testing.T) {
	rec := serve(t, nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 without Middleware, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Middleware") {
		t.Errorf("error should name Middleware: %s", rec.Body.String())
	}
}
