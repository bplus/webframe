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

func renderSection(t *testing.T, level int) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Section(SectionProps{Title: "T", Level: level}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestSectionLevelDefaultsToH2(t *testing.T) {
	out := renderSection(t, 0)
	if !strings.Contains(out, `<h2 class="text-2xl font-semibold`) || !strings.Contains(out, "</h2>") {
		t.Errorf("want h2 text-2xl in %s", out)
	}
}

func TestSectionLevel3RendersH3(t *testing.T) {
	out := renderSection(t, 3)
	if !strings.Contains(out, `<h3 class="text-lg font-semibold`) || !strings.Contains(out, "</h3>") || strings.Contains(out, "<h2") {
		t.Errorf("want h3 text-lg in %s", out)
	}
}

func TestSectionLevelClamps(t *testing.T) {
	if out := renderSection(t, 9); !strings.Contains(out, `<h4 class="text-base font-semibold`) || !strings.Contains(out, "</h4>") {
		t.Errorf("level 9 want h4 in %s", out)
	}
	if out := renderSection(t, 1); !strings.Contains(out, `<h2 class="text-2xl font-semibold`) || strings.Contains(out, "<h1") {
		t.Errorf("level 1 want h2 in %s", out)
	}
}

func TestStack_PacksAtStart(t *testing.T) {
	var buf bytes.Buffer
	if err := Stack().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "justify-") {
		t.Errorf("a Stack has no justify option and should emit no justify class: %s", buf.String())
	}
}

func TestInline_WrapsOnPhoneByDefault(t *testing.T) {
	var buf bytes.Buffer
	if err := Inline().Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "max-md:flex-wrap") {
		t.Errorf("default Inline should wrap below md: %s", out)
	}
	if strings.Contains(out, " flex-wrap") || strings.Contains(out, `"flex-wrap`) {
		t.Errorf("default Inline must not wrap at every width: %s", out)
	}
	buf.Reset()
	if err := Inline(InlineProps{Wrap: true}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, " flex-wrap") && !strings.Contains(out, `"flex-wrap`) {
		t.Errorf("Wrap: true should wrap at every width: %s", out)
	}
}
