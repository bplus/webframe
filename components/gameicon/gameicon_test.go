package gameicon

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var classRe = regexp.MustCompile(`class="([^"]*)"`)

// classes renders p and returns the svg's class tokens (TwMerge does not
// preserve order, so tests compare sets).
func classes(t *testing.T, p Props) []string {
	t.Helper()
	var buf bytes.Buffer
	if err := GameIcon(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	m := classRe.FindStringSubmatch(buf.String())
	if m == nil {
		t.Fatalf("no class attribute in %s", buf.String())
	}
	return strings.Fields(m[1])
}

func TestGameIcon_Classes(t *testing.T) {
	cases := []struct {
		name string
		p    Props
		want []string
	}{
		{"default inherits color", Props{Name: "ring"}, []string{"h-8", "w-8"}},
		{"size lg", Props{Name: "ring", Size: SizeLg}, []string{"h-12", "w-12"}},
		{"tone foreground", Props{Name: "ring", Tone: ToneForeground}, []string{"h-8", "w-8", "text-foreground"}},
		{"tone primary", Props{Name: "ring", Tone: TonePrimary}, []string{"h-8", "w-8", "text-primary"}},
		{"tone accent", Props{Name: "ring", Tone: ToneAccent}, []string{"h-8", "w-8", "text-accent-500"}},
		{"tone muted", Props{Name: "ring", Tone: ToneMuted}, []string{"h-8", "w-8", "text-muted-foreground"}},
		{"class overrides", Props{Name: "ring", Class: "h-4 w-4"}, []string{"h-4", "w-4"}},
	}
	for _, c := range cases {
		got := classes(t, c.p)
		slices.Sort(got)
		want := slices.Clone(c.want)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s: got %v, want %v", c.name, got, want)
		}
	}
}

func TestGameIcon_Href(t *testing.T) {
	var buf bytes.Buffer
	if err := GameIcon(Props{Name: "ring"}).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), `href="/static/icons/game-icons.svg#icon-ring"`) {
		t.Errorf("sprite href missing: %s", buf.String())
	}
}
