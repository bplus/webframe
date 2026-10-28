package listquery

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

var itemOpts = Options{
	DefaultSort:  "name",
	AllowedSorts: []string{"name", "created"},
	FilterKeys:   []string{"status", "owner"},
}

func values(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("ParseQuery(%q): %v", raw, err)
	}
	return v
}

func TestParse_Defaults(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		want Spec
	}{
		{
			name: "zero options",
			opts: Options{},
			want: Spec{Page: 1, PageSize: 25, Dir: Asc},
		},
		{
			name: "default sort and dir",
			opts: Options{DefaultSort: "name", DefaultDir: Desc, DefaultPageSize: 10},
			want: Spec{Sort: "name", Dir: Desc, Page: 1, PageSize: 10},
		},
		{
			name: "default page size above max clamps to max",
			opts: Options{DefaultPageSize: 500, MaxPageSize: 50},
			want: Spec{Dir: Asc, Page: 1, PageSize: 50},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseValues(url.Values{}, tc.opts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseValues(empty) = %+v, want %+v", got, tc.want)
			}
		})
	}
	// Parse reads the request's query.
	req := httptest.NewRequest("GET", "/items?page=2", nil)
	if got := Parse(req, Options{}); got.Page != 2 {
		t.Errorf("Parse page = %d, want 2", got.Page)
	}
}

func TestParse_Hostile(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  Spec
	}{
		{
			name:  "review focus 1",
			query: "sort=evil&dir=sideways&page=-3&per=100000&q=%3Cb%3E",
			want:  Spec{Sort: "name", Dir: Asc, Page: 1, PageSize: 100, Search: "<b>"},
		},
		{
			name:  "non-numeric page and size",
			query: "page=abc&per=1e9",
			want:  Spec{Sort: "name", Dir: Asc, Page: 1, PageSize: 25},
		},
		{
			name:  "zero and negative size fall back to default",
			query: "per=0",
			want:  Spec{Sort: "name", Dir: Asc, Page: 1, PageSize: 25},
		},
		{
			name:  "huge page parses (Apply clamps later)",
			query: "page=99999999999999999999",
			want:  Spec{Sort: "name", Dir: Asc, Page: 1, PageSize: 25},
		},
		{
			name:  "allowed sort and dir pass",
			query: "sort=created&dir=desc&page=4&per=50",
			want:  Spec{Sort: "created", Dir: Desc, Page: 4, PageSize: 50},
		},
		{
			name:  "repeated scalars take the first",
			query: "sort=created&sort=evil&page=2&page=-1",
			want:  Spec{Sort: "created", Dir: Asc, Page: 2, PageSize: 25},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseValues(values(t, tc.query), itemOpts)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseValues(%q) = %+v, want %+v", tc.query, got, tc.want)
			}
		})
	}
	// Nil values must not panic.
	if got := ParseValues(nil, itemOpts); got.Page != 1 {
		t.Errorf("ParseValues(nil).Page = %d, want 1", got.Page)
	}
	// With no AllowedSorts, only the default sort is known.
	got := ParseValues(values(t, "sort=created"), Options{DefaultSort: "name"})
	if got.Sort != "name" {
		t.Errorf("sort without AllowedSorts = %q, want default", got.Sort)
	}
}

func TestParse_FilterKeysOnly(t *testing.T) {
	got := ParseValues(values(t, "status=draft&status=published&evil=1&owner="), itemOpts)
	want := map[string][]string{"status": {"draft", "published"}}
	if !reflect.DeepEqual(got.Filters, want) {
		t.Errorf("Filters = %v, want %v", got.Filters, want)
	}
	if got.Filter("status") != "draft" {
		t.Errorf("Filter(status) = %q, want draft", got.Filter("status"))
	}
	if got.Filter("evil") != "" || got.Filter("owner") != "" {
		t.Error("unlisted or empty filter leaked")
	}
	if !got.HasFilters() {
		t.Error("HasFilters = false, want true")
	}
	if ParseValues(values(t, "evil=1"), itemOpts).HasFilters() {
		t.Error("HasFilters = true for an unlisted key")
	}
	if ParseValues(values(t, "q=x"), itemOpts).HasFilters() != true {
		t.Error("HasFilters = false with a search")
	}
}

func TestParse_Prefix(t *testing.T) {
	opts := itemOpts
	opts.Prefix = "items"
	raw := "items.sort=name&sort=created&items.page=3&page=7&items.q=inner&q=outer&items.status=draft&status=published"
	in := ParseValues(values(t, raw), opts)
	if in.Sort != "name" || in.Page != 3 || in.Search != "inner" || in.Filter("status") != "draft" {
		t.Errorf("prefixed spec read unprefixed params: %+v", in)
	}
	if in.Param("sort") != "items.sort" {
		t.Errorf("Param(sort) = %q, want items.sort", in.Param("sort"))
	}
	out := ParseValues(values(t, raw), itemOpts)
	if out.Sort != "created" || out.Page != 7 || out.Search != "outer" || out.Filter("status") != "published" {
		t.Errorf("unprefixed spec read prefixed params: %+v", out)
	}
	if out.Param("sort") != "sort" {
		t.Errorf("Param(sort) = %q, want sort", out.Param("sort"))
	}
	// The prefix survives every derived URL.
	u := in.URL("/x", WithPage(2))
	if !strings.Contains(u, "items.page=2") || strings.Contains(u, "&page=") || strings.HasPrefix(u, "/x?page=") {
		t.Errorf("derived URL lost the prefix: %q", u)
	}
}

func TestURL_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		spec Spec
	}{
		{
			name: "ampersand in search, space in filter",
			opts: itemOpts,
			spec: Spec{Sort: "created", Dir: Desc, Page: 3, PageSize: 50, Search: "a&b",
				Filters: map[string][]string{"status": {"in review", "draft"}, "owner": {"ada"}}},
		},
		{
			name: "prefixed",
			opts: func() Options { o := itemOpts; o.Prefix = "items"; return o }(),
			spec: Spec{Sort: "name", Dir: Asc, Page: 2, PageSize: 25, Search: "x=y?z#w",
				Filters: map[string][]string{"status": {"a+b", "c%d"}}, prefix: "items"},
		},
		{
			name: "defaults only",
			opts: itemOpts,
			spec: Spec{Sort: "name", Dir: Asc, Page: 1, PageSize: 25},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := tc.spec.URL("/items")
			parsed, err := url.Parse(u)
			if err != nil {
				t.Fatalf("URL produced unparsable %q: %v", u, err)
			}
			got := ParseValues(parsed.Query(), tc.opts)
			if !reflect.DeepEqual(got, tc.spec) {
				t.Errorf("round trip of %q\n got %+v\nwant %+v", u, got, tc.spec)
			}
		})
	}
	// Raw values never appear unescaped.
	s := Spec{Search: "a&b"}
	if u := s.URL("/items"); !strings.Contains(u, "q=a%26b") {
		t.Errorf("URL = %q, want escaped ampersand", u)
	}
}

func TestURL_OmitsZero(t *testing.T) {
	s := Spec{Sort: "name", Dir: Asc, Page: 1, PageSize: 25}
	u := s.URL("/items")
	for _, bad := range []string{"page=", "q=", "status="} {
		if strings.Contains(u, bad) {
			t.Errorf("URL %q contains %q", u, bad)
		}
	}
	if !strings.Contains(u, "sort=name") || !strings.Contains(u, "dir=asc") {
		t.Errorf("URL %q lacks sort/dir", u)
	}
	if u := (Spec{}).URL("/items"); u != "/items" {
		t.Errorf("zero spec URL = %q, want bare base", u)
	}
	// A base with its own query is extended, not broken.
	if u := (Spec{Page: 2}).URL("/items?tab=all"); u != "/items?tab=all&page=2" {
		t.Errorf("URL with existing query = %q", u)
	}
	// dir without a sort is noise and is omitted.
	if u := (Spec{Dir: Desc}).URL("/items"); u != "/items" {
		t.Errorf("dir without sort = %q, want bare base", u)
	}
	// Empty filter values are omitted, including a key with only empty strings.
	s = Spec{Filters: map[string][]string{"status": {""}, "owner": {}}}
	if u := s.URL("/items"); u != "/items" {
		t.Errorf("empty filters URL = %q", u)
	}
}

func TestWithSort_Flips(t *testing.T) {
	base := Spec{Sort: "name", Dir: Asc, Page: 4, PageSize: 25}
	q := func(u string) url.Values {
		p, err := url.Parse(u)
		if err != nil {
			t.Fatal(err)
		}
		return p.Query()
	}
	v := q(base.URL("/items", WithSort("name")))
	if v.Get("sort") != "name" || v.Get("dir") != "desc" || v.Has("page") {
		t.Errorf("first flip = %v, want name desc page reset", v)
	}
	again := ParseValues(v, itemOpts)
	v = q(again.URL("/items", WithSort("name")))
	if v.Get("dir") != "asc" {
		t.Errorf("second flip = %v, want asc", v)
	}
	v = q(base.URL("/items", WithSort("created")))
	if v.Get("sort") != "created" || v.Get("dir") != "asc" || v.Has("page") {
		t.Errorf("new key = %v, want created asc page reset", v)
	}
	if base.Page != 4 || base.Dir != Asc {
		t.Error("URL mutated the receiver")
	}
	if !base.IsSorted("name") || base.IsSorted("created") || (Spec{}).IsSorted("") {
		t.Error("IsSorted wrong")
	}
}

func TestOverrides(t *testing.T) {
	base := Spec{Sort: "name", Dir: Asc, Page: 4, PageSize: 25, Search: "x",
		Filters: map[string][]string{"status": {"draft"}}}
	q := func(u string) url.Values {
		p, err := url.Parse(u)
		if err != nil {
			t.Fatal(err)
		}
		return p.Query()
	}
	v := q(base.URL("/i", WithPage(0)))
	if v.Has("page") {
		t.Errorf("WithPage(0) = %v, want page omitted", v)
	}
	v = q(base.URL("/i", WithPageSize(50)))
	if v.Get("per") != "50" || v.Has("page") {
		t.Errorf("WithPageSize = %v", v)
	}
	v = q(base.URL("/i", WithSearch("new")))
	if v.Get("q") != "new" || v.Has("page") {
		t.Errorf("WithSearch = %v", v)
	}
	v = q(base.URL("/i", WithFilter("status", "published", "archived")))
	if got := v["status"]; !reflect.DeepEqual(got, []string{"published", "archived"}) || v.Has("page") {
		t.Errorf("WithFilter replace = %v", v)
	}
	v = q(base.URL("/i", WithFilter("status")))
	if v.Has("status") {
		t.Errorf("WithFilter delete = %v", v)
	}
	v = q(base.URL("/i", WithFilter("owner", "", "")))
	if v.Has("owner") {
		t.Errorf("WithFilter empty strings = %v", v)
	}
	v = q(base.URL("/i", WithoutFilters()))
	if v.Has("q") || v.Has("status") || v.Get("sort") != "name" || v.Get("per") != "25" || v.Has("page") {
		t.Errorf("WithoutFilters = %v", v)
	}
	// Overrides must not reach back into the receiver's map.
	if base.Filters["status"][0] != "draft" || len(base.Filters) != 1 || base.Search != "x" {
		t.Errorf("receiver mutated: %+v", base)
	}
	// Nil-map receiver survives WithFilter.
	if u := (Spec{}).URL("/i", WithFilter("status", "a")); !strings.Contains(u, "status=a") {
		t.Errorf("WithFilter on nil map = %q", u)
	}
}

func TestSpec_Paging(t *testing.T) {
	s := Spec{Page: 3, PageSize: 10}
	if s.Offset() != 20 || s.Limit() != 10 {
		t.Errorf("Offset/Limit = %d/%d", s.Offset(), s.Limit())
	}
	if s.Pages(0) != 1 || s.Pages(10) != 1 || s.Pages(11) != 2 || s.Pages(30) != 3 || s.Pages(-5) != 1 {
		t.Error("Pages wrong")
	}
	// Zero value behaves as page 1, default size: no divide by zero.
	z := Spec{}
	if z.Offset() != 0 || z.Limit() != 25 || z.Pages(100) != 4 {
		t.Errorf("zero spec Offset/Limit/Pages = %d/%d/%d", z.Offset(), z.Limit(), z.Pages(100))
	}
	n := Spec{Page: -2, PageSize: -9}
	if n.Offset() != 0 || n.Limit() != 25 {
		t.Errorf("negative spec Offset/Limit = %d/%d", n.Offset(), n.Limit())
	}
}

type item struct {
	Name string
	N    int
}

func thirty() []item {
	out := make([]item, 30)
	for i := range out {
		out[i] = item{Name: string(rune('a' + i%26)), N: i}
	}
	return out
}

var sorters = map[string]func(a, b item) bool{
	"n":    func(a, b item) bool { return a.N < b.N },
	"name": func(a, b item) bool { return a.Name < b.Name },
}

func TestApply_SortFilterPage(t *testing.T) {
	items := thirty()
	p := Apply(items, Spec{Sort: "n", Dir: Asc, Page: 3, PageSize: 10}, sorters, nil)
	if len(p.Items) != 10 || p.Total != 30 || p.Items[0].N != 20 || p.Items[9].N != 29 {
		t.Errorf("page 3 = %d items, Total %d, first %v", len(p.Items), p.Total, p.Items[0])
	}
	if p.Spec.Page != 3 {
		t.Errorf("Spec.Page = %d, want 3", p.Spec.Page)
	}
	p = Apply(items, Spec{Sort: "n", Page: 9, PageSize: 10}, sorters, nil)
	if p.Spec.Page != 3 || len(p.Items) != 10 || p.Items[0].N != 20 {
		t.Errorf("page 9 clamp: Spec.Page %d, %d items, first %v", p.Spec.Page, len(p.Items), p.Items[0])
	}
	// Filter: even N only (15 items), page 2 of 10 → 5 items.
	even := func(it item, _ Spec) bool { return it.N%2 == 0 }
	p = Apply(items, Spec{Sort: "n", Page: 2, PageSize: 10}, sorters, even)
	if p.Total != 15 || len(p.Items) != 5 || p.Items[0].N != 20 {
		t.Errorf("filtered = Total %d, %d items, first %v", p.Total, len(p.Items), p.Items)
	}
	// Match may read the Spec.
	p = Apply(items, Spec{Search: "b", PageSize: 10}, sorters,
		func(it item, s Spec) bool { return it.Name == s.Search })
	if p.Total != 2 {
		t.Errorf("search match Total = %d, want 2", p.Total)
	}
	// Unknown sort key and nil sorters leave the input order; the input is not reordered.
	p = Apply(items, Spec{Sort: "zzz", Dir: Desc, PageSize: 5}, sorters, nil)
	if p.Items[0].N != 0 {
		t.Errorf("unknown sort reordered: %v", p.Items[0])
	}
	p = Apply(items, Spec{Sort: "n", Dir: Desc, PageSize: 5}, nil, nil)
	if p.Items[0].N != 0 || items[0].N != 0 {
		t.Error("nil sorters sorted or input mutated")
	}
	// Empty and nil input never panic.
	if p := Apply([]item(nil), Spec{Sort: "n", Page: 5}, sorters, nil); p.Total != 0 || len(p.Items) != 0 || p.Spec.Page != 1 {
		t.Errorf("nil input = %+v", p)
	}
	// Zero-value Spec pages with the default size.
	if p := Apply(items, Spec{}, sorters, nil); len(p.Items) != 25 || p.Spec.PageSize != 25 || p.Spec.Page != 1 {
		t.Errorf("zero spec = %d items, %+v", len(p.Items), p.Spec)
	}
}

func TestApply_Desc(t *testing.T) {
	items := thirty() // names cycle a..z, a..d; N is the input order
	p := Apply(items, Spec{Sort: "name", Dir: Desc, PageSize: 30}, sorters, nil)
	if p.Items[0].Name != "z" || p.Items[29].Name != "a" {
		t.Errorf("desc order wrong: first %v last %v", p.Items[0], p.Items[29])
	}
	// Stable: the four "a".."d" duplicates keep input order (N 0 before N 26).
	for i := 1; i < len(p.Items); i++ {
		prev, cur := p.Items[i-1], p.Items[i]
		if prev.Name < cur.Name {
			t.Fatalf("not descending at %d: %v then %v", i, prev, cur)
		}
		if prev.Name == cur.Name && prev.N > cur.N {
			t.Fatalf("not stable at %d: %v then %v", i, prev, cur)
		}
	}
	if items[0].N != 0 || items[29].N != 29 {
		t.Error("Apply reordered the caller's slice")
	}
}

func TestApply_NilMatch(t *testing.T) {
	items := thirty()
	p := Apply(items, Spec{PageSize: 100}, sorters, nil)
	if p.Total != 30 || len(p.Items) != 30 {
		t.Errorf("nil match = Total %d, %d items", p.Total, len(p.Items))
	}
	for i, it := range p.Items {
		if it.N != i {
			t.Fatalf("nil match reordered at %d: %v", i, it)
		}
	}
}
