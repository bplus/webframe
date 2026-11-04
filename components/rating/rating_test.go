package rating

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

func TestRating_ReadOnlyAria(t *testing.T) {
	out := render(t, Rating(Props{ReadOnly: true, Value: 3}))
	mustContain(t, out, `role="img"`, `aria-label="3 of 5"`, `id="id-`)
	mustNotContain(t, out, "<fieldset", "<input", "<legend")
	if got := strings.Count(out, "<svg"); got != 5 {
		t.Errorf("want 5 stars, got %d", got)
	}
	if got := strings.Count(out, `class="`+filledClass+`"`); got != 3 {
		t.Errorf("want 3 lit stars, got %d", got)
	}
	if got := strings.Count(out, `class="`+emptyClass+`"`); got != 2 {
		t.Errorf("want 2 empty stars, got %d", got)
	}
	mustContain(t, out, `fill="currentColor"`)
}

func TestRating_ReadOnlyLabelAndMax(t *testing.T) {
	out := render(t, Rating(Props{ReadOnly: true, Value: 7, Max: 10, Label: "Quality", ID: "q"}))
	mustContain(t, out, `aria-label="Quality: 7 of 10"`, `id="q"`)
	if got := strings.Count(out, "<svg"); got != 10 {
		t.Errorf("want 10 stars, got %d", got)
	}
}

func TestRating_RadiosCountAndChecked(t *testing.T) {
	out := render(t, Rating(Props{Name: "stars", Value: 3, Label: "Your rating"}))
	mustContain(t, out, `<fieldset`, `id="rating-stars"`, `<legend class="text-sm font-medium leading-none">Your rating</legend>`, `class="wf-rating`)
	// Five star radios plus the Clear radio of an optional rating.
	if got := strings.Count(out, `type="radio" data-star`); got != 5 {
		t.Errorf("want 5 star radios, got %d", got)
	}
	if got := strings.Count(out, `name="stars"`); got != 6 {
		t.Errorf("every radio carries the name, got %d", got)
	}
	if got := strings.Count(out, `class="sr-only"`); got != 12 {
		t.Errorf("radios, star texts and the Clear text are visually hidden: want 12 sr-only, got %d", got)
	}
	checked := regexp.MustCompile(`<input[^>]*value="3"[^>]*\schecked[\s/>]`)
	if !checked.MatchString(out) {
		t.Errorf("radio 3 should be checked:\n%s", out)
	}
	if got := len(regexp.MustCompile(`\schecked[\s/>]`).FindAllString(out, -1)); got != 1 {
		t.Errorf("exactly one checked radio, got %d", got)
	}
	for i := 1; i <= 5; i++ {
		id := "rating-stars-" + string(rune('0'+i))
		mustContain(t, out, `id="`+id+`"`, `for="`+id+`"`)
	}
	mustContain(t, out, "1 star</span>", "2 stars</span>", "5 stars</span>")
	mustContain(t, out, "[input:checked+&amp;]:text-warning", "[&amp;:has(~input:checked)]:text-warning", "[.wf-rating:hover_&amp;:has(~label:hover)]:text-warning", "[input:focus-visible+&amp;]:ring-2")
	mustNotContain(t, out, `role="img"`)
}

func TestRating_NoValueChecksNothing(t *testing.T) {
	for _, v := range []int{0, 6, -1} {
		out := render(t, Rating(Props{Name: "stars", Value: v}))
		if regexp.MustCompile(`\schecked[\s/>]`).MatchString(out) {
			t.Errorf("Value %d should check nothing:\n%s", v, out)
		}
	}
}

func TestRating_RequiredDisabledAndEmptyLabel(t *testing.T) {
	out := render(t, Rating(Props{Name: "stars", Required: true, Disabled: true}))
	mustContain(t, out, `<legend class="sr-only">Rating</legend>`)
	if got := len(regexp.MustCompile(`\srequired[\s/>]`).FindAllString(out, -1)); got != 1 {
		t.Errorf("required goes on one radio of the group, got %d", got)
	}
	if !regexp.MustCompile(`<fieldset[^>]*\sdisabled[\s/>]`).MatchString(out) {
		t.Errorf("fieldset should be disabled:\n%s", out)
	}
	plain := render(t, Rating(Props{Name: "stars"}))
	for _, attr := range []string{"required", "disabled"} {
		if regexp.MustCompile(`\s` + attr + `[\s/>]`).MatchString(plain) {
			t.Errorf("unexpected %s attribute:\n%s", attr, plain)
		}
	}
}

func TestRating_SizeClasses(t *testing.T) {
	for _, tc := range []struct {
		size   Size
		target string
		icon   string
	}{
		{SizeSm, "h-6 w-6", `width="16"`},
		{SizeMd, "h-8 w-8", `width="20"`},
		{SizeLg, "h-11 w-11", `width="28"`},
	} {
		out := render(t, Rating(Props{Name: "s", Size: tc.size}))
		mustContain(t, out, tc.target, tc.icon)
		ro := render(t, Rating(Props{ReadOnly: true, Size: tc.size}))
		mustContain(t, ro, tc.icon)
	}
}

func TestRating_ClassAndAttributes(t *testing.T) {
	out := render(t, Rating(Props{Name: "s", Class: "mt-2", Attributes: templ.Attributes{"aria-describedby": "s-help"}}))
	mustContain(t, out, "mt-2", `aria-describedby="s-help"`)
	ro := render(t, Rating(Props{ReadOnly: true, Class: "mt-2", Attributes: templ.Attributes{"data-x": "1"}}))
	mustContain(t, ro, "mt-2", `data-x="1"`)
}

func TestControlID(t *testing.T) {
	if got := (Props{ID: "x", Name: "n"}).ControlID(); got != "x" {
		t.Errorf("ID wins: got %q", got)
	}
	if got := (Props{Name: "n"}).ControlID(); got != "rating-n" {
		t.Errorf("derived from Name: got %q", got)
	}
	if got := (Props{}).ControlID(); !strings.HasPrefix(got, "id-") {
		t.Errorf("neither: got %q, want a utils.RandomID()", got)
	}
}

// TestRating_RandomIDPerRender: two ratings with neither ID nor Name get
// distinct ids, and within one input-mode render the fieldset and its
// radios share the one id (resolved once).
func TestRating_RandomIDPerRender(t *testing.T) {
	idAttr := regexp.MustCompile(`id="(id-[^"]+)"`)
	a := render(t, Rating(Props{ReadOnly: true, Value: 3}))
	b := render(t, Rating(Props{ReadOnly: true, Value: 3}))
	ma, mb := idAttr.FindStringSubmatch(a), idAttr.FindStringSubmatch(b)
	if ma == nil || mb == nil {
		t.Fatalf("no random id:\n%s\n%s", a, b)
	}
	if ma[1] == mb[1] {
		t.Errorf("two read-only ratings share id %q", ma[1])
	}
	in := render(t, Rating(Props{Max: 2}))
	ids := idAttr.FindAllStringSubmatch(in, -1)
	if len(ids) != 4 {
		t.Fatalf("want fieldset + Clear + 2 radio ids, got %d in:\n%s", len(ids), in)
	}
	root := ids[0][1]
	for _, m := range ids[1:] {
		if !strings.HasPrefix(m[1], root+"-") {
			t.Errorf("radio id %q does not derive from the root id %q", m[1], root)
		}
	}
}

func TestRating_ErrorVisual(t *testing.T) {
	out := render(t, Rating(Props{Name: "r", Max: 3, Attributes: templ.Attributes{"aria-invalid": "true"}}))
	for _, want := range []string{`aria-invalid="true"`, "[fieldset[aria-invalid=true]_&]:ring-2", "ring-destructive/40"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// TestRating_Clear: an optional rating has a Clear radio with an empty
// value, first in the DOM (so a checked Clear lights no star) and shown
// last; a required rating has none, since "" would satisfy required.
func TestRating_Clear(t *testing.T) {
	out := render(t, Rating(Props{Name: "stars", ID: "r", Value: 2}))
	clear := regexp.MustCompile(`<input type="radio" id="r-clear" name="stars" value="" class="sr-only"`)
	if !clear.MatchString(out) {
		t.Fatalf("no Clear radio:\n%s", out)
	}
	if strings.Index(out, `id="r-clear"`) > strings.Index(out, `id="r-1"`) {
		t.Errorf("Clear must come before the stars in the DOM:\n%s", out)
	}
	mustContain(t, out, `for="r-clear"`, "order-last", "[.wf-rating:has(input[data-star]:checked)_&amp;]:inline-flex", `<span class="sr-only">Clear rating</span>`)
	req := render(t, Rating(Props{Name: "stars", ID: "r", Required: true}))
	if strings.Contains(req, "r-clear") || strings.Contains(req, "Clear rating") {
		t.Errorf("a required rating has no Clear:\n%s", req)
	}
	ro := render(t, Rating(Props{ReadOnly: true, Value: 2}))
	if strings.Contains(ro, "Clear") {
		t.Errorf("read-only has no Clear:\n%s", ro)
	}
}
