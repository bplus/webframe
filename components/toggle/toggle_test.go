package toggle

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

// boolAttr matches a bare boolean attribute (not a Tailwind "disabled:" variant).
func boolAttr(name string) *regexp.Regexp {
	return regexp.MustCompile(`\s` + name + `[\s/>]`)
}

func TestToggle_RoleSwitchAndName(t *testing.T) {
	out := render(t, Toggle(Props{ID: "dark", Name: "dark"}))
	mustContain(t, out, `role="switch"`, `name="dark"`, `value="on"`, `type="checkbox"`, `id="dark"`, `<label`, "sr-only", "peer")
	for _, name := range []string{"checked", "disabled"} {
		if boolAttr(name).MatchString(out) {
			t.Errorf("unexpected %s attribute in %s", name, out)
		}
	}
}

func TestToggle_Checked(t *testing.T) {
	out := render(t, Toggle(Props{Name: "dark", Checked: true, Value: "yes"}))
	if !boolAttr("checked").MatchString(out) {
		t.Errorf("missing checked attribute in %s", out)
	}
	mustContain(t, out, `value="yes"`)
}

func TestToggle_Disabled(t *testing.T) {
	out := render(t, Toggle(Props{Name: "dark", Disabled: true}))
	if !boolAttr("disabled").MatchString(out) {
		t.Errorf("missing disabled attribute in %s", out)
	}
}

func TestToggle_FormSizeAttributesAndChildren(t *testing.T) {
	c := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "Dark mode")
		return err
	})
	out := render(t, templ.Join(templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Toggle(Props{Name: "dark", Form: "prefs", Size: SizeS, Class: "mt-2", Attributes: templ.Attributes{"aria-describedby": "dark-help"}}).Render(templ.WithChildren(ctx, c), w)
	})))
	mustContain(t, out, `form="prefs"`, `aria-describedby="dark-help"`, "Dark mode", "mt-2")
}

func TestToggle_ZeroValue(t *testing.T) {
	out := render(t, Toggle())
	mustContain(t, out, `role="switch"`, `value="on"`)
	if strings.Contains(out, "name=") {
		t.Errorf("unexpected name attribute in %s", out)
	}
}
