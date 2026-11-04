package css_test

import (
	"os"
	"strings"
	"testing"
)

// TestTokens_Present checks that each status token is declared in both
// places a semantic color must live: the @theme block (so Tailwind compiles
// bg-success, text-warning, border-info/30 utilities and supplies the light
// values) and the dark palette (every token has a dark value).
func TestTokens_Present(t *testing.T) {
	b, err := os.ReadFile("input.css")
	if err != nil {
		t.Fatalf("read input.css: %v", err)
	}
	css := string(b)

	theme := section(t, css, "@theme {")
	dark := section(t, css, `:root[data-theme="dark"] {`)

	tokens := []string{
		"--color-success:", "--color-success-foreground:",
		"--color-warning:", "--color-warning-foreground:",
		"--color-info:", "--color-info-foreground:",
	}
	for _, tok := range tokens {
		for name, block := range map[string]string{"@theme": theme, "dark": dark} {
			if !strings.Contains(block, tok) {
				t.Errorf("input.css %s block missing %s", name, tok)
			}
		}
		if n := strings.Count(css, tok); n != 2 {
			t.Errorf("input.css declares %s %d times, want exactly 2 (@theme, dark)", tok, n)
		}
	}
}

// section returns the text from the first occurrence of open up to the next
// line that is exactly "}". Blocks in input.css are flat (no nesting).
func section(t *testing.T, css, open string) string {
	t.Helper()
	i := strings.Index(css, open)
	if i < 0 {
		t.Fatalf("input.css has no %q block", open)
	}
	rest := css[i+len(open):]
	j := strings.Index(rest, "\n}")
	if j < 0 {
		t.Fatalf("input.css %q block is unterminated", open)
	}
	return rest[:j]
}
