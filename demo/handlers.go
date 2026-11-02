// Package demo contains HTTP handlers for the webframe demo application only.
// It is not part of webframe's library API; consuming applications must not
// import it.
package demo

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/autocomplete"
	"github.com/bplus/webframe/components/flash"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/pageheader"
	"github.com/bplus/webframe/components/streampage"
	"github.com/bplus/webframe/components/toast"
	"github.com/bplus/webframe/components/trafficstream"
	"github.com/bplus/webframe/demo/pages"
)

// HandleShowcase renders the component showcase gallery page — the demo
// server's home page (GET /). All content comes from the compiled
// pages.Showcase templ component; the request carries no parameters.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleShowcase(w http.ResponseWriter, r *http.Request) {
	if err := App.Render(w, r, "Showcase", pages.Showcase()); err != nil {
		log.Printf("demo: render showcase: %v", err)
	}
}

// HandleTraffic renders the live traffic stream demo page (GET /traffic)
// through streampage: the page header, the trafficstream panel pointed at
// HandleTrafficStream and the panel's script, declared in trafficProps. The
// page itself then opens an SSE connection for its live data.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleTraffic(w http.ResponseWriter, r *http.Request) {
	if err := streampage.Respond(w, r, "Traffic", trafficProps()); err != nil {
		log.Printf("demo: render traffic: %v", err)
	}
}

// trafficProps declares the traffic page: the same stream the hand-written
// page carried before batch 4 (StreamURL /api/traffic/stream, a 560px
// table, the demo provider and event options, two seed rows sharing a
// trace id), now as streampage.Props.
func trafficProps() streampage.Props {
	return streampage.Props{
		ID: "traffic",
		Header: pageheader.Props{
			Title:    "Traffic Stream",
			Subtitle: "A live operator surface for watching request and response activity, filtering by provider and event type, and pausing the feed without losing context.",
		},
		Stream: trafficstream.Props{
			StreamURL:       "/api/traffic/stream",
			Height:          560,
			ProviderOptions: []string{"anthropic", "openai", "gemini", "local"},
			EventOptions:    []string{"request.received", "proxy.forward_start", "proxy.forward_end", "proxy.error", "local.command"},
			InitialRows: []trafficstream.Row{{
				Time:      "12:00:01",
				Provider:  "anthropic",
				Direction: "request",
				Event:     "request.received",
				Status:    "",
				TraceID:   "trace-demo-001",
				Summary:   "POST /v1/messages",
			}, {
				Time:      "12:00:02",
				Provider:  "anthropic",
				Direction: "response",
				Event:     "proxy.forward_end",
				Status:    "200",
				TraceID:   "trace-demo-001",
				Summary:   "Completed upstream round-trip",
			}},
		},
	}
}

// HandleTrafficStream emits demo SSE traffic events. Every 1.2s it writes one
// synthetic JSON event — provider/type/direction/status cycled from fixed
// lists by loop index, so the stream is deterministic, not random — then
// flushes. It blocks until the client disconnects (r.Context() canceled);
// there is no other exit. Responds 500 if the ResponseWriter does not
// implement http.Flusher, since SSE is impossible without flushing.
//
// @boundary HTTP - Server-Sent Events stream
// @side-effects streams to the response indefinitely; holds the connection
// open and a running time.Ticker until client disconnect
func HandleTrafficStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	providers := []string{"anthropic", "openai", "gemini", "local"}
	events := []string{"request.received", "proxy.forward_start", "proxy.forward_end", "proxy.error", "local.command"}
	directions := []string{"request", "response", "event", "local"}
	statuses := []string{"", "200", "202", "500"}
	// 1.2s: slow enough to read each event as it appears, fast enough that
	// the stream still feels live rather than static.
	ticker := time.NewTicker(1200 * time.Millisecond)
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-r.Context().Done():
			return
		case ts := <-ticker.C:
			provider := providers[i%len(providers)]
			eventType := events[i%len(events)]
			direction := directions[i%len(directions)]
			status := statuses[i%len(statuses)]
			fmt.Fprintf(w, "data: {\"ts\":\"%s\",\"provider\":\"%s\",\"direction\":\"%s\",\"type\":\"%s\",\"status\":\"%s\",\"trace_id\":\"trace-demo-%03d\",\"summary\":\"%s %s\"}\n\n", ts.UTC().Format(time.RFC3339Nano), provider, direction, eventType, status, i+1, strings.ToUpper(direction), provider)
			flusher.Flush()
		}
	}
}

// HandleClick returns an updated button label (demonstrates hx-post + hx-swap).
// Stateless by design: the count round-trips through the hx-vals attribute of
// the replacement button this handler emits, so the server keeps no counter.
// A missing "count" form value yields 1 (first click); a present value is
// parsed and incremented. An unparseable value leaves n at its default and
// still increments, so garbage input resets the counter to 2 rather than
// erroring.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleClick(w http.ResponseWriter, r *http.Request) {
	count := r.FormValue("count")
	n := 1
	if count != "" {
		fmt.Sscanf(count, "%d", &n)
		n++
	}
	fragment(w, r, "click", pages.ClickButton(n))
}

// HandleSearch answers the autocomplete demo (GET /api/search?q=...): it
// filters a hard-coded framework-name list by case-insensitive substring
// match on q and renders autocomplete.Results, one option per match with
// Label = the name and Value = slug(name). An empty q returns the full list;
// zero matches render the component's "No matches" row.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	names := []string{"Alpine.js", "Angular", "Astro", "Django", "Express", "FastAPI", "Flask", "Go Chi", "HTMX", "Laravel", "Next.js", "Nuxt", "Rails", "React", "Remix", "Solid", "Svelte", "Templ", "Vue"}
	var found []autocomplete.Item
	for _, name := range names {
		if q == "" || strings.Contains(strings.ToLower(name), q) {
			found = append(found, autocomplete.Item{Label: name, Value: slug(name)})
		}
	}
	fragment(w, r, "search", autocomplete.Results(found))
}

// slug lowercases s, turns every run of non-alphanumeric characters into a
// single "-", and trims leading and trailing "-" ("Next.js" -> "next-js").
//
// @pure
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, c := range strings.ToLower(s) {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			b.WriteRune(c)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// HandlePicked answers the autocomplete form's submit (POST /api/picked):
// it echoes the hidden "framework" value and the visible "q" text the form
// sent, so the demo shows which of the two a real form would receive.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandlePicked(w http.ResponseWriter, r *http.Request) {
	fragment(w, r, "picked", pages.PickedResult(r.FormValue("framework"), r.FormValue("q")))
}

// HandleValidate answers the Validate hx-post of the showcase's two
// blur-validated fields (pages.ValidatedEmail, pages.ValidatedName). htmx
// sends only the control's own name=value, so the field is recognised by
// which key is present: "email" must be non-empty and contain '@', "name"
// must be at least 2 bytes (byte length, not runes, so a single multibyte
// character like "é" passes). The reply is formfield.RespondFeedback for
// that Field: the error message, or the empty slot (which takes no space)
// when the value is valid, always with status 200 so htmx swaps it. A
// request naming neither field is a 400.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleValidate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	var field formfield.Field
	var errMsg string
	switch {
	case r.Form.Has(pages.ValidatedEmail.Name):
		field = pages.ValidatedEmail
		value := r.Form.Get(field.Name)
		if value == "" {
			errMsg = "Email is required"
		} else if !strings.Contains(value, "@") {
			errMsg = "Enter a valid email address"
		}
	case r.Form.Has(pages.ValidatedName.Name):
		field = pages.ValidatedName
		value := r.Form.Get(field.Name)
		if value == "" {
			errMsg = "Name is required"
		} else if len(value) < 2 {
			errMsg = "Name must be at least 2 characters"
		}
	default:
		http.Error(w, "unknown field", http.StatusBadRequest)
		return
	}
	if err := formfield.RespondFeedback(w, r, field, errMsg); err != nil {
		log.Printf("demo: render validate: %v", err)
	}
}

// toastKinds maps the showcase's "variant" form value to the toast it
// sends. The texts are written so the same reply reads right after any of
// the showcase actions that post here (the Toast buttons, the confirm
// dialog's actions, the toolbar and the table rows): "warning" answers a
// confirmed destructive action and "info" a confirmed plain one. "sticky"
// is an error toast that stays until dismissed.
var toastKinds = map[string]toast.Props{
	"success": toast.Success("Export finished", "The file is ready to download."),
	"error":   toast.Error("Could not save", "The server refused the change. Try again."),
	"info":    toast.Info("Showcase action", "The server received the request and changed nothing."),
	"warning": toast.Warning("Nothing was deleted", "The showcase answers destructive actions with this toast instead of acting."),
	"sticky":  {Kind: toast.KindError, Sticky: true, Title: "Build failed", Body: "This one stays until you dismiss it."},
}

// HandleToast answers the showcase's toast buttons (POST /api/toast): the
// "variant" form value picks one of toastKinds (empty or unknown → "info")
// and toast.Send appends it out of band to #toast-container. Send writes
// nothing for a request that is not from htmx, so a plain POST gets an
// empty 200: a toast has nowhere to land on a full page, and a handler
// that redirects sets a flash instead (see HandleFlash).
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleToast(w http.ResponseWriter, r *http.Request) {
	p, ok := toastKinds[r.FormValue("variant")]
	if !ok {
		p = toastKinds["info"]
	}
	if err := toast.Send(w, r, p); err != nil {
		log.Printf("demo: render toast: %v", err)
	}
}

// HandleFlash answers the showcase's flash form (POST /api/flash): it
// stores a one-shot warning in the flash cookie and redirects (303) to the
// top of the showcase, where the shell renders the banner once in its
// #wf-flash slot above the page title; a reload shows nothing.
//
// @boundary HTTP
// @side-effects sets the wf_flash cookie; writes a redirect
func HandleFlash(w http.ResponseWriter, r *http.Request) {
	flash.Warning(w, r, "This page was reached through a redirect", "The warning rode along in a one-shot cookie; reload and it is gone.")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// slowDelay is how long HandleSlow waits before answering: long enough to
// see the htmx indicator, short enough not to feel broken. Tests set it to
// zero.
var slowDelay = 1500 * time.Millisecond

// HandleSlow answers the showcase's spinner demo (GET /api/slow): it waits
// slowDelay (or until the client goes away) and then writes a one-line
// fragment with the time it answered, so the request indicator has
// something to wait for.
//
// @boundary HTTP
// @side-effects blocks for slowDelay; writes HTML response
func HandleSlow(w http.ResponseWriter, r *http.Request) {
	select {
	case <-r.Context().Done():
		return
	case <-time.After(slowDelay):
	}
	fragment(w, r, "slow", pages.SlowResult(time.Now().Format("15:04:05")))
}

// HandleBoom is the showcase's deliberate plain-text 500 (GET /api/boom):
// http.Error writes "text/plain" and no markup, exactly what a Go handler
// does when it fails without an error page. htmx-errors.js refuses to swap
// a non-HTML error body, so the button's target on the showcase stays as it
// was; without the guard the words would land inside the page.
//
// @boundary HTTP
// @side-effects writes a plain-text 500
func HandleBoom(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "boom: the demo's deliberate plain-text 500", http.StatusInternalServerError)
}

// HandlePanic is the showcase's deliberate panic (GET /api/panic): the
// handler fails the way a nil dereference or an unchecked index would.
// appshell.Middleware recovers it, logs the value and the stack with the
// request id, and calls App.OnPanic, which the demo sets to
// errorpage.PanicHandler, so the answer is the styled 500 page in the
// shell with that request id to quote.
//
// @boundary HTTP
// @side-effects panics; the shell's middleware writes the 500
func HandlePanic(w http.ResponseWriter, r *http.Request) {
	panic("demo: the deliberate panic behind /api/panic")
}

// fragment writes an htmx response fragment, logging a render error (the
// status line is already sent by then).
func fragment(w http.ResponseWriter, r *http.Request, name string, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("demo: render %s: %v", name, err)
	}
}
