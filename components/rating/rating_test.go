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
	mustContain(t, out, `role="img"`, `aria-label="3 of 5"`, `id="rating"`)
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
	if got := strings.Count(out, `type="radio"`); got != 5 {
		t.Errorf("want 5 radios, got %d", got)
	}
	if got := strings.Count(out, `name="stars"`); got != 5 {
		t.Errorf("every radio carries the name, got %d", got)
	}
	if got := strings.Count(out, `class="sr-only"`); got != 10 {
		t.Errorf("radios and star texts are visually hidden: want 10 sr-only, got %d", got)
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
	if got := (Props{}).ControlID(); got != "rating" {
		t.Errorf("neither: got %q", got)
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
