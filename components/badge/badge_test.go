package badge

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestBadge_NewVariants(t *testing.T) {
	cases := []struct {
		variant Variant
		want    []string
	}{
		{VariantSuccess, []string{"text-success", "bg-success/10", "border-success/30"}},
		{VariantWarning, []string{"text-warning", "bg-warning/10", "border-warning/30"}},
		{VariantInfo, []string{"text-info", "bg-info/10", "border-info/30"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.variant), func(t *testing.T) {
			html := render(t, Badge(Props{Variant: tc.variant}))
			for _, w := range tc.want {
				if !strings.Contains(html, w) {
					t.Errorf("variant %q missing class %q in %s", tc.variant, w, html)
				}
			}
			// Soft chips: never the solid primary fill.
			if strings.Contains(html, "bg-primary ") {
				t.Errorf("variant %q fell back to primary styling: %s", tc.variant, html)
			}
		})
	}
}

// The four pre-existing variants must render exactly as before.
func TestBadge_ExistingVariantsUnchanged(t *testing.T) {
	cases := []struct {
		variant Variant
		want    string
	}{
		{VariantDefault, "bg-primary"},
		{"", "bg-primary"},
		{VariantSecondary, "bg-secondary"},
		{VariantDestructive, "bg-destructive"},
		{VariantOutline, "text-foreground"},
	}
	for _, tc := range cases {
		html := render(t, Badge(Props{Variant: tc.variant}))
		if !strings.Contains(html, tc.want) {
			t.Errorf("variant %q missing %q in %s", tc.variant, tc.want, html)
		}
	}
}

func TestStatus_Mapping(t *testing.T) {
	cases := map[string]Variant{
		// Success
		"published": VariantSuccess, "done": VariantSuccess, "success": VariantSuccess,
		"merged": VariantSuccess, "active": VariantSuccess, "ok": VariantSuccess,
		"passed": VariantSuccess, "live": VariantSuccess,
		// Secondary
		"draft": VariantSecondary, "pending": VariantSecondary, "queued": VariantSecondary,
		"paused": VariantSecondary, "idle": VariantSecondary, "new": VariantSecondary,
		// Destructive
		"failed": VariantDestructive, "error": VariantDestructive, "blocked": VariantDestructive,
		"rejected": VariantDestructive, "dead": VariantDestructive,
		// Info
		"running": VariantInfo, "building": VariantInfo, "in-progress": VariantInfo,
		"syncing": VariantInfo, "open": VariantInfo,
		// Warning
		"warning": VariantWarning, "stale": VariantWarning, "degraded": VariantWarning,
		"retrying": VariantWarning, "flaky": VariantWarning,
		// Normalization: case and surrounding space.
		"Published ": VariantSuccess, "  FAILED": VariantDestructive, "In-Progress": VariantInfo,
		// Unknown
		"weird": VariantOutline, "": VariantOutline,
	}
	for kind, want := range cases {
		if got := Status(kind).Variant; got != want {
			t.Errorf("Status(%q).Variant = %q, want %q", kind, got, want)
		}
	}
}

func TestStatus_RendersChildrenText(t *testing.T) {
	// The badge text is still the caller's children; Status only picks the variant.
	c := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "Published")
		return err
	})
	var buf bytes.Buffer
	if err := Badge(Status("published")).Render(templ.WithChildren(context.Background(), c), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()
	for _, w := range []string{">Published<", "text-success"} {
		if !strings.Contains(html, w) {
			t.Errorf("missing %q in %s", w, html)
		}
	}
}

func TestStatusLabel(t *testing.T) {
	cases := map[string]string{
		"in-progress":  "In progress",
		"in_progress":  "In progress",
		"published":    "Published",
		" Published ":  "Published",
		"FAILED":       "Failed",
		"not-yet_done": "Not yet done",
		"":             "",
	}
	for in, want := range cases {
		if got := StatusLabel(in); got != want {
			t.Errorf("StatusLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
