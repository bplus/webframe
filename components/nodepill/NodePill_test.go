package nodepill

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

// --- dotColor() tests ---

func TestDotColor_AllTypes(t *testing.T) {
	tests := []struct {
		nodeType string
		want     string
	}{
		{"concept", "bg-info"},
		{"tool", "bg-success"},
		{"decision", "bg-warning"},
		{"error", "bg-destructive"},
		{"service", "bg-primary"},
		{"person", "bg-info/60"},
		{"incident", "bg-destructive/60"},
		{"code_path", "bg-success/60"},
		{"reference", "bg-muted-foreground"},
	}
	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			p := Props{NodeType: tt.nodeType}
			got := p.dotColor()
			if got != tt.want {
				t.Errorf("dotColor() for %q = %q, want %q", tt.nodeType, got, tt.want)
			}
		})
	}
}

func TestDotColor_UnknownType(t *testing.T) {
	cases := []string{"", "unknown", "CONCEPT", "widget", "foo_bar"}
	for _, nodeType := range cases {
		t.Run(nodeType, func(t *testing.T) {
			p := Props{NodeType: nodeType}
			got := p.dotColor()
			if got != "bg-muted-foreground" {
				t.Errorf("dotColor() for unknown type %q = %q, want %q", nodeType, got, "bg-muted-foreground")
			}
		})
	}
}

// --- displayTitle() tests ---

func TestDisplayTitle_ShortTitle(t *testing.T) {
	p := Props{Title: "Hello"}
	got := p.displayTitle()
	if got != "Hello" {
		t.Errorf("displayTitle() = %q, want %q", got, "Hello")
	}
}

func TestDisplayTitle_ExactLimit(t *testing.T) {
	// Exactly 30 chars — should NOT truncate
	title := "123456789012345678901234567890" // 30 chars
	p := Props{Title: title}
	got := p.displayTitle()
	if got != title {
		t.Errorf("displayTitle() = %q, want %q (no truncation at exact limit)", got, title)
	}
}

func TestDisplayTitle_OverLimit(t *testing.T) {
	// 31 chars — should truncate to 29 chars + ellipsis
	title := "1234567890123456789012345678901" // 31 chars
	p := Props{Title: title}
	got := p.displayTitle()
	want := "12345678901234567890123456789\u2026" // 29 chars + ellipsis
	if got != want {
		t.Errorf("displayTitle() = %q, want %q", got, want)
	}
}

func TestDisplayTitle_EmptyTitle(t *testing.T) {
	p := Props{Title: ""}
	got := p.displayTitle()
	if got != "" {
		t.Errorf("displayTitle() = %q, want empty string", got)
	}
}

func TestDisplayTitle_CustomTruncate(t *testing.T) {
	p := Props{Title: "Hello World", Truncate: 5}
	got := p.displayTitle()
	want := "Hell\u2026" // 4 chars + ellipsis
	if got != want {
		t.Errorf("displayTitle() with Truncate=5 = %q, want %q", got, want)
	}
}

func TestDisplayTitle_CustomTruncate_NoTruncationNeeded(t *testing.T) {
	p := Props{Title: "Hi", Truncate: 10}
	got := p.displayTitle()
	if got != "Hi" {
		t.Errorf("displayTitle() = %q, want %q", got, "Hi")
	}
}

func TestDisplayTitle_CustomTruncate_ExactCustomLimit(t *testing.T) {
	p := Props{Title: "Hello", Truncate: 5}
	got := p.displayTitle()
	if got != "Hello" {
		t.Errorf("displayTitle() at exact custom limit = %q, want %q", got, "Hello")
	}
}

// TestDisplayTitle_MultiByteUTF8_RuneBoundary guards against truncating by
// byte index: a multi-byte title cut mid-rune produces invalid UTF-8 that
// renders as U+FFFD in the browser.
func TestDisplayTitle_MultiByteUTF8_RuneBoundary(t *testing.T) {
	p := Props{Title: "héllo world", Truncate: 3}
	got := p.displayTitle()

	if !utf8.ValidString(got) {
		t.Fatalf("displayTitle() = %q is not valid UTF-8", got)
	}
	want := "hé…" // "hé…" — 2 runes + ellipsis
	if got != want {
		t.Errorf("displayTitle() = %q, want %q", got, want)
	}
}

func TestDisplayTitle_MultiByteUTF8_WideRunes(t *testing.T) {
	// Every rune here is multi-byte; a byte-index cut would slice a rune
	// in half no matter where the cutoff falls.
	p := Props{Title: "こんにちは世界", Truncate: 5}
	got := p.displayTitle()

	if !utf8.ValidString(got) {
		t.Fatalf("displayTitle() = %q is not valid UTF-8", got)
	}
	want := "こんにち…" // 4 runes + ellipsis
	if got != want {
		t.Errorf("displayTitle() = %q, want %q", got, want)
	}
}

// TestDisplayTitle_NegativeTruncate guards against a negative Truncate
// causing a slice-bounds panic; it should be treated as the default (30).
func TestDisplayTitle_NegativeTruncate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("displayTitle() panicked with negative Truncate: %v", r)
		}
	}()

	p := Props{Title: "Hello", Truncate: -1}
	got := p.displayTitle()
	if got != "Hello" {
		t.Errorf("displayTitle() with negative Truncate = %q, want %q", got, "Hello")
	}
}

func TestDisplayTitle_NegativeTruncate_LongTitle(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("displayTitle() panicked with negative Truncate: %v", r)
		}
	}()

	title := "1234567890123456789012345678901" // 31 chars, over default 30
	p := Props{Title: title, Truncate: -5}
	got := p.displayTitle()
	want := "12345678901234567890123456789…" // default-limit behavior
	if got != want {
		t.Errorf("displayTitle() with negative Truncate = %q, want %q", got, want)
	}
}

func TestNodePillHrefIsSanitized(t *testing.T) {
	r := func(href string) string {
		var buf bytes.Buffer
		if err := NodePill(Props{Title: "n", Href: href}).Render(context.Background(), &buf); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	if out := r("javascript:alert(1)"); !strings.Contains(out, "about:invalid") || strings.Contains(out, "javascript:") {
		t.Errorf("unsafe href not sanitized: %s", out)
	}
	if out := r("https://example.com/a"); !strings.Contains(out, `href="https://example.com/a"`) {
		t.Errorf("safe href altered: %s", out)
	}
}
