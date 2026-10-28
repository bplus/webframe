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
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<button hx-post="/api/click" hx-swap="outerHTML" hx-vals='{"count":"%d"}' class="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors">Clicked %d times</button>`, n, n)
}

// HandleSearch returns filtered results (demonstrates hx-get + debounce).
// Filters a hard-coded framework-name list by case-insensitive substring match
// on the "q" query param. An empty q returns the full list; zero matches
// return a "No results found" placeholder item instead of an empty <ul>.
//
// @boundary HTTP
// @side-effects writes HTML response
// @ai-context the query value is never echoed into the response — only the
// static list items are — which is why the raw Fprintf here is XSS-safe; do
// not copy this pattern for user-supplied output
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	items := []string{"Alpine.js", "Angular", "Astro", "Django", "Express", "FastAPI", "Flask", "Go Chi", "HTMX", "Laravel", "Next.js", "Nuxt", "Rails", "React", "Remix", "Solid", "Svelte", "Templ", "Vue"}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<ul class="space-y-1">`)
	found := 0
	for _, item := range items {
		if q == "" || strings.Contains(strings.ToLower(item), q) {
			fmt.Fprintf(w, `<li class="px-3 py-2 rounded-md text-sm hover:bg-accent transition-colors cursor-default">%s</li>`, item)
			found++
		}
	}
	if found == 0 {
		fmt.Fprint(w, `<li class="px-3 py-2 text-sm text-muted-foreground">No results found</li>`)
	}
	fmt.Fprint(w, `</ul>`)
}

// HandleValidate returns field validation feedback (demonstrates hx-post on
// blur). Only two fields have rules — "email" (non-empty, contains '@') and
// "name" (at least 2 bytes — byte length, not runes, so a single multibyte
// character like "é" passes) — and any other "field" value falls through
// with no error, rendering the success message. The submitted value is never
// echoed back, so no output escaping is needed.
//
// @boundary HTTP
// @side-effects writes HTML response
func HandleValidate(w http.ResponseWriter, r *http.Request) {
	field := r.FormValue("field")
	value := r.FormValue("value")

	w.Header().Set("Content-Type", "text/html")

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

	if errMsg != "" {
		fmt.Fprintf(w, `<p class="text-sm text-destructive mt-1">%s</p>`, errMsg)
	} else {
		fmt.Fprintf(w, `<p class="text-sm text-emerald-500 mt-1">Looks good!</p>`)
	}
}

// HandleToast returns a toast notification (demonstrates hx-swap-oob). The
// "variant" form value picks the message and accent color; empty defaults to
// "success". The response swaps out-of-band into #toast-container and relies
// on Alpine.js (x-data/x-init) to auto-dismiss after 4 seconds — the data-ts
// attribute forces the swapped node to differ each call so re-triggering works.
//
// @boundary HTTP
// @side-effects writes HTML response
// @ai-context an unrecognized non-empty variant is NOT defaulted: both map
// lookups miss and the toast renders with an empty message and no accent color
func HandleToast(w http.ResponseWriter, r *http.Request) {
	variant := r.FormValue("variant")
	if variant == "" {
		variant = "success"
	}

	messages := map[string]string{
		"success": "Operation completed successfully!",
		"error":   "Something went wrong. Please try again.",
		"info":    "Here's some useful information.",
		"warning": "Careful — this action cannot be undone.",
	}

	colors := map[string]string{
		"success": "border-emerald-500 bg-emerald-500/10",
		"error":   "border-destructive bg-destructive/10",
		"info":    "border-primary bg-primary/10",
		"warning": "border-amber-500 bg-amber-500/10",
	}

	msg := messages[variant]
	color := colors[variant]
	ts := time.Now().UnixMilli()

	w.Header().Set("Content-Type", "text/html")
	// 4s auto-dismiss: enough time to read the message, short enough that
	// toasts don't pile up if several are triggered in a row.
	fmt.Fprintf(w, `<div id="toast-container" hx-swap-oob="innerHTML">
	<div x-data="{ show: true }" x-show="show" x-transition
		x-init="setTimeout(() => show = false, 4000)"
		class="border-l-4 rounded-md px-4 py-3 shadow-lg text-sm %s" data-ts="%d">
		<div class="flex items-center justify-between">
			<span>%s</span>
			<button @click="show = false" class="ml-4 text-muted-foreground hover:text-foreground">&times;</button>
		</div>
	</div>
</div>`, color, ts, msg)
}
