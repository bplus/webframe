package form

import (
	"bytes"
	"context"
	"io"
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
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

// boolAttr matches a bare boolean attribute.
func boolAttr(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s` + name + `[\s/>]`)
}

func TestForm_DefaultsToPost(t *testing.T) {
	out := render(t, Form(Props{ID: "f", Action: "/items"}))
	mustContain(t, out, `<form`, `id="f"`, `action="/items"`, `method="post"`, `</form>`)
	for _, unwanted := range []string{"enctype", "hx-", "novalidate", `type="hidden"`} {
		if strings.Contains(out, unwanted) {
			t.Errorf("unexpected %q in %s", unwanted, out)
		}
	}
}

func TestForm_MethodGet(t *testing.T) {
	out := render(t, Form(Props{Action: "/items", Method: "get"}))
	mustContain(t, out, `method="get"`)
}

func TestForm_HiddenSorted(t *testing.T) {
	out := render(t, Form(Props{Hidden: map[string]string{"b": "2", "a": "1"}}))
	mustContain(t, out, `type="hidden"`, `name="a"`, `value="1"`, `name="b"`, `value="2"`)
	if a, b := strings.Index(out, `name="a"`), strings.Index(out, `name="b"`); a > b {
		t.Errorf("hidden inputs not in sorted key order: %s", out)
	}
	if n := strings.Count(out, `type="hidden"`); n != 2 {
		t.Errorf("want 2 hidden inputs, got %d in %s", n, out)
	}
}

func TestForm_HxAttrs(t *testing.T) {
	out := render(t, Form(Props{HxPost: "/items", HxTarget: "#body", HxSwap: "outerHTML"}))
	mustContain(t, out, `hx-post="/items"`, `hx-target="#body"`, `hx-swap="outerHTML"`)
	if strings.Contains(out, "hx-get") {
		t.Errorf("unexpected hx-get in %s", out)
	}
	out = render(t, Form(Props{HxGet: "/items"}))
	mustContain(t, out, `hx-get="/items"`)
}

func TestForm_Multipart(t *testing.T) {
	out := render(t, Form(Props{Multipart: true, NoValidate: true}))
	mustContain(t, out, `enctype="multipart/form-data"`)
	if !boolAttr("novalidate").MatchString(out) {
		t.Errorf("missing novalidate in %s", out)
	}
}

func TestForm_ClassAttributesAndChildren(t *testing.T) {
	inner := templ.Raw(`<button type="submit">Save</button>`)
	comp := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Form(Props{Class: "space-y-4", Attributes: templ.Attributes{"data-x": "y"}}).Render(templ.WithChildren(ctx, inner), w)
	})
	out := render(t, comp)
	mustContain(t, out, "space-y-4", `data-x="y"`, `<button type="submit">Save</button></form>`)
}

func TestMessage_VariantsUseThemeTokens(t *testing.T) {
	errOut := render(t, Message(MessageProps{Variant: MessageVariantError}))
	mustContain(t, errOut, "text-destructive")
	if strings.Contains(errOut, "text-red-500") {
		t.Errorf("error message must use the destructive token, not a palette color: %s", errOut)
	}
	infoOut := render(t, Message(MessageProps{Variant: MessageVariantInfo}))
	mustContain(t, infoOut, "text-info")
	if strings.Contains(infoOut, "text-blue-500") {
		t.Errorf("info message must use the info token, not a palette color: %s", infoOut)
	}
}
