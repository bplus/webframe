package trafficstream_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bplus/webframe/components/trafficstream"
)

func TestTrafficStream_Defaults(t *testing.T) {
	var buf bytes.Buffer
	comp := trafficstream.TrafficStream()
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `data-tui-trafficstream`) {
		t.Fatalf("expected trafficstream marker, got: %s", html)
	}
	if !strings.Contains(html, `data-tui-trafficstream-max-items="250"`) {
		t.Fatalf("expected default max items, got: %s", html)
	}
	if !strings.Contains(html, `No traffic yet`) {
		t.Fatalf("expected default empty title, got: %s", html)
	}
	if !strings.Contains(html, `Waiting for request and response events to arrive.`) {
		t.Fatalf("expected default empty description, got: %s", html)
	}
}

func TestTrafficStream_CustomProps(t *testing.T) {
	var buf bytes.Buffer
	comp := trafficstream.TrafficStream(trafficstream.Props{
		ID:              "traffic-pane",
		StreamURL:       "/api/traffic/stream",
		Height:          560,
		MaxItems:        500,
		Class:           "ring-1 ring-primary/20",
		EmptyTitle:      "Nothing captured",
		ProviderOptions: []string{"anthropic", "openai"},
		EventOptions:    []string{"request.received", "response.completed"},
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `id="traffic-pane"`) {
		t.Fatalf("expected id, got: %s", html)
	}
	if !strings.Contains(html, `data-tui-trafficstream-url="/api/traffic/stream"`) {
		t.Fatalf("expected stream url, got: %s", html)
	}
	if !strings.Contains(html, `height:560px`) {
		t.Fatalf("expected height style, got: %s", html)
	}
	if !strings.Contains(html, `data-tui-trafficstream-max-items="500"`) {
		t.Fatalf("expected custom max items, got: %s", html)
	}
	for _, needle := range []string{"anthropic", "openai", "request.received", "response.completed", "Nothing captured", "ring-primary/20"} {
		if !strings.Contains(html, needle) {
			t.Fatalf("expected %q in output, got: %s", needle, html)
		}
	}
}

func TestTrafficStream_InitialRowsHideEmptyState(t *testing.T) {
	var buf bytes.Buffer
	comp := trafficstream.TrafficStream(trafficstream.Props{
		InitialRows: []trafficstream.Row{{
			Time:      "12:00:00",
			Provider:  "anthropic",
			Direction: "request",
			Event:     "request.received",
			Status:    "200",
			TraceID:   "trace-123",
			Summary:   "POST /v1/messages",
		}},
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	for _, needle := range []string{"12:00:00", "anthropic", "request.received", "trace-123", "POST /v1/messages"} {
		if !strings.Contains(html, needle) {
			t.Fatalf("expected %q in output, got: %s", needle, html)
		}
	}
	if !strings.Contains(html, `data-tui-trafficstream-empty`) || !strings.Contains(html, `hidden`) {
		t.Fatalf("expected hidden empty state when rows exist, got: %s", html)
	}
}
