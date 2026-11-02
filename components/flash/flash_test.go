package flash

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/banner"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	if c == nil {
		t.Fatal("render: nil component")
	}
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

// setCookie runs set against a fresh recorder for r and returns the one
// wf_flash cookie it wrote.
func setCookie(t *testing.T, r *http.Request, set func(w http.ResponseWriter)) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	set(rec)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != CookieName {
		t.Fatalf("want exactly one %s cookie, got %v", CookieName, cookies)
	}
	return cookies[0]
}

// take runs Take on a request carrying value as the flash cookie and returns
// the component and the recorder that saw any clearing Set-Cookie.
func take(t *testing.T, value string) (templ.Component, *httptest.ResponseRecorder) {
	t.Helper()
	r := httptest.NewRequest("GET", "/movies", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: value})
	rec := httptest.NewRecorder()
	return Take(rec, r), rec
}

// clearing returns the wf_flash cookie rec wrote and fails unless it expires
// the cookie on Path=/.
func clearing(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != CookieName {
		t.Fatalf("want exactly one clearing %s cookie, got %v", CookieName, cookies)
	}
	c := cookies[0]
	if c.MaxAge >= 0 || c.Path != "/" {
		t.Errorf("clearing cookie must have MaxAge<0 and Path=/, got %q", c.Raw)
	}
	return c
}

func TestSetThenTake_RendersOnceAndClears(t *testing.T) {
	r := httptest.NewRequest("POST", "/movies/7/delete", nil)
	c := setCookie(t, r, func(w http.ResponseWriter) {
		Error(w, r, "Could not delete", "The movie is still in two lists.")
	})

	got, rec := take(t, c.Value)
	out := render(t, got)
	mustContain(t, out,
		`<div class="mb-6">`,
		`role="alert"`,
		"Could not delete",
		"The movie is still in two lists.",
		`aria-label="Dismiss"`,
	)
	clearing(t, rec)

	// The browser drops the cleared cookie, so the next request has none.
	rec2 := httptest.NewRecorder()
	if again := Take(rec2, httptest.NewRequest("GET", "/movies", nil)); again != nil {
		t.Errorf("second Take returned a component, want nil")
	}
	if sc := rec2.Result().Header.Values("Set-Cookie"); len(sc) != 0 {
		t.Errorf("Take with no cookie wrote Set-Cookie %v, want none", sc)
	}
}

func TestSet_CookieAttributes(t *testing.T) {
	r := httptest.NewRequest("POST", "/x", nil)
	c := setCookie(t, r, func(w http.ResponseWriter) { Info(w, r, "Hi", "") })
	if !c.HttpOnly {
		t.Error("want HttpOnly")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /", c.Path)
	}
	if c.MaxAge != 300 {
		t.Errorf("MaxAge = %d, want 300", c.MaxAge)
	}
	if c.Secure {
		t.Error("plain http request must not get a Secure cookie")
	}
	if strings.ContainsAny(c.Value, "+/=") {
		t.Errorf("value %q is not unpadded base64url", c.Value)
	}
}

func TestSet_SecureFollowsTLS(t *testing.T) {
	cases := map[string]struct {
		prep func(*http.Request)
		want bool
	}{
		"plain":            {func(*http.Request) {}, false},
		"tls":              {func(r *http.Request) { r.TLS = &tls.ConnectionState{} }, true},
		"forwarded https":  {func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") }, true},
		"forwarded HTTPS":  {func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "HTTPS") }, true},
		"forwarded http":   {func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "http") }, false},
		"forwarded random": {func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "wss") }, false},
	}
	for name, tc := range cases {
		r := httptest.NewRequest("POST", "/x", nil)
		tc.prep(r)
		c := setCookie(t, r, func(w http.ResponseWriter) { Warning(w, r, "Hi", "") })
		if c.Secure != tc.want {
			t.Errorf("%s: Secure = %v, want %v", name, c.Secure, tc.want)
		}
		// The clearing cookie follows the same rule.
		r2 := httptest.NewRequest("GET", "/y", nil)
		tc.prep(r2)
		r2.AddCookie(&http.Cookie{Name: CookieName, Value: c.Value})
		rec := httptest.NewRecorder()
		Take(rec, r2)
		if cl := clearing(t, rec); cl.Secure != tc.want {
			t.Errorf("%s: clearing Secure = %v, want %v", name, cl.Secure, tc.want)
		}
	}
}

func TestSet_OversizeBodyTruncated(t *testing.T) {
	r := httptest.NewRequest("POST", "/x", nil)
	body := strings.Repeat("é ", 3000) // multi-byte runes, 9000 bytes
	c := setCookie(t, r, func(w http.ResponseWriter) { Error(w, r, "Import failed", body) })
	if len(c.Value) >= 2048 {
		t.Errorf("cookie value is %d bytes, want under 2 KB", len(c.Value))
	}
	m, ok := decode(c.Value)
	if !ok {
		t.Fatalf("truncated value does not decode: %q", c.Value)
	}
	if m.Kind != banner.VariantError || m.Title != "Import failed" {
		t.Errorf("kind/title damaged by truncation: %+v", m)
	}
	if !strings.HasSuffix(m.Body, ellipsis) || len(m.Body) < 1000 {
		t.Errorf("body not truncated with ellipsis, or cut far more than needed (len %d): %q…", len(m.Body), m.Body[:40])
	}
	if !strings.HasPrefix(body, strings.TrimSuffix(m.Body, ellipsis)) {
		t.Error("truncated body is not a prefix of the original (rune boundary broken?)")
	}
	// A title alone that is too long is cut too, never an oversize cookie.
	c = setCookie(t, r, func(w http.ResponseWriter) { Error(w, r, strings.Repeat("x", 5000), "") })
	if len(c.Value) >= 2048 {
		t.Errorf("title-only cookie value is %d bytes, want under 2 KB", len(c.Value))
	}
	if m, ok := decode(c.Value); !ok || !strings.HasSuffix(m.Title, ellipsis) {
		t.Errorf("oversize title not truncated: ok=%v %+v", ok, m)
	}
}

func TestSet_ShortBodyNotTruncated(t *testing.T) {
	r := httptest.NewRequest("POST", "/x", nil)
	c := setCookie(t, r, func(w http.ResponseWriter) { Success(w, r, "Saved", "All 12 rows imported.") })
	m, ok := decode(c.Value)
	if !ok || m != (Message{Kind: banner.VariantSuccess, Title: "Saved", Body: "All 12 rows imported."}) {
		t.Errorf("round trip changed the message: ok=%v %+v", ok, m)
	}
}

func TestTake_GarbageIsNilAndCleared(t *testing.T) {
	b64 := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	cases := map[string]string{
		"not base64":        "%%%not base64%%%",
		"base64 not json":   b64("hello there"),
		"json array":        b64(`[1,2,3]`),
		"json string":       b64(`"just a string"`),
		"wrong field types": b64(`{"k":5,"t":["a"],"b":{}}`),
		"truncated json":    b64(`{"k":"error","t":"Oops`),
		"empty":             "",
		"only padding":      "====",
		"deep nesting":      b64(strings.Repeat("[", 100000)),
		"no text":           b64(`{"k":"error"}`),
		"empty object":      b64(`{}`),
	}
	for name, value := range cases {
		got, rec := take(t, value)
		if got != nil {
			t.Errorf("%s: Take returned a component, want nil", name)
		}
		clearing(t, rec)
	}
}

func TestTake_UnknownKindFallsBackToInfo(t *testing.T) {
	value := base64.RawURLEncoding.EncodeToString([]byte(`{"k":"<script>","t":"Hi","b":"there"}`))
	got, rec := take(t, value)
	out := render(t, got)
	mustContain(t, out, `role="status"`, "border-info", "Hi", "there")
	if strings.Contains(out, "<script>") {
		t.Errorf("tampered kind leaked into the markup:\n%s", out)
	}
	clearing(t, rec)
}

func TestTake_AcceptsPaddedValue(t *testing.T) {
	value := base64.URLEncoding.EncodeToString([]byte(`{"k":"warning","t":"Careful"}`))
	if !strings.HasSuffix(value, "=") {
		t.Fatalf("test precondition: padded value, got %q", value)
	}
	got, _ := take(t, value)
	mustContain(t, render(t, got), "border-warning", "Careful")
}

func TestFlash_RolePerKind(t *testing.T) {
	cases := map[string]struct {
		kind banner.Variant
		role string
		tone string
	}{
		"error":   {banner.VariantError, `role="alert"`, "border-destructive"},
		"warning": {banner.VariantWarning, `role="status"`, "border-warning"},
		"info":    {banner.VariantInfo, `role="status"`, "border-info"},
		"success": {banner.VariantSuccess, `role="status"`, "border-success"},
	}
	for name, tc := range cases {
		out := render(t, Flash(Message{Kind: tc.kind, Title: "T", Body: "B"}))
		mustContain(t, out, tc.role, tc.tone, `x-data="{ open: true }"`, `aria-label="Dismiss"`)
		if !strings.HasPrefix(out, `<div class="mb-6">`) {
			t.Errorf("%s: wrapper missing or carries extra classes:\n%s", name, out)
		}
		for _, bad := range []string{"-mx-", "-ml-", "-mr-", "w-screen", "w-["} {
			if strings.Contains(out, bad) {
				t.Errorf("%s: %q would break the 390px gutter:\n%s", name, bad, out)
			}
		}
	}
}

func TestFlash_EscapesText(t *testing.T) {
	out := render(t, Flash(Message{Kind: banner.VariantError, Title: "<b>x</b>", Body: "a & b"}))
	mustContain(t, out, "&lt;b&gt;x&lt;/b&gt;", "a &amp; b")
	if strings.Contains(out, "<b>x</b>") {
		t.Errorf("title not escaped:\n%s", out)
	}
}

func TestShorthands_SetKind(t *testing.T) {
	r := httptest.NewRequest("POST", "/x", nil)
	for want, fn := range map[banner.Variant]func(http.ResponseWriter, *http.Request, string, string){
		banner.VariantError:   Error,
		banner.VariantWarning: Warning,
		banner.VariantInfo:    Info,
		banner.VariantSuccess: Success,
	} {
		c := setCookie(t, r, func(w http.ResponseWriter) { fn(w, r, "T", "B") })
		m, ok := decode(c.Value)
		if !ok || m.Kind != want {
			t.Errorf("shorthand for %q stored kind %q (ok=%v)", want, m.Kind, ok)
		}
	}
}

// TestTake_IsAppFlashShape pins the wiring: App.Flash = flash.Take compiles.
func TestTake_IsAppFlashShape(t *testing.T) {
	var f func(http.ResponseWriter, *http.Request) templ.Component = Take
	if f == nil {
		t.Fatal("unreachable")
	}
}
