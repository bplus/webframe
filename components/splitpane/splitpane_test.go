package splitpane

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
	"github.com/bplus/webframe/components/badge"
)

func raw(s string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write([]byte(s))
		return err
	})
}

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := SplitPane(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func wantAll(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func wantNone(t *testing.T, out string, bads ...string) {
	t.Helper()
	for _, b := range bads {
		if strings.Contains(out, b) {
			t.Errorf("unexpected %q in:\n%s", b, out)
		}
	}
}

// region returns the opening tag of the element with the given id.
func region(t *testing.T, out, id string) string {
	t.Helper()
	i := strings.Index(out, `id="`+id+`"`)
	if i < 0 {
		t.Fatalf("no element with id %q in:\n%s", id, out)
	}
	start := strings.LastIndex(out[:i], "<")
	end := strings.Index(out[i:], ">")
	return out[start : i+end+1]
}

var listProps = Props{
	ID:       "mail",
	List:     raw("<ul><li>LISTROW</li></ul>"),
	BackHref: "/mail",
}

func detailProps() Props {
	p := listProps
	p.Detail = raw("<p>DETAILBODY</p>")
	return p
}

func TestSplitPane_ListOnly(t *testing.T) {
	out := render(t, listProps)
	wantAll(t, out, `id="mail"`, "LISTROW", `id="mail-detail"`, "md:flex")
	// The detail region is empty and hidden on a phone.
	detail := region(t, out, "mail-detail")
	wantAll(t, detail, "<section", "hidden", "md:block", "min-w-0", "flex-1")
	wantNone(t, out, "DETAILBODY", ">Back<", "/mail\"")
	// The list column is shown on a phone (no hidden class on it).
	list := out[strings.Index(out, `id="mail"`):strings.Index(out, "LISTROW")]
	wantNone(t, list, "hidden")
	wantAll(t, list, "md:w-[360px]", "md:shrink-0", "md:sticky", "md:top-[var(--wf-sticky-top,0px)]",
		"md:max-h-[calc(100dvh-var(--wf-sticky-top,0px))]", "md:overflow-y-auto", "md:border-r", "md:self-start")
}

func TestSplitPane_WithDetail(t *testing.T) {
	out := render(t, detailProps())
	wantAll(t, out, "LISTROW", "DETAILBODY", `href="/mail"`, "<span>Back</span>", `<path d="m12 19-7-7 7-7" />`)
	// The list hides on a phone; the detail region shows.
	list := out[strings.Index(out, `id="mail"`):strings.Index(out, "LISTROW")]
	wantAll(t, list, "hidden", "md:block")
	detail := region(t, out, "mail-detail")
	wantNone(t, detail, "hidden")
	// The back link is phone-only and sits before the detail.
	back := strings.Index(out, "<span>Back</span>")
	body := strings.Index(out, "DETAILBODY")
	if back < 0 || body < 0 || back > body {
		t.Errorf("back link should precede the detail body:\n%s", out)
	}
	wantAll(t, out[strings.Index(out, `id="mail-detail"`):back], "md:hidden")
}

func TestSplitPane_BackLabelAndTitle(t *testing.T) {
	p := detailProps()
	p.BackLabel = "All mail"
	p.Title = "Message"
	out := render(t, p)
	wantAll(t, out, "<span>All mail</span>", `aria-label="Message"`)
	wantNone(t, out, "<span>Back</span>")
	out = render(t, detailProps())
	wantNone(t, out, "aria-label")
}

func TestSplitPane_Widths(t *testing.T) {
	for w, class := range map[Width]string{WidthSm: "md:w-[280px]", WidthMd: "md:w-[360px]", WidthLg: "md:w-[440px]"} {
		p := listProps
		p.ListWidth = w
		wantAll(t, render(t, p), class)
	}
}

func TestSplitPane_DefaultsAndRoot(t *testing.T) {
	out := render(t, Props{})
	wantAll(t, out, `id="splitpane"`, `id="splitpane-detail"`)
	p := Props{ID: "x", DetailID: "pane-body", Class: "mt-8", Attributes: templ.Attributes{"data-k": "v"}}
	out = render(t, p)
	wantAll(t, out, `id="pane-body"`, "mt-8", `data-k="v"`)
	wantNone(t, out, "x-detail")
}

func TestDetailTarget(t *testing.T) {
	if got := DetailTarget(Props{}); got != "#splitpane-detail" {
		t.Errorf("default target %q", got)
	}
	if got := DetailTarget(Props{ID: "mail"}); got != "#mail-detail" {
		t.Errorf("derived target %q", got)
	}
	if got := DetailTarget(Props{ID: "mail", DetailID: "d"}); got != "#d" {
		t.Errorf("explicit target %q", got)
	}
}

// TestLinkAttrs: the htmx attributes only; href is the caller's (an
// action.Action renders its own, and List sets it on the anchor).
func TestLinkAttrs(t *testing.T) {
	attrs := LinkAttrs(Props{ID: "mail"}, "/mail/42")
	want := map[string]string{
		"data-wf-splitpane-link": "true",
		"hx-boost":               "false",
		"hx-target":              "#mail-detail",
		"hx-push-url":            "true",
	}
	if len(attrs) != len(want) {
		t.Errorf("attrs = %v, want exactly %v", attrs, want)
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("%s = %v, want %q", k, attrs[k], v)
		}
	}
	if _, ok := attrs["href"]; ok {
		t.Errorf("LinkAttrs includes href; the caller sets it: %v", attrs)
	}
}

// --- List ---

func renderList(t *testing.T, p Props, items []ListItem) string {
	t.Helper()
	var buf bytes.Buffer
	if err := List(p, items).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render list: %v", err)
	}
	return buf.String()
}

// anchor returns the <a ...>...</a> element whose href is href.
func anchor(t *testing.T, out, href string) string {
	t.Helper()
	i := strings.Index(out, `href="`+href+`"`)
	if i < 0 {
		t.Fatalf("no link to %q in:\n%s", href, out)
	}
	start := strings.LastIndex(out[:i], "<a")
	end := strings.Index(out[i:], "</a>")
	return out[start : i+end+len("</a>")]
}

var mailItems = []ListItem{
	{Href: "/mail/1", Title: "Quarterly numbers", Meta: "Finance"},
	{Href: "/mail/2", Title: "Re: the long subject line that will not fit on one row", Meta: "Ada", Selected: true},
	{Href: "/mail/3", Title: "No meta"},
}

func TestList_LinksCarryHrefAndLinkAttrs(t *testing.T) {
	out := renderList(t, listProps, mailItems)
	wantAll(t, out, `<nav aria-label="Items">`, `<ul id="mail-list"`, "</ul></nav>")
	if n := strings.Count(out, "<li>"); n != 3 {
		t.Errorf("%d rows, want 3:\n%s", n, out)
	}
	for _, it := range mailItems {
		a := anchor(t, out, it.Href)
		wantAll(t, a,
			`href="`+it.Href+`"`, `data-wf-splitpane-link="true"`, `hx-target="#mail-detail"`, `hx-push-url="true"`,
			`title="`+it.Title+`"`, "font-medium", "truncate", "py-2.5", "hover:bg-accent", it.Title,
		)
		if strings.Count(a, `href="`) != 1 {
			t.Errorf("href emitted %d times in %s", strings.Count(a, `href="`), a)
		}
	}
	// The whole list is in order.
	if !(strings.Index(out, "/mail/1") < strings.Index(out, "/mail/2") && strings.Index(out, "/mail/2") < strings.Index(out, "/mail/3")) {
		t.Errorf("rows out of order:\n%s", out)
	}
}

func TestList_SelectedState(t *testing.T) {
	out := renderList(t, listProps, mailItems)
	if n := strings.Count(out, `aria-current="page"`); n != 1 {
		t.Errorf("aria-current count = %d, want 1:\n%s", n, out)
	}
	if n := strings.Count(out, "bg-muted"); n != 1 {
		t.Errorf("bg-muted count = %d, want 1:\n%s", n, out)
	}
	sel := anchor(t, out, "/mail/2")
	wantAll(t, sel, `aria-current="page"`, "bg-muted")
	wantNone(t, anchor(t, out, "/mail/1"), "aria-current", "bg-muted")
	// Nothing selected: no current marker at all.
	wantNone(t, renderList(t, listProps, mailItems[:1]), "aria-current", "bg-muted")
}

func TestList_MetaAndBadge(t *testing.T) {
	out := renderList(t, listProps, mailItems)
	a := anchor(t, out, "/mail/1")
	wantAll(t, a, "text-xs", "text-muted-foreground", ">Finance<")
	// A row without Meta has no second line.
	wantNone(t, anchor(t, out, "/mail/3"), "text-muted-foreground")
	// With Badge, Meta renders inside the badge beside the title, not as the
	// muted line.
	items := []ListItem{{Href: "/mail/9", Title: "Build", Meta: "Failed", Badge: badge.StatusPtr("failed")}}
	a = anchor(t, renderList(t, listProps, items), "/mail/9")
	wantAll(t, a, ">Failed<", "bg-destructive", "font-medium", "Build")
	wantNone(t, a, "text-muted-foreground")
}

// TestList_EmptyKeepsNav: no items renders the nav with an empty <ul>, so
// swapping an emptied list keeps the landmark and its label.
func TestList_EmptyKeepsNav(t *testing.T) {
	for name, items := range map[string][]ListItem{"nil": nil, "empty": {}} {
		out := renderList(t, listProps, items)
		wantAll(t, out, `<nav aria-label="Items">`, `<ul id="mail-list"`, "</ul></nav>")
		wantNone(t, out, "<li", "<a")
		if n := strings.Count(out, "<ul"); n != 1 {
			t.Errorf("%s: %d <ul>, want 1", name, n)
		}
	}
	// The list id follows the pane id.
	wantAll(t, renderList(t, Props{}, nil), `<ul id="splitpane-list"`)
}

// --- Respond ---

func testApp() *appshell.App {
	return &appshell.App{Name: "Demo", Nav: []appshell.NavItem{{Label: "Mail", Href: "/mail"}}}
}

func serve(t *testing.T, path string, p Props, headers map[string]string) (*httptest.ResponseRecorder, error) {
	t.Helper()
	var respondErr error
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respondErr = Respond(w, r, "Mail", p)
	}))
	req := httptest.NewRequest("GET", path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, respondErr
}

func TestRespond_TargetedDetail(t *testing.T) {
	// A list click from md up: only the detail region's content.
	rec, err := serve(t, "/mail/42", detailProps(), map[string]string{"HX-Request": "true", "HX-Target": "mail-detail"})
	if err != nil {
		t.Fatal(err)
	}
	out := rec.Body.String()
	// The detail content, then the list column out of band (so the
	// selected item follows the pick); never the detail region itself or
	// the document.
	wantAll(t, out, "DETAILBODY", "<span>Back</span>", `<div id="mail-listcol" hx-swap-oob="true">`, "LISTROW")
	wantNone(t, out, `id="mail-detail"`, "<html", "<body")
	if strings.Index(out, "DETAILBODY") > strings.Index(out, "hx-swap-oob") {
		t.Errorf("detail content must come before the oob list:\n%s", out)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type %q", ct)
	}
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Target") {
		t.Errorf("Vary %q lacks HX-Target", v)
	}
	// With a "#" prefix too.
	rec, _ = serve(t, "/mail/42", detailProps(), map[string]string{"HX-Request": "true", "HX-Target": "#mail-detail"})
	wantAll(t, rec.Body.String(), "DETAILBODY", "hx-swap-oob")
	wantNone(t, rec.Body.String(), `id="mail-detail"`)
}

func TestRespond_OtherTargetGetsPane(t *testing.T) {
	rec, err := serve(t, "/mail/42", detailProps(), map[string]string{"HX-Request": "true", "HX-Target": "main"})
	if err != nil {
		t.Fatal(err)
	}
	out := rec.Body.String()
	wantAll(t, out, "LISTROW", "DETAILBODY", `id="mail-detail"`)
	wantNone(t, out, "<html", "<body")
}

func TestRespond_FullDocument(t *testing.T) {
	cases := map[string]map[string]string{
		"plain":           nil,
		"boosted":         {"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "mail-detail"},
		"history-restore": {"HX-Request": "true", "HX-History-Restore-Request": "true", "HX-Target": "mail-detail"},
	}
	for name, headers := range cases {
		t.Run(name, func(t *testing.T) {
			rec, err := serve(t, "/mail/42", detailProps(), headers)
			if err != nil {
				t.Fatal(err)
			}
			out := rec.Body.String()
			wantAll(t, out, "<html", "<body", "LISTROW", "DETAILBODY", `id="mail-detail"`, "Mail — Demo")
		})
	}
	// The list route, same shape, no detail.
	rec, err := serve(t, "/mail", listProps, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := rec.Body.String()
	wantAll(t, out, "<html", "LISTROW", `id="mail-detail"`)
	wantNone(t, out, "DETAILBODY", "<span>Back</span>")
}

func TestRespond_NoApp(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/mail", nil)
	err := Respond(rec, req, "Mail", listProps)
	if err == nil {
		t.Fatal("Respond without Middleware should fail")
	}
	if !strings.Contains(err.Error(), "Middleware") {
		t.Errorf("error %q does not name Middleware", err)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("wrote %d bytes without an App", rec.Body.Len())
	}
}
