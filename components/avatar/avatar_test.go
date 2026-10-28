package avatar

import (
	"bytes"
	"context"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var classRe = regexp.MustCompile(`class="([^"]*)"`)

// classes renders an Avatar and returns its root class tokens (TwMerge
// does not preserve order).
func classes(t *testing.T, p Props) []string {
	t.Helper()
	var buf bytes.Buffer
	if err := Avatar(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	m := classRe.FindStringSubmatch(buf.String())
	if m == nil {
		t.Fatalf("no class attribute in %s", buf.String())
	}
	return strings.Fields(m[1])
}

func TestAvatar_Size(t *testing.T) {
	cases := []struct {
		name      string
		p         Props
		want, not []string
	}{
		{"default", Props{}, []string{"h-10", "w-10"}, nil},
		{"sm", Props{Size: SizeSm}, []string{"h-8", "w-8"}, []string{"h-10"}},
		{"lg", Props{Size: SizeLg}, []string{"h-14", "w-14"}, []string{"h-10"}},
		{"class overrides", Props{Size: SizeLg, Class: "h-8 w-8"}, []string{"h-8", "w-8"}, []string{"h-14"}},
	}
	for _, c := range cases {
		got := classes(t, c.p)
		for _, w := range c.want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: missing %s in %v", c.name, w, got)
			}
		}
		for _, n := range c.not {
			if slices.Contains(got, n) {
				t.Errorf("%s: unexpected %s in %v", c.name, n, got)
			}
		}
	}
}

func renderImage(t *testing.T, p ImageProps) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Image(p).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestAvatarImage_SrcSanitized(t *testing.T) {
	out := renderImage(t, ImageProps{Src: "javascript:alert(1)", Alt: "x"})
	if !strings.Contains(out, `about:invalid`) {
		t.Errorf("javascript: src must be neutralised to about:invalid: %s", out)
	}
	if strings.Contains(out, `javascript:`) {
		t.Errorf("javascript: leaked into src: %s", out)
	}
	if ok := renderImage(t, ImageProps{Src: "/x.png", Alt: "x"}); !strings.Contains(ok, `src="/x.png"`) {
		t.Errorf("normal src must pass through unchanged: %s", ok)
	}
	if empty := renderImage(t, ImageProps{Alt: "x"}); strings.Contains(empty, `src=`) {
		t.Errorf("empty Src must still omit the attribute: %s", empty)
	}
}
