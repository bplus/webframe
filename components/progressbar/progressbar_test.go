package progressbar

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestClampPercent(t *testing.T) {
	tests := []struct {
		value int
		max   int
		want  int
	}{
		{0, 0, 0},
		{40, 0, 40},
		{50, 100, 50},
		{1, 3, 33},
		{2, 3, 66},
		{-5, 100, 0},
		{150, 100, 100},
		{5, -1, 5},
		{7, 7, 100},
		{0, 100, 0},
	}
	for _, tt := range tests {
		got := clampPercent(tt.value, tt.max)
		if got != tt.want {
			t.Errorf("clampPercent(%d, %d) = %d, want %d", tt.value, tt.max, got, tt.want)
		}
	}
}

func TestProgressBar_ShowValue(t *testing.T) {
	var buf bytes.Buffer
	comp := ProgressBar(Props{Value: 40, ShowValue: true})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "40%") {
		t.Errorf("expected 40%% text, got: %s", html)
	}
	if !strings.Contains(html, "width: 40%") {
		t.Errorf("expected fill width of 40%%, got: %s", html)
	}
}
