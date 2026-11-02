package descriptionlist

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/badge"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := DescriptionList(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestDL_LabelsAndValues(t *testing.T) {
	out := render(t, Props{ID: "dl", Items: []Item{
		{Label: "Name", Value: "Widget"},
		{Label: "Owner", Value: "Ada"},
	}})
	for _, want := range []string{"<dl", `id="dl"`, "<dt", "<dd", "Name", "Widget", "Owner", "Ada"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	if got := strings.Count(out, "<dt"); got != 2 {
		t.Errorf("want 2 <dt>, got %d in %s", got, out)
	}
	if got := strings.Count(out, "<dd"); got != 2 {
		t.Errorf("want 2 <dd>, got %d in %s", got, out)
	}
	// Label precedes its value inside the same row.
	if strings.Index(out, "Name") > strings.Index(out, "Widget") {
		t.Errorf("label after value: %s", out)
	}
}

func TestDL_HrefMonoCopy(t *testing.T) {
	out := render(t, Props{Items: []Item{
		{Label: "ID", Value: "abc123", Href: "/x", Mono: true, Copyable: true},
	}})
	if !strings.Contains(out, `<a href="/x"`) {
		t.Errorf("missing link: %s", out)
	}
	if !strings.Contains(out, "font-mono") {
		t.Errorf("missing font-mono: %s", out)
	}
	if !strings.Contains(out, "navigator.clipboard.writeText(&#39;abc123&#39;)") {
		t.Errorf("missing copy button for value: %s", out)
	}
	if strings.Index(out, ">abc123</a>") > strings.Index(out, "navigator.clipboard") {
		t.Errorf("copy button not after the value: %s", out)
	}
}

func TestDL_PlainValueHasNoLinkMonoOrCopy(t *testing.T) {
	out := render(t, Props{Items: []Item{{Label: "K", Value: "v"}}})
	for _, bad := range []string{"<a ", "font-mono", "navigator.clipboard", "<button"} {
		if strings.Contains(out, bad) {
			t.Errorf("unexpected %q in %s", bad, out)
		}
	}
}

func TestDL_UnsafeHrefSanitized(t *testing.T) {
	out := render(t, Props{Items: []Item{{Label: "K", Value: "v", Href: "javascript:alert(1)"}}})
	if strings.Contains(out, "javascript:alert(1)") {
		t.Errorf("unsanitized href: %s", out)
	}
}

func TestDL_Badge(t *testing.T) {
	out := render(t, Props{Items: []Item{
		{Label: "Status", Value: "Active", Badge: &badge.Props{Variant: badge.VariantSecondary}},
	}})
	if !strings.Contains(out, "bg-secondary") {
		t.Errorf("missing badge variant classes: %s", out)
	}
	i := strings.Index(out, "bg-secondary")
	if i < 0 || !strings.Contains(out[i:], "Active") {
		t.Errorf("value not inside the badge: %s", out)
	}
}

func TestDL_Component(t *testing.T) {
	out := render(t, Props{Items: []Item{
		{Label: "Custom", Value: "ignored", Component: templ.Raw(`<em data-custom>hi</em>`)},
	}})
	if !strings.Contains(out, "<em data-custom>hi</em>") {
		t.Errorf("missing component: %s", out)
	}
	if strings.Contains(out, "ignored") {
		t.Errorf("Value rendered alongside Component: %s", out)
	}
}

func TestDL_TwoColumns(t *testing.T) {
	one := render(t, Props{Items: []Item{{Label: "K", Value: "v"}}})
	if strings.Contains(one, "md:grid-cols-2") {
		t.Errorf("single column has md:grid-cols-2: %s", one)
	}
	if !strings.Contains(one, "grid-cols-1") {
		t.Errorf("missing grid-cols-1: %s", one)
	}
	two := render(t, Props{Columns: 2, Items: []Item{{Label: "K", Value: "v"}}})
	if !strings.Contains(two, "md:grid-cols-2") {
		t.Errorf("missing md:grid-cols-2: %s", two)
	}
	if !strings.Contains(two, "grid-cols-1") {
		t.Errorf("two columns must still collapse to one below md: %s", two)
	}
}

func TestDL_Dense(t *testing.T) {
	out := render(t, Props{Dense: true, Items: []Item{{Label: "K", Value: "v"}}})
	if !strings.Contains(out, "py-1") {
		t.Errorf("missing py-1: %s", out)
	}
	if strings.Contains(out, "py-2") {
		t.Errorf("dense row still has py-2: %s", out)
	}
}

func TestDL_EmptyValueDash(t *testing.T) {
	out := render(t, Props{Items: []Item{{Label: "Empty"}}})
	if !strings.Contains(out, "—") {
		t.Errorf("missing em dash: %s", out)
	}
	i := strings.Index(out, "<dd")
	if i < 0 || !strings.Contains(out[i:], "text-muted-foreground") {
		t.Errorf("dash not muted: %s", out)
	}
	if strings.Contains(out, "navigator.clipboard") {
		t.Errorf("empty value must not get a copy button: %s", out)
	}
}

func TestDL_MutedValue(t *testing.T) {
	out := render(t, Props{Items: []Item{{Label: "K", Value: "n/a", Muted: true}}})
	i := strings.Index(out, "<dd")
	if i < 0 || !strings.Contains(out[i:], "text-muted-foreground") {
		t.Errorf("muted value lacks muted class: %s", out)
	}
}

func TestDL_ClassAndAttributes(t *testing.T) {
	out := render(t, Props{Class: "mt-4", Attributes: templ.Attributes{"data-x": "1"}, Items: []Item{{Label: "K", Value: "v"}}})
	if !strings.Contains(out, "mt-4") {
		t.Errorf("missing merged class: %s", out)
	}
	if !strings.Contains(out, `data-x="1"`) {
		t.Errorf("missing attribute: %s", out)
	}
}

func TestDL_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := DescriptionList().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<dl") || strings.Contains(out, "<dt") {
		t.Errorf("zero items should render an empty <dl>: %s", out)
	}
}

// BadgeItem builds the badge row callers used to assemble by hand: kind
// picks the variant through badge.StatusPtr, text is the badge's text.
func TestDL_BadgeItem(t *testing.T) {
	it := BadgeItem("Status", "Failed", "failed")
	if it.Label != "Status" || it.Value != "Failed" || it.Badge == nil || it.Badge.Variant != badge.VariantDestructive {
		t.Fatalf("BadgeItem = %+v", it)
	}
	if BadgeItem("S", "x", "whatever").Badge.Variant != badge.VariantOutline {
		t.Errorf("unknown kind should fall back to outline")
	}
	out := render(t, Props{Items: []Item{it}})
	for _, want := range []string{"Status", "bg-destructive"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
	i := strings.Index(out, "bg-destructive")
	if i < 0 || !strings.Contains(out[i:], "Failed") {
		t.Errorf("value not inside the badge: %s", out)
	}
}
