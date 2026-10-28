package badge

import "strings"

// statusVariants maps a normalized status word (lower-case, trimmed) to the
// badge Variant that conveys it. Words not listed here fall back to
// VariantOutline in Status.
var statusVariants = map[string]Variant{
	// Success
	"published": VariantSuccess,
	"done":      VariantSuccess,
	"success":   VariantSuccess,
	"merged":    VariantSuccess,
	"active":    VariantSuccess,
	"ok":        VariantSuccess,
	"passed":    VariantSuccess,
	"live":      VariantSuccess,
	// Secondary
	"draft":   VariantSecondary,
	"pending": VariantSecondary,
	"queued":  VariantSecondary,
	"paused":  VariantSecondary,
	"idle":    VariantSecondary,
	"new":     VariantSecondary,
	// Destructive
	"failed":   VariantDestructive,
	"error":    VariantDestructive,
	"blocked":  VariantDestructive,
	"rejected": VariantDestructive,
	"dead":     VariantDestructive,
	// Info
	"running":     VariantInfo,
	"building":    VariantInfo,
	"in-progress": VariantInfo,
	"syncing":     VariantInfo,
	"open":        VariantInfo,
	// Warning
	"warning":  VariantWarning,
	"stale":    VariantWarning,
	"degraded": VariantWarning,
	"retrying": VariantWarning,
	"flaky":    VariantWarning,
}

// Status maps a status word to badge Props so apps never pick colors.
// Matching is case-insensitive and ignores surrounding spaces. The badge
// text is still the caller's children; use StatusLabel for a display form.
//
//	published, done, success, merged, active, ok, passed, live → Success
//	draft, pending, queued, paused, idle, new                → Secondary
//	failed, error, blocked, rejected, dead                    → Destructive
//	running, building, in-progress, syncing, open             → Info
//	warning, stale, degraded, retrying, flaky                 → Warning
//	anything else                                             → Outline
//
// @pure
func Status(kind string) Props {
	key := strings.ToLower(strings.TrimSpace(kind))
	if v, ok := statusVariants[key]; ok {
		return Props{Variant: v}
	}
	return Props{Variant: VariantOutline}
}

// StatusLabel turns a status word into display text: surrounding spaces
// are trimmed, dashes and underscores become spaces, and only the first
// letter is upper-cased ("in-progress" → "In progress", "FAILED" → "Failed").
// An empty or all-space input yields "".
//
// @pure
func StatusLabel(kind string) string {
	s := strings.TrimSpace(kind)
	if s == "" {
		return ""
	}
	s = strings.NewReplacer("-", " ", "_", " ").Replace(s)
	s = strings.ToLower(s)
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}
