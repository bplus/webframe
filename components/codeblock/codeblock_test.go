package codeblock_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/codeblock"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestCodeBlock_EscapesCode(t *testing.T) {
	out := render(t, codeblock.CodeBlock(codeblock.Props{Code: `<script>alert("x")</script> && done`}))
	if strings.Contains(out, "<script>") {
		t.Fatalf("code was not escaped:\n%s", out)
	}
	for _, want := range []string{"&lt;script&gt;", "&amp;&amp; done", `<figure class="wf-code`, "<pre", "<code>"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestCodeBlock_ZeroValueRenders(t *testing.T) {
	out := render(t, codeblock.CodeBlock())
	if !strings.Contains(out, "<pre") || strings.Contains(out, "justify-between") {
		t.Errorf("zero Props: want a bare <pre> with no header row:\n%s", out)
	}
}

func TestCodeBlock_LineNumbersCountLines(t *testing.T) {
	out := render(t, codeblock.CodeBlock(codeblock.Props{Code: "one\ntwo\n\nfour\n", LineNumbers: true}))
	if got := strings.Count(out, `<span class="wf-code-line">`); got != 4 {
		t.Errorf("want 4 numbered lines (trailing newline dropped, blank line kept), got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "data-wf-code-numbered") {
		t.Errorf("missing data-wf-code-numbered on the figure:\n%s", out)
	}
	if strings.Contains(out, "</span>\n") {
		t.Errorf("numbered lines must not be separated by newlines (each is display:grid):\n%s", out)
	}
}

func TestCodeBlock_NoLineNumbersKeepsText(t *testing.T) {
	out := render(t, codeblock.CodeBlock(codeblock.Props{Code: "one\ntwo\n"}))
	if strings.Contains(out, "wf-code-line") || strings.Contains(out, "data-wf-code-numbered") {
		t.Errorf("no line spans without LineNumbers:\n%s", out)
	}
	if !strings.Contains(out, "<code>one\ntwo</code>") {
		t.Errorf("text should render verbatim minus one trailing newline:\n%s", out)
	}
}

func TestCodeBlock_WrapClasses(t *testing.T) {
	wrapped := render(t, codeblock.CodeBlock(codeblock.Props{Code: "x", Wrap: true}))
	for _, want := range []string{"whitespace-pre-wrap", "break-words"} {
		if !strings.Contains(wrapped, want) {
			t.Errorf("Wrap: missing %q:\n%s", want, wrapped)
		}
	}
	if strings.Contains(wrapped, `tabindex="0"`) {
		t.Errorf("a wrapped, uncapped block is not scrollable and needs no tabindex:\n%s", wrapped)
	}
	plain := render(t, codeblock.CodeBlock(codeblock.Props{Code: "x"}))
	for _, want := range []string{"overflow-x-auto", "whitespace-pre", `tabindex="0"`} {
		if !strings.Contains(plain, want) {
			t.Errorf("no Wrap: missing %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "whitespace-pre-wrap") {
		t.Errorf("no Wrap must not soft-wrap:\n%s", plain)
	}
}

func TestCodeBlock_MaxHeightClass(t *testing.T) {
	cases := map[codeblock.Height]string{
		codeblock.HeightSm: "max-h-[240px]",
		codeblock.HeightMd: "max-h-[400px]",
		codeblock.HeightLg: "max-h-[600px]",
	}
	for h, want := range cases {
		out := render(t, codeblock.CodeBlock(codeblock.Props{Code: "x", MaxHeight: h}))
		if !strings.Contains(out, want) || !strings.Contains(out, "overflow-y-auto") {
			t.Errorf("%q: missing %q / overflow-y-auto:\n%s", h, want, out)
		}
	}
	none := render(t, codeblock.CodeBlock(codeblock.Props{Code: "x"}))
	if strings.Contains(none, "max-h-") {
		t.Errorf("HeightNone must not cap:\n%s", none)
	}
}

func TestCodeBlock_HeaderLanguageAndCopy(t *testing.T) {
	out := render(t, codeblock.CodeBlock(codeblock.Props{Code: "fmt.Println(\"hi\")", Language: "go", Copy: true}))
	for _, want := range []string{`class="language-go"`, ">go</span>", `aria-label="Copy code"`, `navigator.clipboard.writeText(&#39;fmt.Println(\&#34;hi\&#34;)&#39;)`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	onlyCopy := render(t, codeblock.CodeBlock(codeblock.Props{Code: "x", Copy: true}))
	if !strings.Contains(onlyCopy, "x-on:click") || strings.Contains(onlyCopy, "language-") {
		t.Errorf("Copy alone: header with the button and no language class:\n%s", onlyCopy)
	}
}

func TestCodeBlock_CaptionIDClassAttributes(t *testing.T) {
	out := render(t, codeblock.CodeBlock(codeblock.Props{
		ID: "snippet", Class: "mt-4", Code: "x", Caption: "Listing 1",
		Attributes: templ.Attributes{"data-test": "yes"},
	}))
	for _, want := range []string{`id="snippet"`, "mt-4", `<figcaption class="`, ">Listing 1</figcaption>", `data-test="yes"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	none := render(t, codeblock.CodeBlock(codeblock.Props{Code: "x"}))
	if strings.Contains(none, "<figcaption") {
		t.Errorf("no Caption must render no figcaption:\n%s", none)
	}
}

// TestCodeBlock_CRLFNormalized: Windows line ends count as one line each
// and no carriage return reaches the output.
func TestCodeBlock_CRLFNormalized(t *testing.T) {
	out := render(t, codeblock.CodeBlock(codeblock.Props{Code: "a\r\nb\r\nc\r\n", LineNumbers: true}))
	if got := strings.Count(out, `<span class="wf-code-line">`); got != 3 {
		t.Errorf("want 3 numbered lines, got %d:\n%s", got, out)
	}
	if strings.Contains(out, "\r") {
		t.Errorf("carriage return leaked into the output:\n%q", out)
	}
	plain := render(t, codeblock.CodeBlock(codeblock.Props{Code: "a\r\nb\r\n"}))
	if strings.Contains(plain, "\r") || !strings.Contains(plain, "a\nb") {
		t.Errorf("un-numbered rendering not normalized:\n%q", plain)
	}
}

// TestCodeBlock_GutterDigits: the figure carries --wf-code-digits with the
// digit count of the last line number, only when LineNumbers is set.
func TestCodeBlock_GutterDigits(t *testing.T) {
	long := strings.Repeat("x\n", 1000)
	out := render(t, codeblock.CodeBlock(codeblock.Props{Code: long, LineNumbers: true}))
	if !strings.Contains(out, `style="--wf-code-digits:4;"`) {
		t.Errorf("1000 lines want a 4-digit gutter:\n%s", out[:min(len(out), 400)])
	}
	short := render(t, codeblock.CodeBlock(codeblock.Props{Code: "one\ntwo", LineNumbers: true}))
	if !strings.Contains(short, `style="--wf-code-digits:1;"`) {
		t.Errorf("2 lines want a 1-digit gutter:\n%s", short)
	}
	none := render(t, codeblock.CodeBlock(codeblock.Props{Code: long}))
	if strings.Contains(none, "--wf-code-digits") {
		t.Errorf("no line numbers, no digits property:\n%s", none[:min(len(none), 400)])
	}
}

// TestCodeBlock_StickyGutterOnlyWhenScrolling checks the marker the sticky
// gutter rules in input.css key on: present for numbered lines that scroll
// sideways, absent with Wrap (whose rendering is unchanged) and without
// line numbers.
func TestCodeBlock_StickyGutterOnlyWhenScrolling(t *testing.T) {
	scrolling := render(t, codeblock.CodeBlock(codeblock.Props{Code: "a\nb", LineNumbers: true}))
	if !strings.Contains(scrolling, "data-wf-code-scroll") {
		t.Errorf("numbered, unwrapped block: want data-wf-code-scroll:\n%s", scrolling)
	}
	for name, p := range map[string]codeblock.Props{
		"wrap":       {Code: "a\nb", LineNumbers: true, Wrap: true},
		"no numbers": {Code: "a\nb"},
	} {
		if out := render(t, codeblock.CodeBlock(p)); strings.Contains(out, "data-wf-code-scroll") {
			t.Errorf("%s: unexpected data-wf-code-scroll:\n%s", name, out)
		}
	}
}
