package utils

import (
	"context"
	"testing"

	"github.com/a-h/templ"
)

// TestPairedID: within one render the n-th call per role yields
// prefix-n, the roles count independently (so they pair up), and a new
// render starts again at 0.
func TestPairedID(t *testing.T) {
	ctx := templ.InitializeContext(context.Background())
	get := func(ctx context.Context, role string) string {
		t.Helper()
		id, err := PairedID(ctx, "p", role)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	got := []string{get(ctx, "a"), get(ctx, "b"), get(ctx, "a"), get(ctx, "b"), get(ctx, "a")}
	want := []string{"p-0", "p-0", "p-1", "p-1", "p-2"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("call %d = %q, want %q (all: %q)", i, got[i], want[i], got)
		}
	}
	if id := get(templ.InitializeContext(context.Background()), "a"); id != "p-0" {
		t.Errorf("new render = %q, want p-0", id)
	}
}
