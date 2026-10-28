package selectbox

import (
	"bytes"
	"context"
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
