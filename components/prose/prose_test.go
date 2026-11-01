package prose

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in %s", w, out)
		}
	}
}

func TestProse_HTMLVerbatimInsideScope(t *testing.T) {
	out := render(t, Prose(Props{ID: "doc", HTML: `<h2>Title</h2><p>Body with <a href="/x">a link</a> &amp; a <code>call()</code>.</p>`}))
	mustContain(t, out, `id="doc"`, "wf-prose", "max-w-prose", "min-w-0",
		`<h2>Title</h2><p>Body with <a href="/x">a link</a> &amp; a <code>call()</code>.</p>`)
	if !strings.HasPrefix(out, "<div") || !strings.HasSuffix(out, "</div>") {
		t.Errorf("content must sit inside the scope div: %s", out)
	}
}

func TestProse_TextEscapesAndParagraphs(t *testing.T) {
	out := render(t, Prose(Props{Text: "First line\nsecond line\n\n\n  Second para <script>alert(1)</script>\r\n\r\nThird & last"}))
	if n := strings.Count(out, "<p>"); n != 3 {
		t.Errorf("got %d <p>, want 3: %s", n, out)
	}
	mustContain(t, out, "<p>First line<br>second line</p>", "&lt;script&gt;alert(1)&lt;/script&gt;", "Third &amp; last")
	if strings.Contains(out, "<script>") {
		t.Errorf("script tag must be escaped: %s", out)
	}
}

func TestProse_HTMLWinsOverText(t *testing.T) {
	out := render(t, Prose(Props{HTML: "<p>html</p>", Text: "text"}))
	mustContain(t, out, "<p>html</p>")
	if strings.Contains(out, "text") {
		t.Errorf("Text must be ignored when HTML is set: %s", out)
	}
}

func TestProse_EmptyRendersEmptyScope(t *testing.T) {
	out := render(t, Prose(Props{}))
	if strings.Contains(out, "<p>") {
		t.Errorf("no paragraphs expected: %s", out)
	}
	mustContain(t, out, "wf-prose")
}

func TestProse_SizesAndMaxWidths(t *testing.T) {
	cases := []struct {
		p    Props
		want string
		not  []string
	}{
		{Props{}, "max-w-prose", []string{"wf-prose--sm", "wf-prose--lg", "max-w-[100ch]"}},
		{Props{Size: SizeSm}, "wf-prose--sm", []string{"wf-prose--lg"}},
		{Props{Size: SizeLg}, "wf-prose--lg", []string{"wf-prose--sm"}},
		{Props{MaxWidth: MaxWidthWide}, "max-w-[100ch]", []string{"max-w-prose"}},
		{Props{MaxWidth: MaxWidthNone}, "wf-prose", []string{"max-w-"}},
	}
	for _, tc := range cases {
		out := render(t, Prose(tc.p))
		mustContain(t, out, tc.want)
		for _, n := range tc.not {
			if strings.Contains(out, n) {
				t.Errorf("%+v: unexpected %q in %s", tc.p, n, out)
			}
		}
	}
}

func TestProse_ClassAndAttributes(t *testing.T) {
	out := render(t, Prose(Props{Class: "mt-4", Attributes: templ.Attributes{"data-x": "1"}, Text: "x"}))
	mustContain(t, out, "mt-4", `data-x="1"`)
}
