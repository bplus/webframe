package dashboardheader_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"webframe/components/dashboardheader"
)

func TestHeader_TitleOnly(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardheader.Header(dashboardheader.Props{
		Title: "Graph Health",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "Graph Health") {
		t.Errorf("expected title, got: %s", html)
	}
	if !strings.Contains(html, "<h1") {
		t.Errorf("expected h1 tag, got: %s", html)
	}
	if strings.Contains(html, "<p") {
		t.Errorf("expected no subtitle paragraph, got: %s", html)
	}
}

func TestHeader_WithSubtitle(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardheader.Header(dashboardheader.Props{
		Title:    "Knowledge Graph Overview",
		Subtitle: "1,158 nodes across 174 dimensions",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "Knowledge Graph Overview") {
		t.Errorf("expected title, got: %s", html)
	}
	if !strings.Contains(html, "1,158 nodes across 174 dimensions") {
		t.Errorf("expected subtitle, got: %s", html)
	}
}

func TestHeader_WithTimeRange(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardheader.Header(dashboardheader.Props{
		Title:     "Growth Report",
		TimeRange: "Last 30 days",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "Last 30 days") {
		t.Errorf("expected time range badge, got: %s", html)
	}
}

func TestHeader_Sticky(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardheader.Header(dashboardheader.Props{
		Title:  "Monitoring",
		Sticky: true,
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "sticky") {
		t.Errorf("expected sticky class, got: %s", html)
	}
}

func TestHeader_EmptyTitle(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardheader.Header(dashboardheader.Props{})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, "<h1") {
		t.Errorf("expected no h1 tag when title is empty, got: %s", html)
	}
}

func TestHeader_NotSticky(t *testing.T) {
	var buf bytes.Buffer
	comp := dashboardheader.Header(dashboardheader.Props{
		Title: "Overview",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, "sticky") {
		t.Errorf("expected no sticky class, got: %s", html)
	}
}
