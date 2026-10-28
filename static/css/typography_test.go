package css_test

import (
	"os"
	"strings"
	"testing"
)

func TestOutputCSS_HasContentTypography(t *testing.T) {
	b, err := os.ReadFile("output.css")
	if err != nil {
		t.Fatalf("read output.css: %v", err)
	}
	css := string(b)
	for _, want := range []string{".wf-content) p:not([class])", "a:not([class])"} {
		if !strings.Contains(css, want) {
			t.Errorf("output.css missing %q", want)
		}
	}
}
