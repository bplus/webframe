package dropdown

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
