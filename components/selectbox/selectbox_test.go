package selectbox

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

// tagContaining returns the first element tag (from "<"+name to ">") in out.
func tag(t *testing.T, out, name string) string {
	t.Helper()
	i := strings.Index(out, "<"+name)
	if i < 0 {
		t.Fatalf("no <%s in %s", name, out)
	}
	j := strings.Index(out[i:], ">")
	if j < 0 {
		t.Fatalf("unterminated <%s in %s", name, out)
	}
	return out[i : i+j+1]
}

func TestTrigger_ButtonAttributesOnButton(t *testing.T) {
	out := render(t, Trigger(TriggerProps{
		Name:             "type",
		Attributes:       templ.Attributes{"data-on-hidden": "1"},
		ButtonAttributes: templ.Attributes{"aria-describedby": "x", "aria-required": "true"},
	}))
	button := tag(t, out, "button")
	hidden := tag(t, out, "input")
	if !strings.Contains(button, `aria-describedby="x"`) || !strings.Contains(button, `aria-required="true"`) {
		t.Errorf("ButtonAttributes missing from button tag: %s", button)
	}
	if strings.Contains(hidden, "aria-describedby") || strings.Contains(hidden, "aria-required") {
		t.Errorf("ButtonAttributes leaked onto hidden input: %s", hidden)
	}
	if !strings.Contains(hidden, `data-on-hidden="1"`) || !strings.Contains(hidden, `name="type"`) || !strings.Contains(hidden, `type="hidden"`) {
		t.Errorf("Attributes/Name should stay on the hidden input: %s", hidden)
	}
	if strings.Contains(button, "data-on-hidden") {
		t.Errorf("Attributes leaked onto button: %s", button)
	}
}

func TestTrigger_ButtonAttributesOverrideAriaInvalid(t *testing.T) {
	out := render(t, Trigger(TriggerProps{ButtonAttributes: templ.Attributes{"aria-invalid": "true"}}))
	if !strings.Contains(tag(t, out, "button"), `aria-invalid="true"`) {
		t.Errorf("caller aria-invalid should win: %s", out)
	}
	if !strings.Contains(tag(t, out, "button"), "data-tui-selectbox-content-id") {
		t.Errorf("built-in data attributes must survive the merge: %s", out)
	}
}

func TestTrigger_AriaInvalidOnlyWhenHasError(t *testing.T) {
	// The class list carries aria-invalid:* Tailwind variants, so match the
	// attribute form only.
	plain := tag(t, render(t, Trigger(TriggerProps{Name: "type"})), "button")
	if strings.Contains(plain, `aria-invalid="`) {
		t.Errorf("no aria-invalid attribute expected without HasError: %s", plain)
	}
	errored := tag(t, render(t, Trigger(TriggerProps{Name: "type", HasError: true})), "button")
	if !strings.Contains(errored, `aria-invalid="true"`) {
		t.Errorf("HasError should emit aria-invalid=\"true\": %s", errored)
	}
	// A caller's ButtonAttributes value still wins over HasError.
	overridden := tag(t, render(t, Trigger(TriggerProps{HasError: true, ButtonAttributes: templ.Attributes{"aria-invalid": "grammar"}})), "button")
	if !strings.Contains(overridden, `aria-invalid="grammar"`) || strings.Contains(overridden, `aria-invalid="true"`) {
		t.Errorf("caller aria-invalid should win over HasError: %s", overridden)
	}
}

// TestOrphans_UniqueAndPaired: two Trigger+Content pairs rendered without a
// SelectBox on one page get distinct ids, each trigger targeting its own
// content; inside a SelectBox the wrapper-derived id still links them.
func TestOrphans_UniqueAndPaired(t *testing.T) {
	seq := func(cs ...templ.Component) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			ctx = templ.InitializeContext(ctx)
			for _, c := range cs {
				if err := c.Render(ctx, w); err != nil {
					return err
				}
			}
			return nil
		})
	}
	ids := func(out string) (triggers, contents []string) {
		for _, m := range regexp.MustCompile(`data-tui-popover-trigger="([^"]*)"`).FindAllStringSubmatch(out, -1) {
			triggers = append(triggers, m[1])
		}
		for _, m := range regexp.MustCompile(`data-tui-popover-id="([^"]*)"`).FindAllStringSubmatch(out, -1) {
			contents = append(contents, m[1])
		}
		return
	}

	triggers, contents := ids(render(t, seq(Trigger(), Content(), Trigger(), Content())))
	if len(triggers) != 2 || len(contents) != 2 {
		t.Fatalf("triggers %q, contents %q, want two of each", triggers, contents)
	}
	if contents[0] == contents[1] {
		t.Errorf("both orphaned contents have id %q", contents[0])
	}
	for i := range triggers {
		if triggers[i] == "" || triggers[i] != contents[i] {
			t.Errorf("trigger %d targets %q, content %d is %q", i, triggers[i], i, contents[i])
		}
	}

	inside := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return SelectBox(Props{ID: "pick"}).Render(templ.WithChildren(ctx, seq(Trigger(), Content())), w)
	})
	triggers, contents = ids(render(t, inside))
	if len(triggers) != 1 || len(contents) != 1 || triggers[0] != "pick-content" || contents[0] != "pick-content" {
		t.Errorf("inside SelectBox: triggers %q, contents %q, want [pick-content] both", triggers, contents)
	}
}
