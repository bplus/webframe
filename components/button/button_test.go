package button

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestButton_HrefSanitized(t *testing.T) {
	var buf bytes.Buffer
	if err := Button(Props{Href: "javascript:alert(1)"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if out := buf.String(); strings.Contains(out, "javascript:") {
		t.Errorf("javascript: URL not sanitized: %s", out)
	}
}

func TestButton_HrefRelativeKept(t *testing.T) {
	var buf bytes.Buffer
	if err := Button(Props{Href: "/movies?q=1"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if out := buf.String(); !strings.Contains(out, `href="/movies?q=1"`) {
		t.Errorf("relative href lost: %s", out)
	}
}
