package filedrop

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

// boolAttr matches a bare boolean attribute.
func boolAttr(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s` + name + `[\s/>]`)
}

// inputTag returns the <input type="file"> element of out.
func inputTag(t *testing.T, out string) string {
	t.Helper()
	m := regexp.MustCompile(`<input[^>]*type="file"[^>]*>`).FindString(out)
	if m == "" {
		t.Fatalf("no file input in:\n%s", out)
	}
	return m
}

func TestFileDrop_InputAttributes(t *testing.T) {
	out := render(t, FileDrop(Props{
		ID: "art", Name: "artwork", Accept: "image/*,.pdf", Multiple: true, Required: true, Disabled: true,
		Attributes: templ.Attributes{"aria-describedby": "art-help"},
	}))
	in := inputTag(t, out)
	mustContain(t, in, `id="art"`, `name="artwork"`, `accept="image/*,.pdf"`, `class="sr-only"`, `x-ref="input"`, `x-on:change="accept($event.target.files)"`, `aria-describedby="art-help"`)
	for _, attr := range []string{"multiple", "required", "disabled"} {
		if !boolAttr(attr).MatchString(in) {
			t.Errorf("input missing %s:\n%s", attr, in)
		}
	}
	mustContain(t, out, `id="art-root"`, `for="art"`, "cursor-not-allowed", "disabled: true")
}

func TestFileDrop_Defaults(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "doc"}))
	in := inputTag(t, out)
	mustContain(t, in, `id="filedrop-doc"`, `type="file"`)
	for _, attr := range []string{"multiple", "required", "disabled"} {
		if boolAttr(attr).MatchString(in) {
			t.Errorf("input has unexpected %s:\n%s", attr, in)
		}
	}
	mustNotContain(t, in, "accept=")
	mustContain(t, out, `id="filedrop-doc-root"`, "Drop a file here or choose", "cursor-pointer", "max: 0", "multiple: false", "preview: false", "disabled: false")
	mustNotContain(t, out, "Up to ")
	multi := render(t, FileDrop(Props{Name: "docs", Multiple: true}))
	mustContain(t, multi, "Drop files here or choose")
}

func TestFileDrop_HiddenInputInsideLabelZone(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "doc"}))
	label := regexp.MustCompile(`(?s)<label[^>]*>.*?</label>`).FindString(out)
	if label == "" {
		t.Fatalf("no label zone:\n%s", out)
	}
	mustContain(t, label, `type="file"`, "border-dashed", `x-on:drop.prevent="drop($event)"`, `x-on:dragover.prevent`, `x-bind:data-over="over"`, "data-over:border-primary", "has-focus-visible:ring-[3px]", "has-aria-invalid:border-destructive", `<svg`, "pointer-events-none")
}

func TestFileDrop_ListContainer(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "doc"}))
	mustContain(t, out, `<ul role="list"`, `x-show="files.length > 0"`, "x-cloak", `x-for="(f, i) in files"`, `x-text="f.name"`, `x-text="size(f.size)"`, `x-on:click="remove(i)"`, `x-bind:aria-label="'Remove ' + f.name"`, "truncate")
}

func TestFileDrop_MaxSizeMessageSlot(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "doc", MaxSize: 2 << 20}))
	mustContain(t, out, `<p role="alert" class="text-sm text-destructive" x-show="error" x-text="error" x-cloak></p>`, "max: 2097152", `maxLabel: &#39;2 MB&#39;`, "Up to 2 MB")
	multi := render(t, FileDrop(Props{Name: "doc", MaxSize: 512 << 10, Multiple: true}))
	mustContain(t, multi, "Up to 512 KB each")
	custom := render(t, FileDrop(Props{Name: "doc", MaxSize: 1 << 20, Label: "Add artwork", Hint: "PNG or JPEG"}))
	mustContain(t, custom, "Add artwork", "PNG or JPEG")
	mustNotContain(t, custom, "Up to ")
}

func TestFileDrop_PreviewFlag(t *testing.T) {
	with := render(t, FileDrop(Props{Name: "doc", Preview: true}))
	mustContain(t, with, "preview: true", `<template x-if="f.url">`, `x-bind:src="f.url"`, "object-cover")
	without := render(t, FileDrop(Props{Name: "doc"}))
	mustContain(t, without, "preview: false")
	mustNotContain(t, without, "<img", `x-if="f.url"`)
}

func TestFileDrop_AlpineDataIsOneAttribute(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "doc"}))
	if got := strings.Count(out, "x-data="); got != 1 {
		t.Errorf("want one x-data root, got %d", got)
	}
	mustContain(t, out, "new DataTransfer()", "URL.createObjectURL", "URL.revokeObjectURL")
}

func TestFormatSize(t *testing.T) {
	for _, tc := range []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{2 << 20, "2 MB"},
		{(5 << 20) + (300 << 10), "5.3 MB"},
		{3 << 30, "3 GB"},
	} {
		if got := FormatSize(tc.n); got != tc.want {
			t.Errorf("FormatSize(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

// TestControlID: ID names the file input itself, so ControlID is ID.
func TestControlID(t *testing.T) {
	if got := (Props{ID: "x", Name: "n"}).ControlID(); got != "x" {
		t.Errorf("ID wins: got %q", got)
	}
	if got := (Props{Name: "n"}).ControlID(); got != "filedrop-n" {
		t.Errorf("derived from Name: got %q", got)
	}
	if got := (Props{}).ControlID(); !strings.HasPrefix(got, "id-") {
		t.Errorf("neither: got %q", got)
	}
}

// TestFileDrop_IDNamesInput: the element with id=ID is the
// <input type="file">; the Alpine root is ID+"-root".
func TestFileDrop_IDNamesInput(t *testing.T) {
	out := render(t, FileDrop(Props{ID: "art", Name: "artwork"}))
	mustContain(t, inputTag(t, out), `id="art"`)
	mustContain(t, out, `<div id="art-root"`)
	if strings.Count(out, `id="art"`) != 1 {
		t.Errorf("id art must be on the input only:\n%s", out)
	}
}

// TestFileDrop_ChooseAgain: the line shows only with ChooseAgain, hides
// once files are picked, names the plural with Multiple, and the input's
// aria-describedby names it ahead of the caller's ids.
func TestFileDrop_ChooseAgain(t *testing.T) {
	off := render(t, FileDrop(Props{ID: "art"}))
	mustNotContain(t, off, "Choose the file again", `id="art-again"`)
	on := render(t, FileDrop(Props{ID: "art", ChooseAgain: true, Attributes: templ.Attributes{"aria-describedby": "art-help"}}))
	mustContain(t, on, `<p id="art-again"`, "Choose the file again", `x-show="files.length === 0"`)
	mustContain(t, inputTag(t, on), `aria-describedby="art-again art-help"`)
	many := render(t, FileDrop(Props{ID: "art", Multiple: true, ChooseAgain: true}))
	mustContain(t, many, "Choose the files again")
	mustContain(t, inputTag(t, many), `aria-describedby="art-again"`)
}

// TestFileDrop_RemoveTarget: the remove button is a 44px square (size-11)
// with a negative margin so the row keeps its height.
func TestFileDrop_RemoveTarget(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "f"}))
	btn := regexp.MustCompile(`<button type="button" class="[^"]*"`).FindString(out)
	mustContain(t, btn, "size-11", "-my-2.5")
	mustNotContain(t, btn, " p-1 ")
}

// TestFileDrop_RandomIDPerRender: with neither ID nor Name, two drop zones
// get distinct ids and, within one render, the root id and the input id
// derive from the same stem.
func TestFileDrop_RandomIDPerRender(t *testing.T) {
	rootAttr := regexp.MustCompile(`<div id="(id-[^"]+)-root"`)
	inputAttr := regexp.MustCompile(`<input type="file" id="(id-[^"]+)"`)
	a := render(t, FileDrop(Props{}))
	b := render(t, FileDrop(Props{}))
	ra, rb := rootAttr.FindStringSubmatch(a), rootAttr.FindStringSubmatch(b)
	ia := inputAttr.FindStringSubmatch(a)
	if ra == nil || rb == nil || ia == nil {
		t.Fatalf("no random ids:\n%s\n%s", a, b)
	}
	if ra[1] == rb[1] {
		t.Errorf("two unnamed drop zones share id %q", ra[1])
	}
	if ia[1] != ra[1] {
		t.Errorf("input id %q and root id %q-root do not share a stem", ia[1], ra[1])
	}
	if !strings.Contains(a, `for="`+ra[1]+`"`) {
		t.Errorf("label for does not point at the input:\n%s", a)
	}
}

// TestFileDrop_DestroyRevokesURLs: the Alpine data has a destroy() that
// revokes the object URLs, so a swapped-out drop zone leaks none.
func TestFileDrop_DestroyRevokesURLs(t *testing.T) {
	out := render(t, FileDrop(Props{Name: "f", Preview: true}))
	mustContain(t, out, "destroy() {", "URL.revokeObjectURL(f.url)")
	if got := strings.Count(out, "destroy()"); got != 1 {
		t.Errorf("want destroy() once in x-data, got %d", got)
	}
}
