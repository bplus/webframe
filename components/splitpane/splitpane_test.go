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

func TestLinkAttrs(t *testing.T) {
	attrs := LinkAttrs(Props{ID: "mail"}, "/mail/42")
	want := map[string]string{
		"href":        "/mail/42",
		"hx-get":      "/mail/42",
		"hx-target":   "#mail-detail",
		"hx-push-url": "true",
		"hx-trigger":  "click[window.innerWidth >= 768]",
	}
	if len(attrs) != len(want) {
		t.Errorf("attrs = %v, want exactly %v", attrs, want)
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("%s = %v, want %q", k, attrs[k], v)
		}
	}
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
	wantAll(t, out, "DETAILBODY", "<span>Back</span>")
	wantNone(t, out, "LISTROW", `id="mail-detail"`, "<html", "<body")
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type %q", ct)
	}
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Target") {
		t.Errorf("Vary %q lacks HX-Target", v)
	}
	// With a "#" prefix too.
	rec, _ = serve(t, "/mail/42", detailProps(), map[string]string{"HX-Request": "true", "HX-Target": "#mail-detail"})
	wantNone(t, rec.Body.String(), "LISTROW")
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
