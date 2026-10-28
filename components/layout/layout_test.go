package layout

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/action"
)

func TestGapClass(t *testing.T) {
	tests := []struct {
		g    Gap
		want string
	}{
		{GapNone, "gap-0"},
		{GapXS, "gap-1"},
		{GapS, "gap-2"},
		{GapM, "gap-4"},
		{GapL, "gap-6"},
		{GapXL, "gap-10"},
	}
	for _, tt := range tests {
		if got := gapClass(tt.g); got != tt.want {
			t.Errorf("gapClass(%q) = %q, want %q", tt.g, got, tt.want)
		}
	}
}

func TestStack_Default(t *testing.T) {
	var buf bytes.Buffer
	if err := Stack().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"flex-col", "gap-4", "items-stretch"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestInline_WrapAndJustify(t *testing.T) {
	var buf bytes.Buffer
	if err := Inline(InlineProps{Wrap: true, Justify: JustifyBetween}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"flex-wrap", "justify-between"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestInline_Default(t *testing.T) {
	var buf bytes.Buffer
	if err := Inline().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "items-center") {
		t.Errorf("missing items-center in %s", buf.String())
	}
}

func TestSection_TitleAndActions(t *testing.T) {
	var buf bytes.Buffer
	c := Section(SectionProps{
		Title:   "Members",
		Actions: []action.Action{{Label: "Invite", Href: "/invite"}},
	})
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"<h2", "Members", "Invite", "wf-section", "flex-wrap"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestDivider_Label(t *testing.T) {
	var buf bytes.Buffer
	if err := Divider(DividerProps{Label: "or"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), ">or</span>") {
		t.Errorf("label missing: %s", buf.String())
	}
	buf.Reset()
	if err := Divider().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "<hr") {
		t.Errorf("want <hr: %s", buf.String())
	}
}
