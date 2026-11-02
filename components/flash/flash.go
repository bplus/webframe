package flash

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/banner"
)

// CookieName is the cookie that carries the pending message between the
// response that set it and the next full page.
const CookieName = "wf_flash"

const (
	// maxValueBytes is the hard ceiling on the encoded cookie value: the
	// whole value stays under 2 KB, well inside every browser's 4 KB
	// per-cookie limit and leaving room for the attributes.
	maxValueBytes = 2047
	// maxAgeSeconds is how long a flash waits to be shown. A redirect is
	// followed within a second; five minutes covers a slow network or a
	// user who stops on an interstitial without leaving a stale message
	// for the next sign-in.
	maxAgeSeconds = 300
	// ellipsis marks a Body that was cut to fit the cookie.
	ellipsis = "…"
)

// Message is one flash: its banner kind, a bold title and a plain body.
// Either text may be empty; a Message with both empty renders nothing.
type Message struct {
	Kind        banner.Variant
	Title, Body string
}

// wire is the JSON shape stored in the cookie. Short keys keep the value
// small; the struct is separate from Message so the cookie format is not
// tied to the exported field names.
type wire struct {
	Kind  banner.Variant `json:"k,omitempty"`
	Title string         `json:"t,omitempty"`
	Body  string         `json:"b,omitempty"`
}

// Set stores m in the flash cookie so the next page rendered for this
// browser shows it once. Call it before the first body write. The cookie is
// Secure when r arrived over TLS or with X-Forwarded-Proto: https. Text too
// long for the 2 KB cap is truncated (Body first, then Title) and ends in "…".
func Set(w http.ResponseWriter, r *http.Request, m Message) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    encode(m),
		Path:     "/",
		MaxAge:   maxAgeSeconds,
		HttpOnly: true,
		Secure:   secure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// Error sets an error flash: the common case, shown with role="alert".
func Error(w http.ResponseWriter, r *http.Request, title, body string) {
	Set(w, r, Message{Kind: banner.VariantError, Title: title, Body: body})
}

// Warning sets a warning flash: something needs attention but did not fail.
func Warning(w http.ResponseWriter, r *http.Request, title, body string) {
	Set(w, r, Message{Kind: banner.VariantWarning, Title: title, Body: body})
}

// Info sets a neutral flash.
func Info(w http.ResponseWriter, r *http.Request, title, body string) {
	Set(w, r, Message{Kind: banner.VariantInfo, Title: title, Body: body})
}

// Success sets a success flash, the ordinary "Saved" shown once on the
// page a POST redirects to.
func Success(w http.ResponseWriter, r *http.Request, title, body string) {
	Set(w, r, Message{Kind: banner.VariantSuccess, Title: title, Body: body})
}

// Take consumes the flash cookie on r and returns Flash(m) for its message,
// or nil when there is no cookie, the value does not decode, or the message
// has no text. Whenever the cookie is present Take clears it, decoded or
// not; with no cookie it writes no header. It matches appshell.App.Flash.
func Take(w http.ResponseWriter, r *http.Request) templ.Component {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return nil
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure(r),
		SameSite: http.SameSiteLaxMode,
	})
	m, ok := decode(c.Value)
	if !ok || (m.Title == "" && m.Body == "") {
		return nil
	}
	return Flash(m)
}

// secure reports whether cookies for r should carry the Secure attribute:
// the request itself was TLS, or a reverse proxy terminated TLS and said so.
func secure(r *http.Request) bool {
	if r == nil {
		return false
	}
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// encode returns the cookie value for m, truncating Body (then Title) until
// the base64url(JSON) value is under the cap. Marshalling a struct of
// strings cannot fail, so the error is ignored.
func encode(m Message) string {
	v := wire{Kind: m.Kind, Title: m.Title, Body: m.Body}
	for {
		b, _ := json.Marshal(v)
		s := base64.RawURLEncoding.EncodeToString(b)
		over := len(s) - maxValueBytes
		if over <= 0 {
			return s
		}
		// Each base64 character carries 3/4 of a byte; cut at least that
		// much, plus the ellipsis that replaces it, from the body first.
		cut := over*3/4 + 1 + len(ellipsis)
		switch {
		case v.Body != "":
			v.Body = truncate(v.Body, cut)
		case v.Title != "":
			v.Title = truncate(v.Title, cut)
		default:
			// Only the kind is left and it still does not fit: impossible
			// with the known variants, but never loop on a hostile one.
			v.Kind = ""
			b, _ = json.Marshal(v)
			return base64.RawURLEncoding.EncodeToString(b)
		}
	}
}

// truncate removes at least n bytes from the end of s, backs up to a rune
// boundary (JSON escaping can only grow the value, never shrink it, so
// cutting whole runes is safe), and appends the ellipsis. It returns "" when
// nothing would be left but the ellipsis.
func truncate(s string, n int) string {
	if s == ellipsis {
		return ""
	}
	s = strings.TrimSuffix(s, ellipsis)
	end := len(s) - n
	if end <= 0 {
		return ""
	}
	for end > 0 && !utf8.RuneStart(s[end]) {
		end--
	}
	if end == 0 {
		return ""
	}
	return s[:end] + ellipsis
}

// decode parses a cookie value written by encode. It reports false for
// anything that is not base64url of a JSON object; unknown kinds fall back
// to the neutral banner so a tampered kind cannot pick an unstyled variant.
func decode(value string) (Message, bool) {
	// Tolerate padded values from a hand-written cookie.
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(value, "="))
	if err != nil {
		return Message{}, false
	}
	var v wire
	if err := json.Unmarshal(raw, &v); err != nil {
		return Message{}, false
	}
	switch v.Kind {
	case banner.VariantInfo, banner.VariantSuccess, banner.VariantWarning, banner.VariantError:
	default:
		v.Kind = banner.VariantInfo
	}
	return Message{Kind: v.Kind, Title: v.Title, Body: v.Body}, true
}
