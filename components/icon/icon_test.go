package icon

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

// resetIconCache clears package-level cache state so tests don't interfere
// with each other or with whatever ran before them in the same process.
func resetIconCache() {
	iconMutex.Lock()
	iconContents = make(map[string]string)
	iconMutex.Unlock()
	iconGenerations.Store(0)
}

func TestIconCacheBounded(t *testing.T) {
	resetIconCache()

	star := Icon("star")
	ctx := context.Background()

	// Render with more distinct Class values than the cache cap, which
	// should force at least one eviction (full clear) along the way.
	for i := 0; i < iconCacheCap+500; i++ {
		var buf bytes.Buffer
		p := Props{Class: fmt.Sprintf("class-%d", i)}
		if err := star(p).Render(ctx, &buf); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}

	iconMutex.RLock()
	size := len(iconContents)
	iconMutex.RUnlock()

	if size > iconCacheCap {
		t.Fatalf("iconContents grew to %d entries, want <= %d", size, iconCacheCap)
	}
}

func TestIconCacheServesRepeatedRenderFromCache(t *testing.T) {
	resetIconCache()

	heart := Icon("heart")
	ctx := context.Background()
	p := Props{Class: "repeat-me"}

	var first bytes.Buffer
	if err := heart(p).Render(ctx, &first); err != nil {
		t.Fatalf("first render: %v", err)
	}
	if got := iconGenerations.Load(); got != 1 {
		t.Fatalf("generations after first render = %d, want 1", got)
	}

	var second bytes.Buffer
	if err := heart(p).Render(ctx, &second); err != nil {
		t.Fatalf("second render: %v", err)
	}
	if got := iconGenerations.Load(); got != 1 {
		t.Fatalf("generations after second (cached) render = %d, want still 1", got)
	}

	if first.String() != second.String() {
		t.Fatalf("cached render differs from original:\nfirst:  %s\nsecond: %s", first.String(), second.String())
	}
}

func renderIcon(t *testing.T, name string, p Props) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Icon(name)(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render %s %+v: %v", name, p, err)
	}
	return buf.String()
}

// TestIconEscapesAttributes: every string prop lands HTML-escaped inside its
// attribute, so a quote cannot close the attribute and inject another one,
// and a repeat render (served from cache) is the same escaped markup.
func TestIconEscapesAttributes(t *testing.T) {
	resetIconCache()
	p := Props{
		Class:       `x" onload="alert(1)`,
		Fill:        `"><script>`,
		Stroke:      `a&b`,
		StrokeWidth: `2"`,
	}
	for i := 0; i < 2; i++ { // second pass is the cached copy
		got := renderIcon(t, "star", p)
		for _, bad := range []string{`" onload="`, `<script>`, `a&b`, `"2""`} {
			if strings.Contains(got, bad) {
				t.Errorf("pass %d: output contains raw %q:\n%s", i, bad, got)
			}
		}
		for _, want := range []string{
			`class="x&#34; onload=&#34;alert(1)"`,
			`fill="&#34;&gt;&lt;script&gt;"`,
			`stroke="a&amp;b"`,
			`stroke-width="2&#34;"`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("pass %d: output lacks %q:\n%s", i, want, got)
			}
		}
	}
	if got := iconGenerations.Load(); got != 1 {
		t.Errorf("generations = %d, want 1 (second render cached)", got)
	}
}

// TestIconCacheKeyUnambiguous: two different Props whose values, joined with
// the key's separators, spell the same string must not share a cache entry.
func TestIconCacheKeyUnambiguous(t *testing.T) {
	resetIconCache()
	a := renderIcon(t, "star", Props{Color: "x|f:y"})
	b := renderIcon(t, "star", Props{Color: "x", Fill: "y|f:"})
	if a == b {
		t.Fatalf("distinct props rendered identically (cache key collision):\n%s", b)
	}
	if !strings.Contains(b, `fill="y|f:"`) || !strings.Contains(b, `stroke="x"`) {
		t.Errorf("second render is not its own props:\n%s", b)
	}
}
