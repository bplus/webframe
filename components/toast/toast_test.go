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

// TestToast_RemovesItselfAfterLeave: a dismissed or timed-out toast leaves
// the DOM. show=false drives the x-show fade; the x-effect then removes
// the element after removeDelayMS, a plain timer so a toast with no
// transition is removed too. x-show stays for the fade; the effect is on
// the root, so $el is the toast itself and the stack collapses behind it.
func TestToast_RemovesItselfAfterLeave(t *testing.T) {
	for name, p := range map[string]Props{
		"timed":  Info("A", "B"),
		"sticky": {Title: "A", Sticky: true},
	} {
		t.Run(name, func(t *testing.T) {
			out := render(t, Toast(p))
			mustContain(t, out,
				`x-show="show"`, "x-transition",
				`x-effect="if (!show) setTimeout(() =&gt; $el.remove(), 300)"`,
			)
			// The effect sits on the toast root, before the body block.
			if strings.Index(out, "x-effect=") > strings.Index(out, "border-l-4") {
				t.Errorf("x-effect must be on the root element:\n%s", out)
			}
		})
	}
	if removeDelayMS <= 75 {
		t.Errorf("removeDelayMS %d must outlast x-transition's default 75ms leave", removeDelayMS)
	}
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
		// No auto-dismiss timer; the removal effect's own setTimeout
		// (x-effect, after show=false) is not one.
		mustNotContain(t, out, "x-init", "show = false, ")
		mustContain(t, out, `aria-label="Dismiss"`, "x-effect=")
	})
}

func TestToast_InlineHasNoOOB(t *testing.T) {
	out := render(t, Toast(Info("A", "B")))
	mustNotContain(t, out, "hx-swap-oob")
}

func TestOOB_WrapsTheToast(t *testing.T) {
	p := Info("A", "B")
	p.Attributes = templ.Attributes{"data-x": "1"}
	out := render(t, OOB(p))
	// The oob attribute sits on a wrapper whose only child is the toast
	// root (htmx inserts an oob element's content for beforeend), so the
	// root keeps its role, Alpine state and dismiss button.
	if !strings.HasPrefix(out, `<div hx-swap-oob="beforeend:#toast-container"><div `) {
		t.Errorf("OOB does not start with the wrapper then the toast root:\n%s", out)
	}
	mustContain(t, out, `role="status"`, `x-data`, `data-x="1"`)
	if _, ok := p.Attributes["hx-swap-oob"]; ok {
		t.Error("OOB mutated the caller's Attributes map")
	}
	if out := render(t, OOB(Props{})); out != "" {
		t.Errorf("empty OOB rendered %q", out)
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

func TestRespond_ToastOnlyResponse(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/items/1", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	if err := Respond(rec, req, http.StatusConflict, Error("Could not delete", "Still in use")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict || rec.Header().Get("HX-Reswap") != "none" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Errorf("status %d, HX-Reswap %q, Content-Type %q", rec.Code, rec.Header().Get("HX-Reswap"), rec.Header().Get("Content-Type"))
	}
	if body := rec.Body.String(); !strings.Contains(body, `hx-swap-oob="beforeend:#toast-container"`) || !strings.Contains(body, "Could not delete") {
		t.Errorf("body lacks the oob toast:\n%s", body)
	}
	// Not htmx: plain text with the status, no HTML.
	rec = httptest.NewRecorder()
	if err := Respond(rec, httptest.NewRequest("DELETE", "/items/1", nil), http.StatusConflict, Error("Could not delete", "Still in use")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict || strings.Contains(rec.Body.String(), "<") || !strings.Contains(rec.Body.String(), "Still in use") {
		t.Errorf("plain response: %d %q", rec.Code, rec.Body.String())
	}
	// Zero status is 200.
	rec = httptest.NewRecorder()
	_ = Respond(rec, req, 0, Info("Queued", ""))
	if rec.Code != http.StatusOK {
		t.Errorf("zero status = %d", rec.Code)
	}
}
