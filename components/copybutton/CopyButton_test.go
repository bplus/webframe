package copybutton_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/copybutton"
	"github.com/bplus/webframe/utils"
)

func TestSizeClasses_Default(t *testing.T) {
	var buf bytes.Buffer
	comp := copybutton.CopyButton(copybutton.Props{
		Value: "test-value",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	// TwMerge may reorder classes, so check each individually
	if !strings.Contains(html, "h-8") {
		t.Errorf("expected h-8 class, got: %s", html)
	}
	if !strings.Contains(html, "w-8") {
		t.Errorf("expected w-8 class, got: %s", html)
	}
}

func TestSizeClasses_Sm(t *testing.T) {
	var buf bytes.Buffer
	comp := copybutton.CopyButton(copybutton.Props{
		Value: "test-value",
		Size:  copybutton.SizeSm,
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	// TwMerge may reorder classes, so check each individually
	if !strings.Contains(html, "h-6") {
		t.Errorf("expected h-6 class, got: %s", html)
	}
	if !strings.Contains(html, "w-6") {
		t.Errorf("expected w-6 class, got: %s", html)
	}
}

func TestCopyButton_WithLabel(t *testing.T) {
	var buf bytes.Buffer
	comp := copybutton.CopyButton(copybutton.Props{
		Value: "some-uuid",
		Label: "Copy ID",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "Copy ID") {
		t.Errorf("expected label text, got: %s", html)
	}
	if !strings.Contains(html, "gap-1.5") {
		t.Errorf("expected gap class when label present, got: %s", html)
	}
}

func TestCopyButton_WithID(t *testing.T) {
	var buf bytes.Buffer
	comp := copybutton.CopyButton(copybutton.Props{
		ID:    "my-btn",
		Value: "v",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `id="my-btn"`) {
		t.Errorf("expected id attribute, got: %s", html)
	}
}

func TestCopyButton_ValueInOnClick(t *testing.T) {
	var buf bytes.Buffer
	comp := copybutton.CopyButton(copybutton.Props{
		Value: "abc-123",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "abc-123") {
		t.Errorf("expected value in click handler, got: %s", html)
	}
	if !strings.Contains(html, "navigator.clipboard.writeText") {
		t.Errorf("expected clipboard API call, got: %s", html)
	}
}

func TestEscapeJS_SingleQuotes(t *testing.T) {
	got := utils.EscapeJS("it's a test")
	want := `it\'s a test`
	if got != want {
		t.Errorf("EscapeJS single quote: got %q, want %q", got, want)
	}
}

func TestEscapeJS_DoubleQuotes(t *testing.T) {
	got := utils.EscapeJS(`say "hello"`)
	want := `say \"hello\"`
	if got != want {
		t.Errorf("EscapeJS double quote: got %q, want %q", got, want)
	}
}

func TestEscapeJS_Backslash(t *testing.T) {
	got := utils.EscapeJS(`path\to\file`)
	want := `path\\to\\file`
	if got != want {
		t.Errorf("EscapeJS backslash: got %q, want %q", got, want)
	}
}

func TestEscapeJS_Newlines(t *testing.T) {
	got := utils.EscapeJS("line1\nline2\rline3")
	want := `line1\nline2\rline3`
	if got != want {
		t.Errorf("EscapeJS newlines: got %q, want %q", got, want)
	}
}

func TestEscapeJS_ScriptTag(t *testing.T) {
	got := utils.EscapeJS("</script>")
	want := `<\/script>`
	if got != want {
		t.Errorf("EscapeJS script tag: got %q, want %q", got, want)
	}
}

func TestEscapeJS_Combined(t *testing.T) {
	got := utils.EscapeJS("it's a \"test\"\nwith </script>")
	want := `it\'s a \"test\"\nwith <\/script>`
	if got != want {
		t.Errorf("EscapeJS combined: got %q, want %q", got, want)
	}
}
