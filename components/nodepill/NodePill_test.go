package nodepill

import "testing"

// --- dotColor() tests ---

func TestDotColor_AllTypes(t *testing.T) {
	tests := []struct {
		nodeType string
		want     string
	}{
		{"concept", "bg-blue-500"},
		{"tool", "bg-green-500"},
		{"decision", "bg-amber-500"},
		{"error", "bg-red-500"},
		{"service", "bg-purple-500"},
		{"person", "bg-cyan-500"},
		{"incident", "bg-orange-500"},
		{"code_path", "bg-emerald-500"},
		{"reference", "bg-slate-500"},
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
