package badge

import "strings"

// statusVariants maps a lower-cased, trimmed status word to its Variant.
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
func Status(kind string) Props {
	key := strings.ToLower(strings.TrimSpace(kind))
	if v, ok := statusVariants[key]; ok {
		return Props{Variant: v}
	}
	return Props{Variant: VariantOutline}
}

// StatusPtr is Status for fields that take *Props, such as
// datatable.Cell.Badge. datatable.BadgeCell and descriptionlist.BadgeItem
// wrap it.
func StatusPtr(kind string) *Props {
	p := Status(kind)
	return &p
}

// StatusLabel turns a status word into display text: surrounding spaces
// are trimmed, dashes and underscores become spaces, and only the first
// letter is upper-cased ("in-progress" → "In progress", "FAILED" → "Failed").
// An empty or all-space input yields "".
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
