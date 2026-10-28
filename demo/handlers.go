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

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/badge"
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

// HandleTraffic renders the live traffic stream demo page (GET /traffic). The
// page itself then opens an SSE connection to HandleTrafficStream for its
// live data.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleTraffic(w http.ResponseWriter, r *http.Request) {
	if err := App.Render(w, r, "Traffic", pages.Traffic()); err != nil {
		log.Printf("demo: render traffic: %v", err)
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

// HandleSearch returns filtered results (demonstrates hx-get + debounce).
// Filters a hard-coded framework-name list by case-insensitive substring match
// on the "q" query param. An empty q returns the full list; zero matches
// render a "No results found" line.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	items := []string{"Alpine.js", "Angular", "Astro", "Django", "Express", "FastAPI", "Flask", "Go Chi", "HTMX", "Laravel", "Next.js", "Nuxt", "Rails", "React", "Remix", "Solid", "Svelte", "Templ", "Vue"}
	var found []string
	for _, item := range items {
		if q == "" || strings.Contains(strings.ToLower(item), q) {
			found = append(found, item)
		}
	}
	fragment(w, r, "search", pages.SearchResults(found))
}

// HandleValidate returns field validation feedback (demonstrates hx-post on
// blur). Only two fields have rules — "email" (non-empty, contains '@') and
// "name" (at least 2 bytes — byte length, not runes, so a single multibyte
// character like "é" passes) — and any other "field" value falls through
// with no error, rendering the success message.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleValidate(w http.ResponseWriter, r *http.Request) {
	field := r.FormValue("field")
	value := r.FormValue("value")

	var errMsg string
	switch field {
	case "email":
		if value == "" {
			errMsg = "Email is required"
		} else if !strings.Contains(value, "@") {
			errMsg = "Enter a valid email address"
		}
	case "name":
		if value == "" {
			errMsg = "Name is required"
		} else if len(value) < 2 {
			errMsg = "Name must be at least 2 characters"
		}
	}
	fragment(w, r, "validate", pages.ValidationFeedback(errMsg))
}

// toastKinds maps a toast variant to its badge variant, label and message.
var toastKinds = map[string]struct {
	variant    badge.Variant
	label, msg string
}{
	"success": {badge.VariantDefault, "Success", "Operation completed successfully!"},
	"error":   {badge.VariantDestructive, "Error", "Something went wrong. Please try again."},
	"info":    {badge.VariantSecondary, "Info", "Here's some useful information."},
	"warning": {badge.VariantOutline, "Warning", "Careful — this action cannot be undone."},
}

// HandleToast returns a toast notification (demonstrates hx-swap-oob). The
// "variant" form value picks the message and badge; empty or unknown
// defaults to "success". The response is appended out-of-band to
// #toast-container and removes itself after 4 seconds (see pages.Toast).
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleToast(w http.ResponseWriter, r *http.Request) {
	k, ok := toastKinds[r.FormValue("variant")]
	if !ok {
		k = toastKinds["success"]
	}
	fragment(w, r, "toast", pages.Toast(k.variant, k.label, k.msg))
}

// fragment writes an htmx response fragment, logging a render error (the
// status line is already sent by then).
func fragment(w http.ResponseWriter, r *http.Request, name string, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("demo: render %s: %v", name, err)
	}
}
