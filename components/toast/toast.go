package toast

import (
	"net/http"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/appshell"
)

// oobAttr is the htmx attribute that routes a response fragment to the
// Document's toast container instead of the request's target.
const (
	oobAttr  = "hx-swap-oob"
	oobValue = "beforeend:#toast-container"
)

// withOOB returns a copy of p whose Attributes carry the out-of-band swap.
// The caller's map is not modified.
func withOOB(p Props) Props {
	attrs := make(templ.Attributes, len(p.Attributes)+1)
	for k, v := range p.Attributes {
		attrs[k] = v
	}
	attrs[oobAttr] = oobValue
	p.Attributes = attrs
	return p
}

// Send appends the toast to the response for an htmx request: it renders
// OOB(p) to w, which htmx swaps into #toast-container out of band. For a
// request that is not from htmx (appshell.IsHX is false) it writes nothing
// and returns nil: such a handler ends in a redirect or a full page, where
// a toast fragment has nowhere to land; it should set a one-shot flash
// (package flash) instead.
//
// Send never sets the status code or Content-Type: the handler's main
// response does, and Send is called after that response's body (or
// headers) has been written. Called first, with nothing written yet, it
// makes Go send a 200 with its own sniffed Content-Type, so a handler
// whose only output is the toast sets Content-Type itself before calling
// it. A render error is returned, not written.
func Send(w http.ResponseWriter, r *http.Request, p Props) error {
	if !appshell.IsHX(r) {
		return nil
	}
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

// Success builds the Props for a success toast. Opt-in: pages do not show
// success messages unless the user asked for one.
func Success(title, body string) Props {
	return Props{Kind: KindSuccess, Title: title, Body: body}
}
