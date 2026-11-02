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

// TestMiddleware_OnPanicPanics: a hook that panics before writing is logged
// (a second line, with the first still there) and the default bare page
// answers 500, so a broken error page does not drop the connection.
func TestMiddleware_OnPanicPanics(t *testing.T) {
	logs := captureLog(t)
	a := testApp()
	a.OnPanic = func(w http.ResponseWriter, r *http.Request, id string, err any) {
		w.Header().Set("HX-Retarget", "#somewhere") // set, never sent
		panic("hook broke")
	}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("first") }))
	req := httptest.NewRequest("GET", "/broken", nil)
	req.Header.Set("X-Request-Id", "rid-4")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Request id: rid-4") || !strings.Contains(strings.ToLower(body), "<!doctype html>") {
		t.Errorf("default page missing:\n%s", body)
	}
	if rec.Header().Get("HX-Retarget") != "" {
		t.Errorf("header set by the broken hook leaked: %q", rec.Header().Get("HX-Retarget"))
	}
	got := logs.String()
	for _, want := range []string{
		"appshell: panic serving GET /broken (request rid-4): first",
		"appshell: OnPanic panicked for GET /broken (request rid-4): hook broke",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("log line %q missing, got:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "appshell: "); n != 2 {
		t.Errorf("want exactly two log lines, got %d:\n%s", n, got)
	}
}

// TestMiddleware_OnPanicPanicsAfterWrite: a hook that panics after writing
// is logged only; nothing is appended.
func TestMiddleware_OnPanicPanicsAfterWrite(t *testing.T) {
	logs := captureLog(t)
	a := testApp()
	a.OnPanic = func(w http.ResponseWriter, r *http.Request, id string, err any) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "half")
		panic("late hook")
	}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("first") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 502 || rec.Body.String() != "half" {
		t.Errorf("%d %q, want 502 \"half\" untouched", rec.Code, rec.Body.String())
	}
	if !strings.Contains(logs.String(), "OnPanic panicked") {
		t.Errorf("second panic not logged:\n%s", logs.String())
	}
}

// TestMiddleware_StripsRedirectHeadersOn500: a handler that had set a
// redirect (HTTP or htmx) before panicking gets a plain 500, not a 500
// that redirects, retargets or rewrites the URL bar; unrelated headers
// stay.
func TestMiddleware_StripsRedirectHeadersOn500(t *testing.T) {
	captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, k := range redirectHeaders {
			w.Header().Set(k, "/elsewhere")
		}
		w.Header().Set("X-Keep", "yes")
		panic("mid-redirect")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/", nil))
	if rec.Code != 500 {
		t.Fatalf("status %d, want 500", rec.Code)
	}
	for _, k := range redirectHeaders {
		if v := rec.Header().Get(k); v != "" {
			t.Errorf("%s = %q, want stripped", k, v)
		}
	}
	if rec.Header().Get("X-Keep") != "yes" {
		t.Errorf("unrelated header dropped")
	}
	if len(redirectHeaders) != 8 || redirectHeaders[0] != "Location" {
		t.Errorf("redirectHeaders = %q", redirectHeaders)
	}
}

// TestMiddleware_NestedSharesOneID: an inner Middleware reuses the id the
// outer one stored, over a differing X-Request-Id header, so the log line
// and the 500 page name one id.
func TestMiddleware_NestedSharesOneID(t *testing.T) {
	var inner, outer string
	outerApp, innerApp := testApp(), testApp()
	h := outerApp.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outer = RequestID(r.Context())
		innerApp.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inner = RequestID(r.Context())
		})).ServeHTTP(w, r)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "from-proxy")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if outer != "from-proxy" || inner != outer {
		t.Errorf("outer %q inner %q, want one id from the header", outer, inner)
	}
	outer, inner = "", ""
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if outer == "" || inner != outer {
		t.Errorf("outer %q inner %q, want one generated id", outer, inner)
	}
}

// flushThrough serves h over a real connection and checks that the first
// chunk reaches the client before the handler finishes: the handler writes
// "first", flushes, and waits for the test to have read it before writing
// "second".
func flushThrough(t *testing.T, h http.Handler) {
	t.Helper()
	srv := httptest.NewServer(h)
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()
	buf := make([]byte, 5)
	if _, err := io.ReadFull(res.Body, buf); err != nil || string(buf) != "first" {
		t.Fatalf("first chunk = %q %v, want \"first\" before the handler returns", buf, err)
	}
	close(flushGate)
	rest, err := io.ReadAll(res.Body)
	if err != nil || string(rest) != "second" {
		t.Errorf("rest = %q %v", rest, err)
	}
}

var flushGate chan struct{}

// TestMiddleware_FlushThroughController: a handler flushing with
// http.NewResponseController(w).Flush() flushes through the wrapper.
func TestMiddleware_FlushThroughController(t *testing.T) {
	flushGate = make(chan struct{})
	flushThrough(t, testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "first")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("Flush: %v", err)
		}
		<-flushGate
		_, _ = io.WriteString(w, "second")
	})))
}

// TestMiddleware_FlushThroughFlusher: the same with the classic
// w.(http.Flusher) assertion an SSE handler makes.
func TestMiddleware_FlushThroughFlusher(t *testing.T) {
	flushGate = make(chan struct{})
	flushThrough(t, testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "first")
		f, ok := w.(http.Flusher)
		if !ok {
			t.Errorf("wrapper is not an http.Flusher")
			return
		}
		f.Flush()
		<-flushGate
		_, _ = io.WriteString(w, "second")
	})))
}

// TestMiddleware_HijackReachable: the wrapper is an http.Hijacker and the
// hijack goes through to the real connection: the handler takes it over
// and writes a raw response the client reads.
func TestMiddleware_HijackReachable(t *testing.T) {
	logs := captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Errorf("wrapper is not an http.Hijacker")
			return
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			t.Errorf("Hijack: %v", err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 3\r\nConnection: close\r\n\r\nraw")
		_ = rw.Flush()
		panic("after hijack") // logged; nothing may be written onto the hijacked connection
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || string(body) != "raw" {
		t.Errorf("%d %q, want the hijacked raw response", res.StatusCode, body)
	}
	if got := logs.String(); !strings.Contains(got, "(request ") || strings.Contains(got, "render default 500") || strings.Contains(got, "hijacked") {
		t.Errorf("want the panic logged and no 500 attempted on the hijacked connection, got:\n%s", got)
	}
	// Against a recorder (no Hijack underneath) the assertion still holds
	// and the call reports the error instead of panicking.
	rec := httptest.NewRecorder()
	testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, err := w.(http.Hijacker).Hijack(); err == nil {
			t.Errorf("Hijack on a recorder should fail")
		}
	})).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
}

// readerFromRecorder is a recorder that, like net/http's response, has a
// ReadFrom; the wrapper must hand the copy to it.
type readerFromRecorder struct {
	*httptest.ResponseRecorder
	readFromCalls int
}

func (r *readerFromRecorder) ReadFrom(src io.Reader) (int64, error) {
	r.readFromCalls++
	return io.Copy(r.ResponseRecorder, src)
}

// TestMiddleware_ReadFromPassthrough: io.Copy onto the wrapper (what
// http.ServeContent does) reaches the underlying writer's ReadFrom when it
// has one, falls back to Write when it has none, and counts as written
// either way so a later panic appends no 500.
func TestMiddleware_ReadFromPassthrough(t *testing.T) {
	captureLog(t)
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(io.ReaderFrom); !ok {
			t.Errorf("wrapper is not an io.ReaderFrom")
		}
		// struct{io.Reader} hides strings.Reader's WriteTo, which io.Copy
		// would otherwise prefer over the destination's ReadFrom.
		if _, err := io.Copy(w, struct{ io.Reader }{strings.NewReader("payload")}); err != nil {
			t.Errorf("copy: %v", err)
		}
		panic("after copy")
	}))
	rf := &readerFromRecorder{ResponseRecorder: httptest.NewRecorder()}
	h.ServeHTTP(rf, httptest.NewRequest("GET", "/", nil))
	if rf.readFromCalls != 1 || rf.Body.String() != "payload" || rf.Code != 200 {
		t.Errorf("ReadFrom calls %d, body %q, code %d", rf.readFromCalls, rf.Body.String(), rf.Code)
	}
	plain := httptest.NewRecorder()
	h.ServeHTTP(plain, httptest.NewRequest("GET", "/", nil))
	if plain.Body.String() != "payload" || plain.Code != 200 {
		t.Errorf("fallback: body %q, code %d", plain.Body.String(), plain.Code)
	}
}

// TestMiddleware_RequestIDRejectsUnsafeHeader: an X-Request-Id longer than
// 64 bytes or with a byte outside [A-Za-z0-9._-] (a newline that would forge
// a log line, a space, a quote) is replaced by a generated id; a safe one up
// to 64 bytes is kept verbatim.
func TestMiddleware_RequestIDRejectsUnsafeHeader(t *testing.T) {
	var got string
	h := testApp().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestID(r.Context())
	}))
	serve := func(header string) string {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-Id", header)
		h.ServeHTTP(httptest.NewRecorder(), req)
		return got
	}
	hex16 := regexp.MustCompile(`^[0-9a-f]{16}$`)
	for _, bad := range []string{
		"abc\nappshell: panic serving GET /admin (request forged): x",
		"has space",
		`quote"d`,
		"tab\there",
		"ünïcode",
		strings.Repeat("a", 65),
	} {
		if id := serve(bad); id == bad || !hex16.MatchString(id) {
			t.Errorf("header %q: id = %q, want a generated 16-hex id", bad, id)
		}
	}
	for _, ok := range []string{"from-proxy", "A.b_c-9", strings.Repeat("z", 64), "550e8400-e29b-41d4-a716-446655440000"} {
		if id := serve(ok); id != ok {
			t.Errorf("header %q: id = %q, want it kept", ok, id)
		}
	}
}
