package icon

import (
	"bytes"
	"context"
	"fmt"
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
