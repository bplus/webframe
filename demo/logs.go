package demo

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/logviewer"
	"github.com/bplus/webframe/demo/pages"
	"github.com/bplus/webframe/utils"
)

// The Logs page (GET /logs, in the sidebar shell) is a logviewer over a
// demo ring buffer that every Items mutation appends a line to. The page
// renders the buffer and polls /logs/tail every three seconds; the poll
// carries a cursor the page was given when it rendered, so each open page
// receives every line appended since its own last poll, once, and a 204
// when there is nothing new.

// logBufferSize is how many lines the ring keeps and how many the viewer
// shows (MaxLines); older lines are dropped on both sides.
const logBufferSize = 200

// logPollEvery is the viewer's polling interval.
const logPollEvery = 3 * time.Second

// maxLogCursors caps the cursors the buffer remembers; past it an arbitrary
// one is forgotten (a page that polls with a forgotten cursor is re-seated
// at the newest line and goes on from there).
const maxLogCursors = 256

// LogBuffer is the demo's in-memory log: a ring of the last logBufferSize
// lines, each numbered by a sequence that keeps counting after the ring
// wraps, plus the poll cursors (sequence last served) of the open pages.
// It is safe for concurrent use.
type LogBuffer struct {
	mu      sync.Mutex
	lines   []logLine
	seq     int
	cursors map[string]int
}

// logLine is one buffered line and its sequence number.
type logLine struct {
	seq  int
	line logviewer.Line
}

// NewLogBuffer returns an empty buffer.
func NewLogBuffer() *LogBuffer {
	return &LogBuffer{cursors: make(map[string]int)}
}

// seedLogs writes the lines a fresh demo server starts with, so /logs is
// never empty: the store seed and the one protected item, as info and a
// warning, timestamped now.
func seedLogs(b *LogBuffer, now time.Time) *LogBuffer {
	b.append(logviewer.LevelInfo, "demo started; items store seeded with 14 items", now.Add(-2*time.Second))
	b.append(logviewer.LevelNone, "settings loaded for account Morgan", now.Add(-time.Second))
	b.append(logviewer.LevelWarn, "item #1 Blade of the Fallen Star is protected: deletes are refused", now)
	return b
}

// Logs is the demo server's log. Tests replace it with a fresh one.
var Logs = seedLogs(NewLogBuffer(), time.Now())

// Append adds one line, timestamped now, dropping the oldest when the ring
// is full.
func (b *LogBuffer) Append(level logviewer.Level, text string) {
	b.append(level, text, time.Now())
}

func (b *LogBuffer) append(level logviewer.Level, text string, at time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	b.lines = append(b.lines, logLine{seq: b.seq, line: logviewer.Line{Text: text, Level: level, At: at}})
	if len(b.lines) > logBufferSize {
		b.lines = b.lines[len(b.lines)-logBufferSize:]
	}
}

// Lines returns a copy of every buffered line, oldest first.
func (b *LogBuffer) Lines() []logviewer.Line {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.linesAfter(0)
}

// linesAfter returns the lines with a sequence above seq. The caller holds
// the lock.
func (b *LogBuffer) linesAfter(seq int) []logviewer.Line {
	var out []logviewer.Line
	for _, l := range b.lines {
		if l.seq > seq {
			out = append(out, l.line)
		}
	}
	return out
}

// Cursor registers a new poll cursor at the newest line and returns its id:
// the first Since with it answers only lines appended from now on. The
// page renders Lines() and polls with the cursor it was given, so nothing
// is shown twice.
func (b *LogBuffer) Cursor() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := utils.RandomID()
	b.seat(id)
	return id
}

// seat records cursor id at the newest sequence, forgetting an arbitrary
// cursor first when the table is full. The caller holds the lock.
func (b *LogBuffer) seat(id string) {
	if len(b.cursors) >= maxLogCursors {
		for k := range b.cursors {
			delete(b.cursors, k)
			break
		}
	}
	b.cursors[id] = b.seq
}

// Since returns the lines appended since the cursor's last Since (or its
// creation) and moves the cursor past them. A cursor the buffer does not
// know (one it forgot, or one from before a restart) is seated at the
// newest line and returns nothing: the page goes on receiving new lines
// without a reload; the lines it missed were never on it anyway.
func (b *LogBuffer) Since(cursor string) []logviewer.Line {
	b.mu.Lock()
	defer b.mu.Unlock()
	seq, ok := b.cursors[cursor]
	if !ok {
		b.seat(cursor)
		return nil
	}
	out := b.linesAfter(seq)
	b.cursors[cursor] = b.seq
	return out
}

// logsProps declares the Logs page's viewer: the whole buffer, polling
// /logs/tail with a fresh cursor every logPollEvery, following, with the
// filter box, trimmed to logBufferSize, the tall box.
func logsProps() logviewer.Props {
	return logviewer.Props{
		ID:       "demo-log",
		Lines:    Logs.Lines(),
		Src:      "/logs/tail?cursor=" + Logs.Cursor(),
		Every:    logPollEvery,
		Follow:   true,
		Filter:   true,
		MaxLines: logBufferSize,
		Height:   logviewer.HeightLg,
		Empty:    "Nothing logged yet. Create, edit, publish or delete an item and the line arrives here within three seconds.",
	}
}

// HandleLogs renders the Logs page (GET /logs) in the shell of the App in
// the request context (SidebarApp: the route is wrapped in its Middleware).
//
// @boundary HTTP
// @side-effects registers a poll cursor on Logs; writes HTML response
func HandleLogs(w http.ResponseWriter, r *http.Request) {
	app, err := appshell.Must(r.Context())
	if err != nil {
		log.Printf("demo: render logs: %v", err)
		http.Error(w, "no app in context", http.StatusInternalServerError)
		return
	}
	if err := app.Render(w, r, "Logs", pages.Logs(logsProps())); err != nil {
		log.Printf("demo: render logs: %v", err)
	}
}

// HandleLogsTail answers the viewer's poll (GET /logs/tail?cursor=ID): the
// lines appended since the cursor's last poll as a logviewer.Fragment
// (200), or 204 No Content when there is none, so an idle page costs a
// header exchange and nothing is appended. Without a cursor it answers the
// whole buffer (the raw log), 204 when that is empty too.
//
// @boundary HTTP
// @side-effects advances the cursor on Logs; writes HTML or an empty 204
func HandleLogsTail(w http.ResponseWriter, r *http.Request) {
	var lines []logviewer.Line
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		lines = Logs.Since(cursor)
	} else {
		lines = Logs.Lines()
	}
	if len(lines) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	fragment(w, r, "logs tail", logviewer.Fragment(lines))
}
