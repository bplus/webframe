package pagination

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/liststate"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Pagination(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// spec builds a Spec the way a handler does, through ParseValues, so the
// default page size is recorded and per is omitted from URLs at 25.
func spec(page, size int) liststate.Spec {
	v := url.Values{}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	if size > 0 && size != 25 {
		v.Set("per", strconv.Itoa(size))
	}
	return liststate.ParseValues(v, liststate.Options{})
}

func TestWindow(t *testing.T) {
	cases := []struct {
		page, pages int
		want        []int
	}{
		{1, 1, []int{1}},
		{1, 3, []int{1, 2, 3}},
		{7, 7, []int{1, 2, 3, 4, 5, 6, 7}},
		{1, 20, []int{1, 2, 3, 4, 5, 0, 20}},
		{4, 20, []int{1, 2, 3, 4, 5, 0, 20}},
		{5, 20, []int{1, 0, 4, 5, 6, 0, 20}},
		{16, 20, []int{1, 0, 15, 16, 17, 0, 20}},
		{17, 20, []int{1, 0, 16, 17, 18, 19, 20}},
		{20, 20, []int{1, 0, 16, 17, 18, 19, 20}},
		{0, 0, []int{1}},                          // zero values clamp to a single page
		{99, 20, []int{1, 0, 16, 17, 18, 19, 20}}, // page beyond the end clamps
	}
	for _, c := range cases {
		got := window(c.page, c.pages)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("window(%d, %d) = %v, want %v", c.page, c.pages, got, c.want)
		}
		if len(got) > 7 {
			t.Errorf("window(%d, %d) has %d entries, max 7", c.page, c.pages, len(got))
		}
	}
}

func TestPagination_SummaryAlways(t *testing.T) {
	out := render(t, Props{Total: 3, Spec: spec(1, 25), Base: "/items", Noun: "items"})
	if !strings.Contains(out, "1–3 of 3 items") {
		t.Errorf("missing summary: %s", out)
	}
	if strings.Contains(out, "<a") {
		t.Errorf("single page must not render links: %s", out)
	}
	if strings.Count(out, `aria-disabled="true"`) != 2 {
		t.Errorf("want Previous and Next as disabled spans: %s", out)
	}
	if !strings.Contains(out, `aria-label="Pagination"`) {
		t.Errorf("missing nav label: %s", out)
	}
}

func TestPagination_SummaryNoNoun(t *testing.T) {
	out := render(t, Props{Total: 140, Spec: spec(2, 25), Base: "/items"})
	if !strings.Contains(out, ">26–50 of 140<") {
		t.Errorf("missing bare summary: %s", out)
	}
}

func TestPagination_Links(t *testing.T) {
	out := render(t, Props{Total: 100, Spec: spec(2, 25), Base: "/items", Noun: "items"})
	if !strings.Contains(out, `href="/items?page=3"`) {
		t.Errorf("missing next link: %s", out)
	}
	// Page 1 is the zero value: liststate.Spec.URL omits it, so Previous
	// points at the bare base rather than ?page=1.
	if !strings.Contains(out, `href="/items"`) {
		t.Errorf("missing previous link to base: %s", out)
	}
	if strings.Contains(out, "page=1\"") {
		t.Errorf("page=1 must be omitted from URLs: %s", out)
	}
	if !strings.Contains(out, `<span aria-current="page"`) {
		t.Fatalf("missing aria-current: %s", out)
	}
	i := strings.Index(out, `aria-current="page"`)
	seg := out[i:]
	seg = seg[:strings.Index(seg, "</span>")]
	if !strings.HasSuffix(seg, ">2") {
		t.Errorf("aria-current span should hold page 2: %q", seg)
	}
	if strings.Contains(out, `aria-disabled`) {
		t.Errorf("mid-list page must have live Previous and Next: %s", out)
	}
	if !strings.Contains(out, `href="/items?page=4"`) {
		t.Errorf("missing last page link: %s", out)
	}
	if !strings.Contains(out, "hidden md:flex") && !strings.Contains(out, "md:flex") {
		t.Errorf("page list must be hidden below md: %s", out)
	}
	if strings.Contains(out, "hx-get") {
		t.Errorf("no HxTarget → no hx attributes: %s", out)
	}
}

func TestPagination_Ends(t *testing.T) {
	first := render(t, Props{Total: 100, Spec: spec(1, 25), Base: "/items"})
	if strings.Count(first, `aria-disabled="true"`) != 1 {
		t.Errorf("first page: Previous disabled only: %s", first)
	}
	if !strings.Contains(first, `href="/items?page=2"`) {
		t.Errorf("first page: Next → page 2: %s", first)
	}
	last := render(t, Props{Total: 100, Spec: spec(4, 25), Base: "/items"})
	if strings.Count(last, `aria-disabled="true"`) != 1 {
		t.Errorf("last page: Next disabled only: %s", last)
	}
	if !strings.Contains(last, "76–100 of 100") {
		t.Errorf("last page range: %s", last)
	}
}

func TestPagination_Ellipsis(t *testing.T) {
	out := render(t, Props{Total: 500, Spec: spec(10, 25), Base: "/items"})
	if strings.Count(out, "…") != 2 {
		t.Errorf("want two ellipses for 10 of 20: %s", out)
	}
	for _, n := range []int{9, 11, 20} {
		want := `href="/items?page=` + itoa(n) + `"`
		if !strings.Contains(out, want) {
			t.Errorf("missing %s: %s", want, out)
		}
	}
}

func TestPagination_Hx(t *testing.T) {
	out := render(t, Props{Total: 100, Spec: spec(2, 25), Base: "/items", HxTarget: "#list-results"})
	for _, want := range []string{`hx-get="/items?page=3"`, `hx-target="#list-results"`, `hx-push-url="true"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s: %s", want, out)
		}
	}
}

func TestPagination_Zero(t *testing.T) {
	out := render(t, Props{Total: 0, Spec: liststate.Spec{}, Base: "/items", Noun: "items"})
	if !strings.Contains(out, ">0 items<") {
		t.Errorf("missing zero summary: %s", out)
	}
	if strings.Contains(out, "<a") {
		t.Errorf("zero rows must not render links: %s", out)
	}
	bare := render(t, Props{Base: "/items"})
	if !strings.Contains(bare, ">0<") {
		t.Errorf("missing bare zero summary: %s", bare)
	}
}

func TestPagination_ZeroSpecUsesDefaultSize(t *testing.T) {
	// A zero-value Spec behaves as page 1 with the default size (25).
	out := render(t, Props{Total: 60, Spec: liststate.Spec{}, Base: "/items"})
	if !strings.Contains(out, "1–25 of 60") {
		t.Errorf("zero Spec should page by 25: %s", out)
	}
	if !strings.Contains(out, `href="/items?page=3"`) {
		t.Errorf("zero Spec should yield 3 pages: %s", out)
	}
}

func TestPagination_KeepsListState(t *testing.T) {
	v := url.Values{"page": {"2"}, "per": {"10"}, "sort": {"name"}, "dir": {"desc"}, "q": {"a&b"}}
	sp := liststate.ParseValues(v, liststate.Options{AllowedSorts: []string{"name"}})
	out := render(t, Props{Total: 30, Spec: sp, Base: "/items"})
	if !strings.Contains(out, `href="/items?dir=desc&amp;page=3&amp;per=10&amp;q=a%26b&amp;sort=name"`) {
		t.Errorf("links must carry the full Spec, escaped: %s", out)
	}
}

func TestPagination_IDClassAttributes(t *testing.T) {
	out := render(t, Props{ID: "pg", Class: "mt-4", Attributes: map[string]any{"data-x": "1"}, Base: "/items"})
	for _, want := range []string{`id="pg"`, "mt-4", `data-x="1"`, "<nav"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s: %s", want, out)
		}
	}
}

func TestPagination_BaseFromContext(t *testing.T) {
	// Base "" falls back to the request path appshell.Middleware stored.
	app := &appshell.App{}
	h := app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = Pagination(Props{Total: 100, Spec: spec(2, 25)}).Render(r.Context(), w)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ctx-items?page=2", nil))
	out := rec.Body.String()
	if !strings.Contains(out, `href="/ctx-items?page=3"`) {
		t.Errorf("Base \"\" should fall back to appshell.PathFromContext: %s", out)
	}
	if !strings.Contains(out, `href="/ctx-items"`) {
		t.Errorf("Previous should point at the bare path: %s", out)
	}
}

func TestPagination_NoBaseNoContext(t *testing.T) {
	// Neither Base nor a Middleware path: links are query-only ("?page=3"),
	// which the browser resolves against the current document.
	out := render(t, Props{Total: 100, Spec: spec(2, 25)})
	if !strings.Contains(out, `href="?page=3"`) {
		t.Errorf("want relative query link: %s", out)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
