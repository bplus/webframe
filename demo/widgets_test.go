package demo

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

// widgetsHandler is fullHandler with the log reset too, so every widget
// test starts from the seeded store, account and log.
func widgetsHandler(t *testing.T) http.Handler {
	t.Helper()
	Settings = NewAccount()
	Logs = seedLogs(NewLogBuffer(), time.Now())
	return fullHandler(t)
}

var cursorRe = regexp.MustCompile(`hx-get="/logs/tail\?cursor=([A-Za-z0-9_-]+)"`)

// TestWidgets_LogsPage: /logs renders the viewer in the sidebar shell with
// the seeded lines and a poll every three seconds carrying a fresh cursor;
// /logs/tail answers 204 while nothing is new and 200 with the new lines
// (once) after an Items mutation; a forgotten cursor is re-seated.
func TestWidgets_LogsPage(t *testing.T) {
	h := widgetsHandler(t)
	rec := do(h, "GET", "/logs", nil, nil)
	if rec.Code != 200 {
		t.Fatalf("GET /logs = %d", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body,
		"<!doctype", "wf-shell--sidebar", "Logs — Webframe</title>", `href="/logs"`, `aria-current="page"`,
		`id="demo-log"`, `role="log"`, `hx-trigger="every 3s"`, `hx-swap="beforeend"`, `data-wf-log-max="200"`, "h-[600px]",
		"Filter lines", "Follow", `role="switch"`,
		"items store seeded with 14 items", `data-level="warn"`, "is protected: deletes are refused",
	)
	if n := strings.Count(body, "wf-log-line"); n != 3 {
		t.Errorf("%d log lines on first render, want 3", n)
	}
	m := cursorRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no poll cursor in the page")
	}
	tail := "/logs/tail?cursor=" + m[1]
	// Nothing new yet: 204 and an empty body.
	rec = do(h, "GET", tail, nil, hx(""))
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("idle tail = %d body %q, want 204 and empty", rec.Code, rec.Body.String())
	}
	// A mutation appends a line; the next poll gets it, the one after not.
	if rec := do(h, "POST", "/items", url.Values{"name": {"Logged Lantern"}, "type": {"accessory"}, "status": {"draft"}}, nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d", rec.Code)
	}
	rec = do(h, "GET", tail, nil, hx(""))
	if rec.Code != 200 {
		t.Fatalf("tail after a create = %d, want 200", rec.Code)
	}
	body = rec.Body.String()
	wantAll(t, body, "wf-log-line", `data-level="info"`, "created item #15 Logged Lantern (accessory, draft)", "<time datetime=")
	wantNone(t, body, "<html", "role=\"log\"", "seeded with 14 items")
	if n := strings.Count(body, "wf-log-line"); n != 1 {
		t.Errorf("%d new lines, want 1", n)
	}
	if rec = do(h, "GET", tail, nil, hx("")); rec.Code != http.StatusNoContent {
		t.Errorf("second poll = %d, want 204", rec.Code)
	}
	// Every mutation logs: update, publish, delete, refused delete.
	do(h, "POST", "/items/15", url.Values{"name": {"Logged Lantern"}, "type": {"accessory"}, "status": {"published"}}, nil)
	do(h, "POST", "/items/publish", url.Values{"ids": {"2"}}, hx("items-results"))
	do(h, "DELETE", "/items/3", nil, nil)
	do(h, "DELETE", "/items/1", nil, nil)
	body = do(h, "GET", tail, nil, hx("")).Body.String()
	inOrder(t, body, "updated item #15", "published 1 item (1 selected)", "deleted item #3 Ring of Hoarded Light", "refused to delete item #1")
	wantAll(t, body, `data-level="warn"`, `data-level="error"`)
	// A second page gets its own cursor and only what comes after it.
	m2 := cursorRe.FindStringSubmatch(do(h, "GET", "/logs", nil, nil).Body.String())
	if m2 == nil || m2[1] == m[1] {
		t.Fatalf("second page cursor = %v, want a fresh one", m2)
	}
	if rec = do(h, "GET", "/logs/tail?cursor="+m2[1], nil, hx("")); rec.Code != http.StatusNoContent {
		t.Errorf("fresh cursor poll = %d, want 204", rec.Code)
	}
	// A cursor the buffer never issued is seated at the newest line.
	if rec = do(h, "GET", "/logs/tail?cursor=stale", nil, hx("")); rec.Code != http.StatusNoContent {
		t.Errorf("unknown cursor = %d, want 204", rec.Code)
	}
	Logs.Append("", "one more")
	wantAll(t, do(h, "GET", "/logs/tail?cursor=stale", nil, hx("")).Body.String(), "one more")
	// No cursor: the whole buffer.
	body = do(h, "GET", "/logs/tail", nil, nil).Body.String()
	wantAll(t, body, "seeded with 14 items", "one more")
	// The ring keeps the last logBufferSize lines.
	for i := 0; i < logBufferSize+5; i++ {
		Logs.Append("", "flood")
	}
	if n := len(Logs.Lines()); n != logBufferSize {
		t.Errorf("buffer holds %d lines, want %d", n, logBufferSize)
	}
}

// TestWidgets_Palette: both shells carry the command palette (the dialog
// once at the end of the body, the trigger in the chrome) listing the pages
// and the seeded items with their types as keywords; the bare auth pages
// carry none.
func TestWidgets_Palette(t *testing.T) {
	h := widgetsHandler(t)
	for _, path := range []string{"/items", "/"} {
		body := do(h, "GET", path, nil, nil).Body.String()
		wantAll(t, body,
			`id="wf-palette"`, `data-wf-palette-trigger="wf-palette"`, `aria-keyshortcuts="Meta+K Control+K"`, "⌘K",
			"Jump to a page or an item…",
			`data-label="Dashboard"`, `data-label="Items"`, `data-label="Browse"`, `data-label="Settings"`, `data-label="Logs"`, `href="/logs"`,
			`data-label="Blade of the Fallen Star"`, `data-keywords="weapon"`, `data-keywords="armor"`, `data-keywords="accessory"`, `href="/items/14"`,
			">Items<", // the group heading
		)
		if n := strings.Count(body, `id="wf-palette"`); n != 1 {
			t.Errorf("GET %s: %d palettes, want 1", path, n)
		}
		if n := strings.Count(body, `data-label="`); n != 19 {
			t.Errorf("GET %s: %d palette items, want 19 (5 pages + 14 items)", path, n)
		}
		// The palette is the last thing before the toast container.
		inOrder(t, body, "</main>", `id="wf-palette"`, `id="toast-container"`)
	}
	// The palette follows the store (App.PaletteFor builds it per render):
	// create an item and the next page lists it.
	rec := do(h, "POST", "/items", url.Values{"name": {"Palette Probe"}, "type": {"weapon"}, "status": {"draft"}, "released": {"2026-01-05"}, "rating": {"3"}}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /items = %d, want 303: %s", rec.Code, rec.Body.String())
	}
	for _, path := range []string{"/items", "/"} {
		body := do(h, "GET", path, nil, nil).Body.String()
		wantAll(t, body, `data-label="Palette Probe"`, `href="/items/15"`)
		if n := strings.Count(body, `data-label="`); n != 20 {
			t.Errorf("GET %s after a create: %d palette items, want 20 (5 pages + 15 items)", path, n)
		}
	}
	for _, path := range []string{"/signin", "/signed-out", "/forbidden"} {
		body := do(h, "GET", path, nil, nil).Body.String()
		wantNone(t, body, `id="wf-palette"`, "data-wf-palette-trigger")
	}
	// A plain htmx body swap carries neither.
	wantNone(t, do(h, "GET", "/items", nil, hx("main")).Body.String(), `id="wf-palette"`, "data-wf-palette-trigger")
}

// TestWidgets_PanicIsStyled500: /api/panic panics in its handler; the
// shell's middleware recovers it and the answer is the styled 500 page in
// the shell with the request id, which is the X-Request-Id sent (or a
// generated one).
func TestWidgets_PanicIsStyled500(t *testing.T) {
	h := widgetsHandler(t)
	rec := do(h, "GET", "/api/panic", nil, map[string]string{"X-Request-Id": "req_panic_test"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("GET /api/panic = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	wantAll(t, body, "<!doctype", "wf-topbar", `>500<`, "Something went wrong", "Request id: req_panic_test", "Go home")
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	// Without the header a request id is still shown (16 hex digits).
	body = do(h, "GET", "/api/panic", nil, nil).Body.String()
	if !regexp.MustCompile(`Request id: [0-9a-f]{16}<`).MatchString(body) {
		t.Errorf("no generated request id in:\n%s", body)
	}
	// Over htmx the page is the body only, still 500.
	rec = do(h, "GET", "/api/panic", nil, hx("main"))
	if rec.Code != 500 {
		t.Errorf("htmx GET /api/panic = %d", rec.Code)
	}
	wantAll(t, rec.Body.String(), "Something went wrong")
	wantNone(t, rec.Body.String(), "<html")
	// The server is still up afterwards.
	if rec := do(h, "GET", "/", nil, nil); rec.Code != 200 {
		t.Errorf("GET / after a panic = %d", rec.Code)
	}
}

// TestWidgets_ShowcaseSections pins each widget section's id and the
// markup its demo relies on.
func TestWidgets_ShowcaseSections(t *testing.T) {
	h := widgetsHandler(t)
	body := do(h, "GET", "/", nil, nil).Body.String()
	wantAll(t, body,
		// Contents links and the new group.
		`href="/#content"`, `id="content"`, `href="#prose"`, `href="#codeblock"`, `href="#logviewer"`, `href="#jsonviewer"`, `href="#diffview"`,
		`href="#tagsinput"`, `href="#keyvalueeditor"`, `href="#filedrop"`, `href="#datetimepicker"`, `href="#rating"`, `href="#commandpalette"`,
		// Prose: the HTML specimen (no class on its elements) and the escaped text.
		`id="prose"`, `id="prose-specimen"`, "wf-prose", "<h2>Typography specimen</h2>", "<table>", "<blockquote>", "<pre><code>func (s *ItemStore) Create",
		`id="prose-text-sample"`, "wf-prose--sm", "Drops from the first vault chest.<br>Sells for 400 gold.", "&lt;script&gt;alert(", "<p>TODO: the second paragraph.",
		// Code block: numbered, wrapped, capped.
		`id="codeblock"`, `id="code-numbered"`, "data-wf-code-numbered", `class="wf-code-line"`, "Copy code", `language-go`, `id="code-wrapped"`, "whitespace-pre-wrap", "<figcaption", `id="code-capped"`, "max-h-[240px]",
		// Log viewer: static lines, every level, the filter, the link to /logs.
		`id="logviewer"`, `id="showcase-log"`, "Filter lines", `data-level="info"`, `data-level="warn"`, `data-level="error"`, "h-[240px]", `href="/logs"`,
		// JSON viewer: the nested value, collapsed past depth 2, copy paths.
		`id="jsonviewer"`, `id="showcase-json"`, `&#34;packs&#34;:`, "$.packs[0].items[0].name", "Starfall Pack", "<details", "flex flex-col pl-4",
		// Diff view: two files, both modes, the added file's badge.
		`id="diffview"`, `id="showcase-diff-inline"`, `data-mode="inline"`, `id="showcase-diff-split"`, `data-mode="side-by-side"`, "items/1/tags.txt", ">Added<", "Status: published", "md:hidden",
		// Form widgets.
		`id="tagsinput"`, `id="showcase-tags-root"`, `data-max="5"`, `data-tag="legendary"`, `id="showcase-tags-options"`, "3/5",
		`id="keyvalueeditor"`, `id="showcase-env"`, `name="env.key"`, `value="REGION"`, "Add variable",
		`id="filedrop"`, `id="showcase-files-root"`, `<input type="file" id="showcase-files"`, `for="showcase-files"`, `accept="image/*"`, "multiple", "Up to 1 MB each",
		`id="datetimepicker"`, `id="showcase-date"`, `type="date"`, `value="2026-09-12"`, `max="2026-12-31"`, `id="showcase-datetime"`, `type="datetime-local"`, `value="2026-09-12T09:30"`, `id="showcase-time"`, `type="time"`, `value="22:00:00"`, `step="15"`,
		`id="rating"`, `aria-label="Small: 3 of 5"`, `aria-label="Medium: 4 of 5"`, `aria-label="Large: 5 of 5"`, `id="showcase-rating-input"`, `name="stars"`, "h-11 w-11", "Your rating",
		// The palette section and the panic link under the error-swap section.
		`id="commandpalette"`, "window.wf.palette.open()", "<kbd>⌘K</kbd>",
		`id="panic"`, `href="/api/panic"`, "Panic in a handler",
	)
	// The specimen's elements carry no class, and the only script tag in
	// the Text sample is escaped text.
	specimen := body[strings.Index(body, `id="prose-specimen"`):strings.Index(body, `id="prose-text"`)]
	if strings.Contains(specimen[strings.Index(specimen, ">")+1:], " class=") {
		t.Errorf("prose specimen content carries a class:\n%s", specimen)
	}
	if n := strings.Count(body, "<script>alert"); n != 0 {
		t.Errorf("an unescaped script tag reached the page %d times", n)
	}
}

// TestWidgets_ShowcaseCharts pins the charts and progress group: its
// contents links, each section's id and the markup each sample relies on,
// the Graph nav link and the md wording of the dashboard section.
func TestWidgets_ShowcaseCharts(t *testing.T) {
	h := widgetsHandler(t)
	body := do(h, "GET", "/", nil, nil).Body.String()
	wantAll(t, body,
		// Contents links and the group.
		`id="charts"`, `href="#areachart"`, `href="#barchart"`, `href="#piechart"`, `href="#progressbar"`, `href="#lifecyclestepper"`, `href="#statcomparison"`,
		// Area chart: two series, so the legend shows; day labels.
		`id="areachart"`, `id="showcase-area"`, `viewBox="0 0 320 180"`, ">Views<", ">Downloads<", ">Mon<",
		// Bar chart: horizontal, and the vertical one folded to four bars.
		`id="barchart"`, `id="showcase-bars"`, ">Weapon<", `id="showcase-bars-vertical"`, ">Starfall<", "Other (2)",
		// Pie chart: the donut and the full pie with their legends.
		`id="piechart"`, `id="showcase-donut"`, "<span>Archived (", `id="showcase-pie"`, "<span>Desktop (64%)</span>",
		// Progress bar: 37 of 50 is 74%, the finished one is full.
		`id="progressbar"`, `id="showcase-progress-upload"`, "Uploading textures", "74%", "width: 74%", `id="showcase-progress-build"`, "width: 100%", `id="showcase-progress-bare"`,
		// Lifecycle stepper: compact and large, every state's tooltip.
		`id="lifecyclestepper"`, `id="showcase-stepper"`, ">Build<", `id="showcase-stepper-large"`, `title="Needs attention"`, `title="Failed"`, `title="Skipped"`,
		// Stat comparison: three directions.
		`id="statcomparison"`, `id="showcase-compare-up"`, "from 1,100", "+12.2%", `id="showcase-compare-down"`, `id="showcase-compare-flat"`, "no change",
		// The Graph nav link and the dashboard section's breakpoint.
		`href="/graph"`, "below the md breakpoint (768px at the default root size)",
	)
	wantNone(t, body, "below 900px")
}
