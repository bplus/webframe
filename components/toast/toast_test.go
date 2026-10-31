package toast

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func mustNotContain(t *testing.T, out string, nots ...string) {
	t.Helper()
	for _, not := range nots {
		if strings.Contains(out, not) {
			t.Errorf("output has unexpected %q:\n%s", not, out)
		}
	}
}

func TestToast_RolePerKind(t *testing.T) {
	for name, tc := range map[string]struct {
		props Props
		role  string
		tone  string
	}{
		"info zero": {Info("Heads up", ""), `role="status"`, "border-info"},
		"success":   {Success("Saved", ""), `role="status"`, "border-success"},
		"warning":   {Warning("Careful", ""), `role="status"`, "border-warning"},
		"error":     {Error("Failed", ""), `role="alert"`, "border-destructive"},
	} {
		t.Run(name, func(t *testing.T) {
			out := render(t, Toast(tc.props))
			mustContain(t, out, tc.role, tc.tone, "wf-toast", "bg-card")
		})
	}
}

func TestToast_EmptyRendersNothing(t *testing.T) {
	out := render(t, Toast(Props{Kind: KindError, Sticky: true, ID: "x"}))
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output, got:\n%s", out)
	}
}

func TestToast_TitleBodyDismiss(t *testing.T) {
	out := render(t, Toast(Props{ID: "t1", Class: "mt-2", Title: "Title here", Body: "Body here"}))
	mustContain(t, out,
		`id="t1"`, "mt-2", "Title here", "Body here",
		`aria-label="Dismiss"`, `x-on:click="show = false"`,
		`x-data="{ show: true }"`, `x-show="show"`, "x-transition",
	)
}

func TestToast_Timeout(t *testing.T) {
	t.Run("default 4s", func(t *testing.T) {
		out := render(t, Toast(Info("A", "")))
		mustContain(t, out, "setTimeout(() =&gt; show = false, 4000)")
	})
	t.Run("custom", func(t *testing.T) {
		out := render(t, Toast(Props{Title: "A", Timeout: 1500 * time.Millisecond}))
		mustContain(t, out, "show = false, 1500)")
		mustNotContain(t, out, "4000")
	})
	t.Run("sticky has no timer", func(t *testing.T) {
		out := render(t, Toast(Props{Title: "A", Sticky: true, Timeout: time.Second}))
		mustNotContain(t, out, "x-init", "setTimeout")
		mustContain(t, out, `aria-label="Dismiss"`)
	})
}

func TestToast_InlineHasNoOOB(t *testing.T) {
	out := render(t, Toast(Info("A", "B")))
	mustNotContain(t, out, "hx-swap-oob")
}

func TestOOB_AddsAttributeWithoutMutatingCaller(t *testing.T) {
	p := Info("A", "B")
	p.Attributes = templ.Attributes{"data-x": "1"}
	out := render(t, OOB(p))
	mustContain(t, out, `hx-swap-oob="beforeend:#toast-container"`, `data-x="1"`)
	if _, ok := p.Attributes["hx-swap-oob"]; ok {
		t.Error("OOB mutated the caller's Attributes map")
	}
}

func TestSend_NonHX_WritesNothing(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/save", nil)
	w := httptest.NewRecorder()
	if err := Send(w, r, Error("Nope", "x")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got:\n%s", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "" {
		t.Errorf("Send must not set Content-Type, got %q", ct)
	}
}

func TestSend_HX_WritesOOBToast(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/save", nil)
	r.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Send(w, r, Error("Save failed", "Try again")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	out := w.Body.String()
	mustContain(t, out,
		`hx-swap-oob="beforeend:#toast-container"`,
		"Save failed", "Try again", `role="alert"`,
	)
	if w.Code != http.StatusOK {
		t.Errorf("Send must not change the status, got %d", w.Code)
	}
}

func TestToast_EscapesTitleAndBody(t *testing.T) {
	out := render(t, Toast(Info(`<script>alert("t")</script>`, `<b>bold</b> & "q"`)))
	mustNotContain(t, out, "<script>", "<b>")
	mustContain(t, out, "&lt;script&gt;", "&lt;b&gt;bold&lt;/b&gt;", "&amp;")
}
