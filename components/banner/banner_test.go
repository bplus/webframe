package banner

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
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

func TestBanner_EmptyRendersNothing(t *testing.T) {
	for name, props := range map[string][]Props{
		"no props":      nil,
		"zero props":    {{}},
		"only variant":  {{Variant: VariantError}},
		"only dismiss":  {{Dismissible: true}},
		"only actions":  {{Actions: []action.Action{{Label: "Retry", Href: "/retry"}}}},
		"only id/class": {{ID: "b", Class: "mt-4"}},
	} {
		t.Run(name, func(t *testing.T) {
			out := render(t, Banner(props...))
			if strings.TrimSpace(out) != "" {
				t.Errorf("expected empty output, got:\n%s", out)
			}
		})
	}
}

func TestBanner_ErrorIsAlert(t *testing.T) {
	out := render(t, Banner(Props{Variant: VariantError, Title: "3 fields need attention", Body: "Fix them below."}))
	mustContain(t, out,
		`role="alert"`,
		"text-destructive",
		"border-destructive",
		"bg-destructive/10",
		"border-l-4",
		"3 fields need attention",
		"Fix them below.",
		"font-semibold",
		"text-sm",
	)
	mustNotContain(t, out, `role="status"`)
}

func TestBanner_InfoIsStatus(t *testing.T) {
	out := render(t, Banner(Props{Title: "Heads up"}))
	mustContain(t, out, `role="status"`, "text-info", "border-info", "bg-info/10", "Heads up")
	mustNotContain(t, out, `role="alert"`, "x-show", `aria-label="Dismiss"`)

	for _, tc := range []struct {
		v    Variant
		want string
	}{
		{VariantSuccess, "success"},
		{VariantWarning, "warning"},
	} {
		out := render(t, Banner(Props{Variant: tc.v, Body: "b"}))
		mustContain(t, out, `role="status"`, "text-"+tc.want, "border-"+tc.want, "bg-"+tc.want+"/10")
	}
}

func TestBanner_DetailsAndActions(t *testing.T) {
	details := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<ul><li>Name is required</li></ul>`)
		return err
	})
	out := render(t, Banner(Props{
		Variant: VariantError,
		Title:   "1 field needs attention",
		Details: details,
		Actions: []action.Action{
			{Label: "Retry", HxPost: "/items", HxTarget: "#form"},
			{Label: "Help", Href: "/help"},
		},
	}))
	mustContain(t, out,
		`<ul><li>Name is required</li></ul>`,
		`hx-post="/items"`,
		`href="/help"`,
		">Retry<", ">Help<",
	)

	// Details alone is enough to render.
	out = render(t, Banner(Props{Details: details}))
	mustContain(t, out, `role="status"`, "Name is required")
}

func TestBanner_Dismissible(t *testing.T) {
	out := render(t, Banner(Props{Title: "Saved draft", Dismissible: true}))
	mustContain(t, out,
		`x-data="{ open: true }"`,
		`x-show="open"`,
		`aria-label="Dismiss"`,
		`type="button"`,
		`x-on:click="open = false"`,
		`<svg`,
	)
}

func TestBanner_IDClassAttributes(t *testing.T) {
	out := render(t, Banner(Props{
		ID:         "form-errors",
		Class:      "mt-6",
		Attributes: templ.Attributes{"data-test": "banner"},
		Title:      "T",
	}))
	mustContain(t, out, `id="form-errors"`, "mt-6", `data-test="banner"`)
}
