package emptystate_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"webframe/components/emptystate"
)

func TestEmptyState_DefaultProps(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState()
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "<div") {
		t.Errorf("expected wrapper div, got: %s", html)
	}
	// No title/description/icon when props are empty
	if strings.Contains(html, "<h3") {
		t.Errorf("expected no h3 when title is empty, got: %s", html)
	}
	if strings.Contains(html, "<p") {
		t.Errorf("expected no p when description is empty, got: %s", html)
	}
}

func TestEmptyState_WithTitle(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState(emptystate.Props{
		Title: "No nodes found",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "No nodes found") {
		t.Errorf("expected title text, got: %s", html)
	}
	if !strings.Contains(html, "<h3") {
		t.Errorf("expected h3 tag, got: %s", html)
	}
}

func TestEmptyState_WithDescription(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState(emptystate.Props{
		Title:       "No results",
		Description: "Try adjusting your search filters",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "No results") {
		t.Errorf("expected title, got: %s", html)
	}
	if !strings.Contains(html, "Try adjusting your search filters") {
		t.Errorf("expected description, got: %s", html)
	}
	if !strings.Contains(html, "<p") {
		t.Errorf("expected p tag for description, got: %s", html)
	}
}

func TestEmptyState_WithIcon(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState(emptystate.Props{
		Title: "Empty graph",
		Icon:  "🔍",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "🔍") {
		t.Errorf("expected icon, got: %s", html)
	}
	if !strings.Contains(html, "text-4xl") {
		t.Errorf("expected icon styling, got: %s", html)
	}
}

func TestEmptyState_WithID(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState(emptystate.Props{
		ID:    "empty-search",
		Title: "Nothing here",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, `id="empty-search"`) {
		t.Errorf("expected id attribute, got: %s", html)
	}
}

func TestEmptyState_WithCustomClass(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState(emptystate.Props{
		Title: "No data",
		Class: "bg-muted rounded-lg",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if !strings.Contains(html, "bg-muted") {
		t.Errorf("expected custom class, got: %s", html)
	}
}

func TestEmptyState_NoIconOmitsIconDiv(t *testing.T) {
	var buf bytes.Buffer
	comp := emptystate.EmptyState(emptystate.Props{
		Title: "No nodes",
	})
	if err := comp.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, "text-4xl") {
		t.Errorf("expected no icon div when icon is empty, got: %s", html)
	}
}
