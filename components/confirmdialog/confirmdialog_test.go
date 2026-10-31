package confirmdialog

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

// openTag returns the <button ...> opening tag that carries the given
// data-wf-confirm-part value.
func openTag(t *testing.T, out, part string) string {
	t.Helper()
	re := regexp.MustCompile(`<button[^>]*data-wf-confirm-part="` + regexp.QuoteMeta(part) + `"[^>]*>`)
	m := re.FindString(out)
	if m == "" {
		t.Fatalf("no <button> with data-wf-confirm-part=%q in:\n%s", part, out)
	}
	return m
}

func TestHost_RendersOneWfConfirmInstanceClosed(t *testing.T) {
	out := render(t, Host())
	if n := strings.Count(out, `id="wf-confirm"`); n != 1 {
		t.Fatalf("want exactly one id=\"wf-confirm\", got %d:\n%s", n, out)
	}
	for _, want := range []string{
		`data-tui-dialog`,
		`data-dialog-instance="wf-confirm"`,
		`data-tui-dialog-hidden="true"`,
		`data-tui-dialog-open="false"`,
		`data-tui-dialog-disable-autofocus="true"`,
		`role="alertdialog"`,
		`aria-labelledby="wf-confirm-title"`,
		`aria-describedby="wf-confirm-message"`,
		`id="wf-confirm-title"`,
		`id="wf-confirm-message"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, `data-tui-dialog-open="true"`) {
		t.Errorf("host must render closed:\n%s", out)
	}
	if strings.Contains(out, `data-tui-dialog-trigger`) {
		t.Errorf("host must have no Trigger (confirm.js opens it):\n%s", out)
	}
	if strings.Contains(out, `aria-label="Close"`) {
		t.Errorf("templUI X close button should be hidden:\n%s", out)
	}
}

func TestHost_DefaultsOnRootAndParts(t *testing.T) {
	out := render(t, Host())
	for _, want := range []string{
		`data-wf-confirm-title="Are you sure?"`,
		`data-wf-confirm-label="Confirm"`,
		`data-wf-confirm-cancel="Cancel"`,
		`data-wf-confirm-part="title"`,
		`data-wf-confirm-part="message"`,
		`data-wf-confirm-part="cancel"`,
		`data-wf-confirm-part="confirm"`,
		`data-wf-confirm-part="confirm-destructive"`,
		`>Are you sure?<`,
		`>Cancel<`,
		`>Confirm<`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, `data-wf-confirm-destructive`) {
		t.Errorf("non-destructive host must not carry the destructive flag:\n%s", out)
	}
	if strings.Contains(openTag(t, out, "confirm"), " hidden") {
		t.Errorf("primary confirm button should be visible by default")
	}
	if !strings.Contains(openTag(t, out, "confirm-destructive"), " hidden") {
		t.Errorf("destructive confirm button should be hidden by default")
	}
}

func TestConfirmDialog_TextsAndEscaping(t *testing.T) {
	out := render(t, ConfirmDialog(Props{
		ID: "d", Title: "Delete <item>", Message: "Really & truly?", ConfirmLabel: "Delete", CancelLabel: "Keep",
	}))
	for _, want := range []string{
		`>Delete &lt;item&gt;<`, `>Really &amp; truly?<`, `>Delete<`, `>Keep<`,
		`data-wf-confirm-title="Delete &lt;item&gt;"`, `data-wf-confirm-label="Delete"`, `data-wf-confirm-cancel="Keep"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestConfirmDialog_DestructivePicksDestructiveButton(t *testing.T) {
	out := render(t, ConfirmDialog(Props{ID: "d", Destructive: true}))
	if !strings.Contains(out, `data-wf-confirm-destructive="true"`) {
		t.Errorf("missing destructive flag on root:\n%s", out)
	}
	primary := openTag(t, out, "confirm")
	destructive := openTag(t, out, "confirm-destructive")
	if !strings.Contains(primary, " hidden") {
		t.Errorf("primary confirm button should be hidden when Destructive: %s", primary)
	}
	if strings.Contains(destructive, " hidden") {
		t.Errorf("destructive confirm button should be visible when Destructive: %s", destructive)
	}
	if !strings.Contains(destructive, "bg-destructive") {
		t.Errorf("destructive confirm button should use the destructive variant: %s", destructive)
	}
	if strings.Contains(primary, "bg-destructive") {
		t.Errorf("primary confirm button should not use the destructive variant: %s", primary)
	}
}

func TestConfirmDialog_CancelIsDialogClose(t *testing.T) {
	out := render(t, Host())
	cancel := strings.Index(out, `data-wf-confirm-part="cancel"`)
	closeIdx := strings.Index(out, `data-tui-dialog-close`)
	if cancel < 0 || closeIdx < 0 || closeIdx > cancel {
		t.Errorf("cancel button must sit inside a dialog.Close wrapper (close at %d, cancel at %d):\n%s", closeIdx, cancel, out)
	}
	if strings.Contains(openTag(t, out, "cancel"), "bg-destructive") || strings.Contains(openTag(t, out, "cancel"), "bg-primary") {
		t.Errorf("cancel should be the outline variant")
	}
}

func TestConfirmDialog_PhoneFooterWrapsOnePerRow(t *testing.T) {
	out := render(t, Host())
	footer := regexp.MustCompile(`<div[^>]*class="[^"]*flex-wrap[^"]*"[^>]*>`).FindString(out)
	if footer == "" {
		t.Fatalf("footer with flex-wrap not found:\n%s", out)
	}
	if !strings.Contains(footer, "flex-row") || strings.Contains(footer, "flex-col-reverse") {
		t.Errorf("footer should be a wrapping row at every width: %s", footer)
	}
	for _, part := range []string{"cancel", "confirm", "confirm-destructive"} {
		tag := openTag(t, out, part)
		if !strings.Contains(tag, "w-full") || !strings.Contains(tag, "min-[400px]:w-auto") {
			t.Errorf("%s button should be w-full below 400px and auto above: %s", part, tag)
		}
	}
}

func TestConfirmDialog_GeneratesIDWhenEmpty(t *testing.T) {
	out := render(t, ConfirmDialog(Props{}))
	if !strings.Contains(out, `aria-labelledby="id-`) {
		t.Errorf("empty ID should still wire aria-labelledby to a generated id:\n%s", out)
	}
}

func TestConfirmDialog_CallerAttributesAndClassOnRoot(t *testing.T) {
	out := render(t, ConfirmDialog(Props{ID: "d", Class: "mt-2", Attributes: templ.Attributes{"data-x": "1"}}))
	root := regexp.MustCompile(`<div[^>]*id="d"[^>]*>`).FindString(out)
	if root == "" {
		t.Fatalf("root not found:\n%s", out)
	}
	for _, want := range []string{`data-x="1"`, "mt-2", `data-tui-dialog`} {
		if !strings.Contains(root, want) {
			t.Errorf("root missing %q: %s", want, root)
		}
	}
}
