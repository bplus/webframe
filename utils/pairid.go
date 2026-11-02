package utils

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/a-h/templ"
)

// PairedID returns an id that the n-th call for (prefix, role) and the n-th
// call for (prefix, otherRole) in the same templ render agree on:
// prefix + "-" + n, n counting from 0 per role per render. A component pair
// that normally shares an id through its parent's context (a dropdown's
// Trigger and Content) uses it as the fallback when rendered without that
// parent, so the first orphaned Trigger links to the first orphaned Content,
// the second to the second, and two orphaned pairs on one page never
// share an id.
//
// "Render" is templ's per-render state (the value templ.InitializeContext
// puts in ctx, which every component of one Render call shares); the count
// is kept there through one templ.OnceHandle per (prefix, role, n). Two
// separate Render calls each count from 0, so two fragments rendered apart
// (an htmx swap) can still repeat an id: give components rendered that way a
// parent or explicit ids.
func PairedID(ctx context.Context, prefix, role string) (string, error) {
	for n := 0; ; n++ {
		claimed := false
		mark := templ.ComponentFunc(func(context.Context, io.Writer) error {
			claimed = true
			return nil
		})
		if err := pairHandle(prefix, role, n).Once().Render(templ.WithChildren(ctx, mark), io.Discard); err != nil {
			return "", fmt.Errorf("utils.PairedID(%q, %q): %w", prefix, role, err)
		}
		if claimed {
			return fmt.Sprintf("%s-%d", prefix, n), nil
		}
	}
}

type pairKey struct {
	prefix, role string
	n            int
}

var (
	pairMu      sync.Mutex
	pairHandles = map[pairKey]*templ.OnceHandle{}
)

// pairHandle returns the process-wide OnceHandle for one (prefix, role, n)
// slot. The handles are global; whether one has been rendered is per render.
// The map grows to the largest number of orphans one page ever had.
func pairHandle(prefix, role string, n int) *templ.OnceHandle {
	k := pairKey{prefix, role, n}
	pairMu.Lock()
	defer pairMu.Unlock()
	h, ok := pairHandles[k]
	if !ok {
		h = templ.NewOnceHandle()
		pairHandles[k] = h
	}
	return h
}
