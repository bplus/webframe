package swatch

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func render(t *testing.T, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Swatch(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestSwatch_SanitizesColor(t *testing.T) {
	if out := render(t, Props{Color: "red; x"}); strings.Contains(out, "; x") {
		t.Errorf("unsanitized color leaked: %s", out)
	}
	if out := render(t, Props{Color: "#ff0000"}); !strings.Contains(out, "#ff0000") {
		t.Errorf("expected #ff0000 in: %s", out)
	}
}

func TestSwatch_Label(t *testing.T) {
	if out := render(t, Props{Color: "#fff", Label: "primary"}); !strings.Contains(out, "primary") {
		t.Errorf("label missing: %s", out)
	}
	if out := render(t, Props{Color: "#fff"}); strings.Contains(out, "text-xs") {
		t.Errorf("label element should be omitted: %s", out)
	}
}
