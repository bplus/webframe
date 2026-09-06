package tabs_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/tabs"
)

func TestContentEmptyValueDoesNotEmitNopComponentText(t *testing.T) {
	var buf bytes.Buffer
	comp := tabs.Content(tabs.ContentProps{})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, "return templ.NopComponent") {
		t.Errorf("expected no literal NopComponent text, got: %s", html)
	}
	if !strings.Contains(html, "missing required 'Value' attribute") {
		t.Errorf("expected error message, got: %s", html)
	}
}
