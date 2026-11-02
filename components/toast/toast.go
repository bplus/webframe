package toast

import (
	"io"
	"net/http"
	"strings"

	"github.com/bplus/webframe/components/appshell"
)

// Send appends the toast to the response for an htmx request: it renders
// OOB(p) to w, which htmx swaps into #toast-container out of band. For a
// request that is not from htmx (appshell.IsHX is false) it writes nothing
// and returns nil: such a handler ends in a redirect or a full page, where
// a toast fragment has nowhere to land; it should set a one-shot flash
// (package flash) instead.
//
// Send never sets the status code: the handler's main response does, and
// Send is normally called after that response's body. It sets Content-Type
// to HTML only when nothing has set one yet (a handler whose only output
// is the toast; for an error status use Respond, which also tells htmx to
// swap nothing). A render error is returned, not written.
func Send(w http.ResponseWriter, r *http.Request, p Props) error {
	if !appshell.IsHX(r) {
		return nil
	}
	if h := w.Header(); h.Get("Content-Type") == "" {
		h.Set("Content-Type", "text/html; charset=utf-8")
	}
	return OOB(p).Render(r.Context(), w)
}

// Respond makes the toast the whole response: the action failed (or had
// nothing to show) and nothing on the page changes except the toast. It
// sets HX-Reswap: none, so htmx leaves the request's target untouched
// (appshell.Document configures htmx to swap error statuses too, and
// without this header a toast-only 409 would replace the target with an
// empty body), sets Content-Type, writes status (0 → 200) and renders
// OOB(p). For a request that is not from htmx it writes status and the
// title and body as plain text; such a handler normally prefers a flash
// and a redirect.
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
