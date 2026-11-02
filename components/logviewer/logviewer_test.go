package logviewer_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/logviewer"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestLogViewer_ZeroValue(t *testing.T) {
	out := render(t, logviewer.LogViewer())
	for _, want := range []string{`class="wf-log`, `role="log"`, "h-[400px]", `data-wf-log-max="0"`, `follow: false`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	for _, no := range []string{"hx-get", `type="search"`, `role="switch"`, "flex-wrap"} {
		if strings.Contains(out, no) {
			t.Errorf("zero Props must not render %q (no toolbar, no polling):\n%s", no, out)
		}
	}
}

func TestLogViewer_LevelsAndTime(t *testing.T) {
	at := time.Date(2026, 10, 31, 15, 4, 5, 0, time.UTC)
	out := render(t, logviewer.LogViewer(logviewer.Props{Lines: []logviewer.Line{
		{Text: "plain", At: at},
		{Text: "note", Level: logviewer.LevelInfo},
		{Text: "careful", Level: logviewer.LevelWarn},
		{Text: "boom <b>", Level: logviewer.LevelError},
		{Text: ""},
	}}))
	for _, want := range []string{
		`<time datetime="2026-10-31T15:04:05Z"`, ">15:04:05</time>",
		`data-level="info"`, "text-info",
		`data-level="warn"`, "text-warning",
		`data-level="error"`, "text-destructive",
		"boom &lt;b&gt;",
		`<span class="min-w-0 flex-1"> </span>`, // an empty line keeps its height
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, `class="wf-log-line`); got != 5 {
		t.Errorf("want 5 lines, got %d:\n%s", got, out)
	}
	if strings.Count(out, "<time") != 1 {
		t.Errorf("only the line with At renders a <time>:\n%s", out)
	}
	// templ may emit one space between <time> and the text span; the line is
	// a flex container, which does not render whitespace-only text runs, so
	// white-space: pre cannot show it. Only a newline would be a defect.
	if strings.Contains(out, "</time>\n") {
		t.Errorf("newline between <time> and text:\n%s", out)
	}
}

func TestLogViewer_FilterBox(t *testing.T) {
	out := render(t, logviewer.LogViewer(logviewer.Props{Filter: true}))
	for _, want := range []string{`type="search"`, `x-model="q"`, `aria-label="Filter lines"`, `x-effect="apply()"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestLogViewer_PollAttributes(t *testing.T) {
	out := render(t, logviewer.LogViewer(logviewer.Props{Src: "/logs/tail", Follow: true}))
	for _, want := range []string{
		`hx-get="/logs/tail"`, `hx-trigger="every 5s"`, `hx-swap="beforeend"`,
		`x-on:htmx:after-swap="onSwap()"`, `role="switch"`, `x-model="follow"`, `checked`, `follow: true`,
		`x-on:scroll="onScroll()"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	// The poll lives on the lines container, not the scroll box.
	if !strings.Contains(out, `<div x-ref="lines" class="whitespace-pre" hx-get=`) {
		t.Errorf("hx-get must be on the lines container:\n%s", out)
	}
	fast := render(t, logviewer.LogViewer(logviewer.Props{Src: "/t", Every: 1500 * time.Millisecond}))
	if !strings.Contains(fast, `hx-trigger="every 1500ms"`) {
		t.Errorf("sub-second Every renders in ms:\n%s", fast)
	}
	slow := render(t, logviewer.LogViewer(logviewer.Props{Src: "/t", Every: 30 * time.Second}))
	if !strings.Contains(slow, `hx-trigger="every 30s"`) {
		t.Errorf("whole-second Every renders in s:\n%s", slow)
	}
}

func TestLogViewer_WrapInheritsToLines(t *testing.T) {
	wrapped := render(t, logviewer.LogViewer(logviewer.Props{Wrap: true, Lines: []logviewer.Line{{Text: "x"}}}))
	if !strings.Contains(wrapped, `x-ref="lines" class="whitespace-pre-wrap break-words"`) {
		t.Errorf("Wrap sets soft wrap on the lines container:\n%s", wrapped)
	}
	plain := render(t, logviewer.LogViewer(logviewer.Props{Lines: []logviewer.Line{{Text: "x"}}}))
	if !strings.Contains(plain, `x-ref="lines" class="whitespace-pre"`) || !strings.Contains(plain, "overflow-auto") {
		t.Errorf("no Wrap: lines keep white-space pre and the box scrolls:\n%s", plain)
	}
}

func TestLogViewer_MaxLines(t *testing.T) {
	lines := make([]logviewer.Line, 10)
	for i := range lines {
		lines[i] = logviewer.Line{Text: "line " + string(rune('a'+i))}
	}
	out := render(t, logviewer.LogViewer(logviewer.Props{Lines: lines, MaxLines: 3}))
	if !strings.Contains(out, `data-wf-log-max="3"`) || !strings.Contains(out, "max: 3") {
		t.Errorf("MaxLines must reach the attribute and the Alpine state:\n%s", out)
	}
	if got := strings.Count(out, `class="wf-log-line`); got != 3 {
		t.Errorf("server render keeps the last MaxLines (3), got %d:\n%s", got, out)
	}
	for _, want := range []string{"line h", "line i", "line j"} {
		if !strings.Contains(out, want) {
			t.Errorf("newest lines kept; missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "line a") {
		t.Errorf("oldest lines dropped:\n%s", out)
	}
}

func TestLogViewer_HeightTiers(t *testing.T) {
	for h, want := range map[logviewer.Height]string{
		logviewer.HeightSm: "h-[240px]",
		logviewer.HeightMd: "h-[400px]",
		logviewer.HeightLg: "h-[600px]",
	} {
		out := render(t, logviewer.LogViewer(logviewer.Props{Height: h}))
		if !strings.Contains(out, want) {
			t.Errorf("%q: missing %q:\n%s", h, want, out)
		}
	}
}

func TestLogViewer_DownloadAndEmpty(t *testing.T) {
	out := render(t, logviewer.LogViewer(logviewer.Props{
		Download: &action.Action{Label: "Download log", Href: "/logs/raw"},
		Empty:    "No lines yet",
	}))
	for _, want := range []string{`href="/logs/raw"`, "<span>Download log</span>", `x-show="n === 0"`, ">No lines yet</p>", "h-8"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	withLines := render(t, logviewer.LogViewer(logviewer.Props{Empty: "No lines yet", Lines: []logviewer.Line{{Text: "x"}}}))
	if strings.Contains(withLines, "No lines yet") {
		t.Errorf("Empty text is not rendered when there are lines:\n%s", withLines)
	}
}

func TestLogViewer_IDClassAttributes(t *testing.T) {
	out := render(t, logviewer.LogViewer(logviewer.Props{ID: "deploy-log", Class: "mt-4", Attributes: templ.Attributes{"data-test": "yes"}}))
	for _, want := range []string{`id="deploy-log"`, "mt-4", `data-test="yes"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// TestFragment_PollingHandler is the Src contract: a handler renders
// Fragment(newLines) and the body is lines only, ready for beforeend.
func TestFragment_PollingHandler(t *testing.T) {
	at := time.Date(2026, 10, 31, 15, 4, 5, 0, time.UTC)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = logviewer.Fragment([]logviewer.Line{
			{Text: "pulled image", At: at},
			{Text: "restart failed", Level: logviewer.LevelError},
		}).Render(r.Context(), w)
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/logs/tail", nil))
	body := rec.Body.String()
	if !strings.HasPrefix(body, `<div class="wf-log-line`) || !strings.HasSuffix(body, "</div>") {
		t.Errorf("Fragment is bare line divs:\n%s", body)
	}
	if got := strings.Count(body, `class="wf-log-line`); got != 2 {
		t.Errorf("want 2 lines, got %d:\n%s", got, body)
	}
	for _, no := range []string{"wf-log ", "x-data", "hx-get", `role="log"`} {
		if strings.Contains(body, no) {
			t.Errorf("Fragment must not include the viewer chrome (%q):\n%s", no, body)
		}
	}
	if empty := render(t, logviewer.Fragment(nil)); empty != "" {
		t.Errorf("Fragment(nil) is an empty body, got %q", empty)
	}
}
