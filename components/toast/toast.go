package toast

import (
	"io"
	"net/http"
	"strings"

	"github.com/bplus/webframe/components/appshell"
)

// Send appends OOB(p) to the response for an htmx request. For a non-htmx
// request it writes nothing and returns nil; use a flash (package flash)
// there instead.
//
// Send never sets the status code and is normally called after the main
// response body. It sets Content-Type only when none is set yet. For a
// toast-only error response use Respond.
func Send(w http.ResponseWriter, r *http.Request, p Props) error {
	if !appshell.IsHX(r) {
		return nil
	}
	if h := w.Header(); h.Get("Content-Type") == "" {
		h.Set("Content-Type", "text/html; charset=utf-8")
	}
	return OOB(p).Render(r.Context(), w)
}

// Respond makes the toast the whole response, leaving the page unchanged
// except for the toast. It sets HX-Reswap: none, so htmx does not replace
// the request's target even on an error status, writes status (0 means
// 200) and renders OOB(p). For a non-htmx request it writes the title and
// body as plain text.
func Respond(w http.ResponseWriter, r *http.Request, status int, p Props) error {
	if status == 0 {
		status = http.StatusOK
	}
	if !appshell.IsHX(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, err := io.WriteString(w, strings.TrimSpace(p.Title+"\n"+p.Body)+"\n")
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Reswap", "none")
	w.WriteHeader(status)
	return OOB(p).Render(r.Context(), w)
}

// Error builds the Props for an error toast (role="alert").
func Error(title, body string) Props {
	return Props{Kind: KindError, Title: title, Body: body}
}

// Warning builds the Props for a warning toast.
func Warning(title, body string) Props {
	return Props{Kind: KindWarning, Title: title, Body: body}
}

// Info builds the Props for a neutral toast.
func Info(title, body string) Props {
	return Props{Kind: KindInfo, Title: title, Body: body}
}

// Success builds the Props for a success toast.
func Success(title, body string) Props {
	return Props{Kind: KindSuccess, Title: title, Body: body}
}
