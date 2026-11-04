package tagsinput

import (
	"bytes"
	"context"
	"regexp"
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
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func mustNotContain(t *testing.T, out string, nots ...string) {
	t.Helper()
	for _, not := range nots {
		if strings.Contains(out, not) {
			t.Errorf("output has unexpected %q:\n%s", not, out)
		}
	}
}

// disabledAttr matches a bare disabled attribute (not the disabled: class
// variants).
var disabledAttr = regexp.MustCompile(`\sdisabled(\s|>)`)

// frameTag returns the opening tag of the box around the chips and input
// (the parent of the chip list).
func frameTag(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, `<ul id="t-list"`)
	if i < 0 {
		t.Fatalf("chip list missing:\n%s", out)
	}
	start := strings.LastIndex(out[:i], "<div ")
	return out[start:i]
}

// hasClass reports whether tag's class attribute holds cls as a whole token
// (TwMerge reorders classes, so never assert on a sequence).
func hasClass(tag, cls string) bool {
	return regexp.MustCompile(`(class="|\s)` + regexp.QuoteMeta(cls) + `(\s|")`).MatchString(tag)
}

// textInputTag returns the text input's opening tag.
func textInputTag(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, `<input id="t"`)
	if i < 0 {
		t.Fatalf("text input missing:\n%s", out)
	}
	return out[i : i+strings.Index(out[i:], ">")+1]
}

func TestTagsInput_ServerRenderedChipsAndHiddenJoin(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "tags", Name: "tags", Values: []string{"go", "templ", "htmx"}}))
	mustContain(t, out,
		`<input type="hidden" name="tags" value="go,templ,htmx" x-bind:value="tags.join(sep)">`,
		`<ul id="tags-list" role="list" aria-label="Tags"`,
		`<li data-tag="go"`, `<li data-tag="templ"`, `<li data-tag="htmx"`,
		`<span class="truncate">go</span>`,
		`aria-label="Remove go"`,
		`<template x-for="(tag, i) in tags" :key="tag">`,
		`x-on:click="removeAt(i)"`,
		`id="tags"`,
		`x-model="text"`,
		`x-on:keydown="onKey($event)"`,
		`x-on:blur="commit()"`,
		`aria-describedby="tags-hint"`,
		`<p id="tags-hint"`,
		`data-sep=","`,
		`data-max="0"`,
		`x-data="{`,
	)
	if strings.Count(out, `data-tag=`) != 3 {
		t.Errorf("want 3 server chips, got %d:\n%s", strings.Count(out, `data-tag=`), out)
	}
	// Server chips are in order and before the live template.
	if strings.Index(out, `data-tag="go"`) > strings.Index(out, `data-tag="templ"`) ||
		strings.Index(out, `data-tag="htmx"`) > strings.Index(out, `<template x-for`) {
		t.Errorf("chips out of order:\n%s", out)
	}
	// No cap: no count shown, no datalist.
	mustNotContain(t, out, `x-text="tags.length`, `<datalist`, ` data-disabled>`)
}

func TestTagsInput_CustomSeparator(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "t", Values: []string{"a", "b"}, Separator: ";"}))
	mustContain(t, out, `value="a;b"`, `data-sep=";"`)
}

func TestTagsInput_MaxTagsCountAndCap(t *testing.T) {
	below := render(t, TagsInput(Props{ID: "t", Name: "t", Values: []string{"a", "b"}, MaxTags: 5}))
	mustContain(t, below, `data-max="5"`, `<span x-text="tags.length + '/' + max">2/5</span>`)
	if disabledAttr.MatchString(textInputTag(t, below)) {
		t.Errorf("text input should be enabled below the cap:\n%s", below)
	}

	atCap := render(t, TagsInput(Props{ID: "t", Name: "t", Values: []string{"a", "b", "c"}, MaxTags: 3}))
	mustContain(t, atCap, `3/3`, `x-bind:readonly="atMax"`, `placeholder="Limit reached"`)
	// At the cap the text input is readonly, never disabled (disabling a
	// focused control drops focus and closes a phone's keyboard); the
	// hidden field and the chip buttons are untouched.
	if disabledAttr.MatchString(textInputTag(t, atCap)) {
		t.Errorf("text input must stay enabled (readonly) at the cap:\n%s", atCap)
	}
	if !strings.Contains(textInputTag(t, atCap), " readonly") {
		t.Errorf("text input should be readonly at the cap:\n%s", atCap)
	}
	mustContain(t, atCap, `<input type="hidden" name="t" value="a,b,c" x-bind:value="tags.join(sep)">`)
	mustContain(t, atCap, `aria-label="Remove a">`)
}

func TestTagsInput_Disabled(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "t", Values: []string{"a"}, Disabled: true}))
	mustContain(t, out,
		` data-disabled>`,
		`<input type="hidden" name="t" value="a" x-bind:value="tags.join(sep)" disabled>`,
		`aria-label="Remove a" disabled>`,
	)
	if frame := frameTag(t, out); !hasClass(frame, "cursor-not-allowed") || !hasClass(frame, "opacity-50") {
		t.Errorf("frame should look disabled:\n%s", frame)
	}
	if !disabledAttr.MatchString(textInputTag(t, out)) {
		t.Errorf("text input should be disabled:\n%s", out)
	}
}

func TestTagsInput_Escaping(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "t", Values: []string{`<script>alert("x")</script>`, "a&b"}, Suggestions: []string{`"quoted"`}}))
	mustNotContain(t, out, `<script>alert`)
	mustContain(t, out,
		`data-tag="&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;"`,
		`<span class="truncate">&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;</span>`,
		`value="&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;,a&amp;b"`,
		`<option value="&#34;quoted&#34;"></option>`,
	)
}

func TestTagsInput_Suggestions(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "t", Suggestions: []string{"red", "green"}}))
	mustContain(t, out,
		`list="t-options"`,
		`<datalist id="t-options"><option value="red"></option><option value="green"></option></datalist>`,
	)
}

func TestTagsInput_AttributesMergeOntoInput(t *testing.T) {
	out := render(t, TagsInput(Props{
		ID: "t", Name: "t", HasError: true,
		Attributes: templ.Attributes{"aria-describedby": "t-help t-feedback", "aria-invalid": "true"},
	}))
	mustContain(t, out,
		`aria-describedby="t-hint t-help t-feedback"`,
		`aria-invalid="true"`,
	)
	if frame := frameTag(t, out); !hasClass(frame, "border-destructive") {
		t.Errorf("frame should show the error border:\n%s", frame)
	}
	if strings.Count(out, `aria-describedby=`) != 1 {
		t.Errorf("aria-describedby should appear once:\n%s", out)
	}
}

func TestTagsInput_NoNameNoHiddenField(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Values: []string{"a"}}))
	mustNotContain(t, out, `type="hidden"`)
}

func TestTagsInput_DefaultIDIsGenerated(t *testing.T) {
	out := render(t, TagsInput(Props{Name: "t"}))
	mustContain(t, out, `id="id-`, `-hint"`, `-root"`)
}

func TestTagsInput_ZeroProps(t *testing.T) {
	out := render(t, TagsInput())
	mustContain(t, out, `x-data=`, `role="list"`)
}

// TestTagsInput_ValuesNormalized: Values are trimmed, empties dropped and
// case-insensitive duplicates reduced to the first spelling before
// rendering, so the chips, the hidden joined value and the count agree.
func TestTagsInput_ValuesNormalized(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "t", Values: []string{"a", "A", " b ", ""}, MaxTags: 5}))
	mustContain(t, out, `data-tag="a"`, `data-tag="b"`, `value="a,b"`, "2/5", `aria-label="Remove a"`, `aria-label="Remove b"`)
	mustNotContain(t, out, `data-tag="A"`, `data-tag=" b "`, `data-tag=""`)
	if got := strings.Count(out, "<li data-tag="); got != 2 {
		t.Errorf("want 2 server-rendered chips, got %d:\n%s", got, out)
	}
}

func TestNormalize(t *testing.T) {
	if got := normalize(nil); got != nil {
		t.Errorf("normalize(nil) = %v, want nil", got)
	}
	got := normalize([]string{" x", "y ", "X", "", "  ", "y", "z"})
	want := []string{"x", "y", "z"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("normalize = %q, want %q", got, want)
	}
}

// TestTagsInput_SeparatorKeyCommits: the keydown handler commits on Enter
// and on the separator key read from data-sep (not on a hard-coded comma),
// and the sr-only hint names that key; a multi-character separator leaves
// Enter alone.
func TestTagsInput_SeparatorKeyCommits(t *testing.T) {
	semi := render(t, TagsInput(Props{ID: "t", Name: "t", Separator: ";"}))
	mustContain(t, semi, `data-sep=";"`, `e.key === &#39;Enter&#39; || (this.sep.length === 1 &amp;&amp; e.key === this.sep)`, "Enter or ; adds a tag;")
	mustNotContain(t, semi, `e.key === &#39;,&#39;`, "Enter or comma adds a tag")

	comma := render(t, TagsInput(Props{ID: "t", Name: "t"}))
	mustContain(t, comma, `data-sep=","`, "Enter or comma adds a tag;")

	multi := render(t, TagsInput(Props{ID: "t", Name: "t", Separator: ", "}))
	mustContain(t, multi, `data-sep=", "`, "Enter adds a tag;")
	mustNotContain(t, multi, "Enter or")
}

// TestTagsInput_ListSemantics: the chips are a real flex list (not
// display: contents, which drops list semantics in Safari), the text input
// sits after the list and outside it, and the list is hidden with no tags.
func TestTagsInput_ListSemantics(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "tags", Values: []string{"go"}}))
	i := strings.Index(out, `<ul id="t-list"`)
	ul := out[i : i+strings.Index(out[i:], ">")+1]
	mustContain(t, ul, `role="list"`, `aria-label="Tags"`, "flex", "flex-wrap", `x-bind:hidden="tags.length === 0"`)
	mustNotContain(t, ul, "contents", " hidden ")
	end := strings.Index(out, "</ul>")
	if in := strings.Index(out, `<input id="t"`); in < end {
		t.Errorf("the text input must come after the list, outside it:\n%s", out)
	}
	empty := render(t, TagsInput(Props{ID: "t", Name: "tags"}))
	j := strings.Index(empty, `<ul id="t-list"`)
	if !regexp.MustCompile(`\shidden[\s>]`).MatchString(empty[j : j+strings.Index(empty[j:], ">")+1]) {
		t.Errorf("an empty list renders hidden:\n%s", empty)
	}
}

// TestTagsInput_PasteSplitsLines: the text input intercepts paste and
// splits on line breaks as well as the separator (a text input would strip
// the newlines and paste one tag).
func TestTagsInput_PasteSplitsLines(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "tags"}))
	mustContain(t, textInputTag(t, out), `x-on:paste="onPaste($event)"`)
	mustContain(t, out, "onPaste(e) {", "getData(&#39;text&#39;)", "e.preventDefault()", ".join(this.sep)")
}

// TestTagsInput_RemoveTarget: the chip remove button is 44px below md and
// 24px from md up.
func TestTagsInput_RemoveTarget(t *testing.T) {
	out := render(t, TagsInput(Props{ID: "t", Name: "tags", Values: []string{"go"}}))
	btn := regexp.MustCompile(`<button type="button" class="[^"]*"`).FindString(out)
	mustContain(t, btn, "size-11", "md:size-6", "md:-my-1.5")
}
