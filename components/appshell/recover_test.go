package appshell

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// captureLog routes the standard logger into a buffer for the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return &buf
}

// TestMiddleware_RequestID: the id is the X-Request-Id header when sent,
// else 16 hex digits made per request; RequestID reads it back and is ""
// without Middleware.
func TestMiddleware_RequestID(t *testing.T) {
	var seen []string
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, RequestID(r.Context()))
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "from-proxy")
	h.ServeHTTP(httptest.NewRecorder(), req)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if len(seen) != 3 || seen[0] != "from-proxy" {
		t.Fatalf("ids = %q", seen)
	}
	hex16 := regexp.MustCompile(`^[0-9a-f]{16}$`)
	if !hex16.MatchString(seen[1]) || !hex16.MatchString(seen[2]) || seen[1] == seen[2] {
		t.Errorf("generated ids = %q, want two distinct 16-hex-digit ids", seen[1:])
	}
	if got := RequestID(httptest.NewRequest("GET", "/", nil).Context()); got != "" {
		t.Errorf("RequestID without Middleware = %q, want empty", got)
	}
}

// TestMiddleware_RecoversBeforeHeaders: a panic before any write is logged
// with method, path, id, value and a stack, and the default OnPanic
// answers 500 with a bare document (no shell) naming the id.
func TestMiddleware_RecoversBeforeHeaders(t *testing.T) {
	logs := captureLog(t)
	a := testApp()
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // set, never sent
		panic("kaboom")
	}))
	req := httptest.NewRequest("POST", "/inbox/7", nil)
	req.Header.Set("X-Request-Id", "rid-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
	checkRenderHeaders(t, rec)
	body := rec.Body.String()
	if !strings.Contains(strings.ToLower(body), "<!doctype html>") {
		t.Errorf("missing doctype in:\n%s", body)
	}
	for _, want := range []string{"<title>Something went wrong — Demo</title>", "Something went wrong", "Request id: rid-1"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, "wf-shell") {
		t.Errorf("default 500 must be bare (no shell), got:\n%s", body)
	}
	got := logs.String()
	if !strings.Contains(got, "appshell: panic serving POST /inbox/7 (request rid-1): kaboom") {
		t.Errorf("log line missing, got:\n%s", got)
	}
	if !strings.Contains(got, "goroutine ") || !strings.Contains(got, "recover_test.go") {
		t.Errorf("stack missing from log, got:\n%s", got)
	}
}

// TestMiddleware_RecoverFragmentForHX: a plain htmx request gets the
// page alone, like RenderBare, so htmx-errors.js swaps it (text/html).
func TestMiddleware_RecoverFragmentForHX(t *testing.T) {
	captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("x") }))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 500 || strings.Contains(strings.ToLower(rec.Body.String()), "<!doctype") || !strings.Contains(rec.Body.String(), "Request id: ") {
		t.Errorf("%d:\n%s", rec.Code, rec.Body.String())
	}
}

// TestMiddleware_OnPanicHook: an app's own OnPanic gets the writer, the
// request (with the App in context), the id and the value, and the default
// is not written.
func TestMiddleware_OnPanicHook(t *testing.T) {
	captureLog(t)
	a := testApp()
	var gotID string
	var gotErr any
	a.OnPanic = func(w http.ResponseWriter, r *http.Request, id string, err any) {
		gotID, gotErr = id, err
		if _, ok := FromContext(r.Context()); !ok {
			t.Errorf("OnPanic request has no App in context")
		}
		if RequestID(r.Context()) != id {
			t.Errorf("RequestID(ctx) = %q, id = %q", RequestID(r.Context()), id)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "custom")
	}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic(io.ErrUnexpectedEOF) }))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "rid-2")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 503 || rec.Body.String() != "custom" {
		t.Errorf("%d %q", rec.Code, rec.Body.String())
	}
	if gotID != "rid-2" || gotErr != io.ErrUnexpectedEOF {
		t.Errorf("OnPanic got id=%q err=%v", gotID, gotErr)
	}
}

// TestMiddleware_PanicAfterHeadersLogsOnly: once the handler has written
// (or flushed), the recover path logs and writes nothing more; over a real
// connection the client gets what was sent and a cleanly terminated
// response, not a dropped connection.
func TestMiddleware_PanicAfterHeadersLogsOnly(t *testing.T) {
	logs := captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, "partial")
		w.(http.Flusher).Flush() // the handler sees a Flusher through the wrapper
		panic("late")
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/late", nil)
	req.Header.Set("X-Request-Id", "rid-3")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v (the connection should end cleanly)", err)
	}
	if res.StatusCode != http.StatusAccepted || string(body) != "partial" || res.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("%d %q %q", res.StatusCode, body, res.Header.Get("Content-Type"))
	}
	got := logs.String()
	if !strings.Contains(got, "appshell: panic serving GET /late (request rid-3): late") {
		t.Errorf("log line missing, got:\n%s", got)
	}
	if strings.Contains(got, "http: panic serving") {
		t.Errorf("net/http logged the panic itself; Middleware did not recover it:\n%s", got)
	}
}

// TestMiddleware_PanicAfterWriteHeaderOnly: WriteHeader alone counts as
// written (the status line is gone), so no 500 body is appended.
func TestMiddleware_PanicAfterWriteHeaderOnly(t *testing.T) {
	captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		panic("after status")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 204 || rec.Body.Len() != 0 {
		t.Errorf("%d %q, want 204 and an empty body", rec.Code, rec.Body.String())
	}
}

// TestMiddleware_ErrAbortHandlerPropagates: net/http's own abort sentinel
// keeps its meaning (a quiet connection abort), so Middleware re-panics it
// instead of answering 500.
func TestMiddleware_ErrAbortHandlerPropagates(t *testing.T) {
	logs := captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("recovered %v, want http.ErrAbortHandler", v)
		}
		if logs.Len() != 0 {
			t.Errorf("ErrAbortHandler must not be logged, got:\n%s", logs.String())
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}

// TestMiddleware_NoPanicUntouched: without a panic the wrapper is
// transparent: status, headers and body pass through unchanged.
func TestMiddleware_NoPanicUntouched(t *testing.T) {
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "made")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/", nil))
	if rec.Code != 201 || rec.Body.String() != "made" || rec.Header().Get("X-Custom") != "yes" {
		t.Errorf("%d %q %q", rec.Code, rec.Body.String(), rec.Header().Get("X-Custom"))
	}
}
