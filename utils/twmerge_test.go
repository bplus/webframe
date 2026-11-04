package utils

import (
	"fmt"
	"testing"
)

// TestTwMerge_KeepsInputOrder guards against random class order. The
// upstream merge builds its result by ranging over a Go map and caches the
// answer per input string, so one input looks stable inside a process while
// a fresh process can print the same classes in another order. Each
// iteration uses a new input so the cache never hides the problem.
func TestTwMerge_KeepsInputOrder(t *testing.T) {
	for i := 0; i < 300; i++ {
		in := fmt.Sprintf("w-full mt-[%dpx] p-2 text-sm", i)
		if got := TwMerge(in); got != in {
			t.Fatalf("TwMerge(%q) = %q, want the input order kept", in, got)
		}
	}
}

func TestTwMerge(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"no conflict keeps order", []string{"w-full", "mt-4"}, "w-full mt-4"},
		{"later conflicting class wins in its own place", []string{"p-2 m-1", "p-4"}, "m-1 p-4"},
		{"conflict across modifiers is separate", []string{"p-2 hover:p-3", "p-4"}, "hover:p-3 p-4"},
		{"non-tailwind classes stay in place", []string{"wf-x w-full", "my-card mt-4"}, "wf-x w-full my-card mt-4"},
		{"repeated non-tailwind class is kept", []string{"a b", "a"}, "a b a"},
		{"repeated tailwind class keeps the last", []string{"p-2 m-1 p-2"}, "m-1 p-2"},
		{"empty parts", []string{"", "w-full", ""}, "w-full"},
		{"all empty", []string{"", "  "}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TwMerge(tc.in...); got != tc.want {
				t.Fatalf("TwMerge(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
