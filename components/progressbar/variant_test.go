package progressbar

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// Variant picks the fill from theme tokens; the zero value is primary. App
// code never passes a class name.
func TestProgressBar_Variant(t *testing.T) {
	cases := map[Variant]string{
		VariantDefault:     "bg-primary",
		VariantSuccess:     "bg-success",
		VariantWarning:     "bg-warning",
		VariantDestructive: "bg-destructive",
		VariantInfo:        "bg-info",
		Variant("nope"):    "bg-primary",
	}
	for v, want := range cases {
		var buf bytes.Buffer
		if err := ProgressBar(Props{Value: 50, Variant: v}).Render(context.Background(), &buf); err != nil {
			t.Fatalf("render: %v", err)
		}
		if !strings.Contains(buf.String(), "h-full rounded-full transition-all "+want) {
			t.Errorf("Variant %q: want fill class %q in %s", v, want, buf.String())
		}
	}
}
