package dropdown

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestItem_HrefSanitized(t *testing.T) {
	var buf bytes.Buffer
	if err := Item(ItemProps{Href: "javascript:alert(1)"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if out := buf.String(); strings.Contains(out, "javascript:") {
		t.Errorf("javascript: URL not sanitized: %s", out)
	}
}

// orphanIDs renders c once and returns the trigger targets
// (data-tui-popover-trigger) and content ids (data-tui-popover-id) in order.
func orphanIDs(t *testing.T, c templ.Component) (triggers, contents []string) {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, m := range regexp.MustCompile(`data-tui-popover-trigger="([^"]*)"`).FindAllStringSubmatch(buf.String(), -1) {
		triggers = append(triggers, m[1])
	}
	for _, m := range regexp.MustCompile(`data-tui-popover-id="([^"]*)"`).FindAllStringSubmatch(buf.String(), -1) {
		contents = append(contents, m[1])
	}
	return triggers, contents
}

// seq renders cs one after another in the same render pass (one page).
func seq(cs ...templ.Component) templ.Component {
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

// TestOrphans_UniqueAndPaired: two Trigger+Content pairs (and two
// SubTrigger+SubContent pairs) rendered without their parent on one page get
// distinct ids, and each trigger targets its own content.
func TestOrphans_UniqueAndPaired(t *testing.T) {
	for name, page := range map[string]templ.Component{
		"top": seq(Trigger(), Content(), Trigger(), Content()),
		"sub": seq(SubTrigger(), SubContent(), SubTrigger(), SubContent()),
	} {
		triggers, contents := orphanIDs(t, page)
		if len(triggers) != 2 || len(contents) != 2 {
			t.Fatalf("%s: triggers %q, contents %q, want two of each", name, triggers, contents)
		}
		if contents[0] == contents[1] {
			t.Errorf("%s: both orphaned contents have id %q", name, contents[0])
		}
		for i := range triggers {
			if triggers[i] == "" || triggers[i] != contents[i] {
				t.Errorf("%s: trigger %d targets %q, content %d is %q", name, i, triggers[i], i, contents[i])
			}
		}
	}
}

// TestDropdown_PairingUnchanged: inside a Dropdown the explicit ID still
// links Trigger and Content.
func TestDropdown_PairingUnchanged(t *testing.T) {
	page := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return Dropdown(Props{ID: "menu"}).Render(templ.WithChildren(ctx, seq(Trigger(), Content())), w)
	})
	triggers, contents := orphanIDs(t, page)
	if len(triggers) != 1 || len(contents) != 1 || triggers[0] != "menu" || contents[0] != "menu" {
		t.Errorf("triggers %q, contents %q, want [menu] both", triggers, contents)
	}
}
