// templui util templui.go - version: v1.3.0 installed by templui v1.3.0

package utils

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"crypto/rand"

	"github.com/a-h/templ"

	twmerge "github.com/Oudwins/tailwind-merge-go"
)

// TwMerge combines Tailwind classes and resolves conflicts.
// Example: "bg-red-500 hover:bg-blue-500", "bg-green-500" → "hover:bg-blue-500 bg-green-500"
//
// Later arguments win on conflicting utilities, which is why every component
// puts its caller-supplied props.Class last — consumers can override component
// defaults. Always use this instead of string concatenation, which would emit
// both conflicting classes and leave the outcome to CSS order.
//
// The result keeps the input order: each surviving class sits where its last
// occurrence was. The upstream merge, tailwind-merge-go v0.2.1, decides which
// classes survive but returns them in Go map order (with every non-Tailwind
// class first), so the order changes from one process to the next. inputOrder
// restores the input order, so the same props render the same HTML on every
// run. Do not remove it while the module is on a version with that bug; a
// test (TestTwMerge_KeepsInputOrder) fails if the order ever drifts again.
func TwMerge(classes ...string) string {
	return inputOrder(classes, twmerge.Merge(classes...))
}

// inputOrder returns the classes in merged, ordered by the position of their
// last occurrence in parts. merged is always drawn from parts, so every class
// in it is found. Counting handles a class the merge kept more than once
// (upstream keeps repeated non-Tailwind classes).
func inputOrder(parts []string, merged string) string {
	keep := strings.Fields(merged)
	if len(keep) < 2 {
		return merged
	}
	left := make(map[string]int, len(keep))
	for _, c := range keep {
		left[c]++
	}
	var all []string
	for _, p := range parts {
		all = append(all, strings.Fields(p)...)
	}
	out := make([]string, 0, len(keep))
	for i := len(all) - 1; i >= 0; i-- {
		if c := all[i]; left[c] > 0 {
			left[c]--
			out = append(out, c)
		}
	}
	slices.Reverse(out)
	return strings.Join(out, " ")
}

// If returns value if condition is true, otherwise the zero value of type T.
// Example: true, "bg-red-500" → "bg-red-500"
//
// Intended for conditional Tailwind classes inside templ expressions, where
// a zero value ("" for strings) is a harmless no-op. The value argument is
// evaluated eagerly regardless of condition.
func If[T comparable](condition bool, value T) T {
	var empty T
	if condition {
		return value
	}
	return empty
}

// IfElse returns trueValue if condition is true, otherwise falseValue.
// Example: true, "bg-red-500", "bg-gray-300" → "bg-red-500"
//
// A ternary substitute for templ expressions. Both branches are evaluated
// eagerly, so don't pass expensive or side-effecting expressions.
func IfElse[T any](condition bool, trueValue T, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

// MergeAttributes combines multiple Attributes into one.
// Example: MergeAttributes(attr1, attr2) → combined attributes
//
// On duplicate keys the later map wins (last-write). A new map is always
// returned; the input maps are never mutated, so components can safely merge
// caller-supplied Attributes with their own defaults. Map iteration order is
// irrelevant across maps since only key collisions matter.
//
// Last-wins applies only among the Attributes maps passed into this call.
// Once the merged result is spread onto a templ element, it cannot override
// attributes the template already wrote literally: templ appends spread
// attributes rather than replacing, and HTML parsers keep the first
// occurrence of a duplicate attribute — so a template's own hard-coded
// attributes always win over anything merged in this way.
func MergeAttributes(attrs ...templ.Attributes) templ.Attributes {
	merged := templ.Attributes{}
	for _, attr := range attrs {
		for k, v := range attr {
			merged[k] = v
		}
	}
	return merged
}

// RandomID generates a random ID string. crypto/rand.Text yields a 26-char
// uppercase RFC 4648 base32 token (A-Z, 2-7; ≥128 bits of entropy).
// Example: RandomID() → "id-4B25NVMEWOLI3D22HBI4XMQY6R"
//
// Used by components to link elements that need matching id/for/aria
// attributes when the caller doesn't supply an ID. The "id-" prefix
// guarantees a letter-first value that is always a valid HTML id. IDs are
// unique per call, not stable across renders — don't use them as anchors or
// hx-target references that must survive a re-render.
func RandomID() string {
	return fmt.Sprintf("id-%s", rand.Text())
}

// ScriptVersion is a timestamp generated at app start for cache busting.
// Used in Script() templates to append ?v=<timestamp> to script URLs.
// Fixed at package init and constant for the process lifetime, so every
// restart busts browser caches while a running process stays consistent.
// Multiple instances behind a load balancer stamp different ?v= values, so
// clients re-download scripts when they switch instances.
var ScriptVersion = fmt.Sprintf("%d", time.Now().Unix())

// ScriptURL generates cache-busted script URLs.
// Override this to use custom cache busting (CDN, content hashing, etc.)
//
// Example override in your app:
//
//	func init() {
//	    utils.ScriptURL = func(path string) string {
//	        return myAssetManifest.GetURL(path)
//	    }
//	}
//
// Override it in init() or before the first render: swapping it while serving
// is unsynchronized and would mix URL schemes across pages.
var ScriptURL = func(path string) string {
	return path + "?v=" + ScriptVersion
}
