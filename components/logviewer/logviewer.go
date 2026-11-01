// Package logviewer renders a text log in a fixed-height monospace box with
// a toolbar: a client-side filter, a Follow switch, and a download action.
// It is the text-stream counterpart of trafficstream (which stays the
// tabular one).
//
// Lines render on the server; with Src the lines container polls it with
// htmx (hx-get, hx-trigger="every Ns", hx-swap="beforeend") and the handler
// answers with Fragment(lines): the new lines only, or an empty body when
// there is nothing new. Alpine state inside the component owns Follow (the
// newest line stays in view only while the user is already at the bottom;
// a manual scroll up stays put), the filter (hides non-matching lines,
// case-insensitive, text and time), and MaxLines (the oldest lines are
// removed after each poll, with the scroll position corrected so what the
// user is reading does not move). Appending never resets the scroll.
//
// The box has one of three fixed heights (Height) and scrolls inside, so it
// never grows the page however many lines arrive.
//
// Phone (390px): the toolbar wraps (filter box full width, Follow and
// Download on the next row); the box keeps its Height tier; long lines
// wrap when Wrap is set and otherwise scroll horizontally inside the box,
// never widening the page.
package logviewer

import (
	"fmt"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/button"
	"github.com/bplus/webframe/utils"
)

// Level is the severity tone of one Line. The zero value is untoned text.
type Level string

const (
	// LevelNone is plain text (zero value).
	LevelNone Level = ""
	// LevelInfo tints the line with the info token.
	LevelInfo Level = "info"
	// LevelWarn tints the line with the warning token.
	LevelWarn Level = "warn"
	// LevelError tints the line with the destructive token.
	LevelError Level = "error"
)

// Line is one log line. At is optional: the zero time renders no <time>.
type Line struct {
	Text  string
	Level Level
	At    time.Time
}

// Height is the box's fixed height. The zero value is HeightMd.
type Height string

const (
	// HeightMd is 400px (zero value).
	HeightMd Height = ""
	// HeightSm is 240px.
	HeightSm Height = "sm"
	// HeightLg is 600px.
	HeightLg Height = "lg"
)

// DefaultEvery is the polling interval used when Props.Every is zero.
const DefaultEvery = 5 * time.Second

// Props configures one LogViewer.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	// Lines are rendered on the server, oldest first. With MaxLines set,
	// only the last MaxLines of them render.
	Lines []Line
	// Src is the URL the lines container polls with hx-get every Every;
	// the handler responds with Fragment(newLines) or an empty body.
	// Empty means no polling.
	Src string
	// Every is the polling interval; zero means DefaultEvery (5s).
	Every time.Duration
	// Follow is the Follow switch's initial state: scroll to the newest
	// line after each poll, but only while the user is already at the
	// bottom. On first render a following box starts scrolled to its
	// bottom. The switch renders only with Src (without polling nothing
	// arrives to follow); Follow without Src still starts at the bottom.
	Follow bool
	// Filter adds a search box that hides non-matching lines client-side.
	Filter bool
	// Wrap soft-wraps long lines; otherwise they scroll horizontally
	// inside the box.
	Wrap bool
	// MaxLines keeps at most this many lines in the box; the oldest are
	// dropped after each poll (and before first render). Zero is unbounded.
	MaxLines int
	// Height is the box's fixed height tier; zero is HeightMd (400px).
	Height Height
	// Download renders this action (a link to the raw log, say) at the
	// toolbar's right.
	Download *action.Action
	// Empty is the text shown while there are no lines; it hides once a
	// poll appends one.
	Empty string
}

// visible is Lines trimmed to the last MaxLines.
func (p Props) visible() []Line {
	if p.MaxLines > 0 && len(p.Lines) > p.MaxLines {
		return p.Lines[len(p.Lines)-p.MaxLines:]
	}
	return p.Lines
}

// hasToolbar reports whether anything renders above the box.
func (p Props) hasToolbar() bool {
	return p.Filter || p.Src != "" || p.Download != nil
}

// every formats the hx-trigger interval: whole seconds as "5s", anything
// finer in milliseconds.
func (p Props) every() string {
	d := p.Every
	if d <= 0 {
		d = DefaultEvery
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("every %ds", int64(d/time.Second))
	}
	return fmt.Sprintf("every %dms", d.Milliseconds())
}

func heightClass(h Height) string {
	switch h {
	case HeightSm:
		return "h-[240px]"
	case HeightLg:
		return "h-[600px]"
	}
	return "h-[400px]"
}

func (p Props) rootClass() string {
	return utils.TwMerge(
		"wf-log flex min-w-0 flex-col overflow-hidden rounded-md border border-border bg-card text-card-foreground",
		p.Class,
	)
}

func (p Props) boxClass() string {
	return utils.TwMerge(
		"min-w-0 overflow-auto py-2 font-mono text-xs leading-5",
		"focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50",
		heightClass(p.Height),
	)
}

// linesClass sets white-space on the container so Fragment lines (which
// do not know Wrap) inherit it.
func (p Props) linesClass() string {
	return utils.IfElse(p.Wrap, "whitespace-pre-wrap break-words", "whitespace-pre")
}

// xData is the Alpine state (the root's x-effect="apply()" re-filters on
// every q change; it also runs once before the lines ref exists, which
// apply tolerates): follow (the switch), q (filter text), max
// (MaxLines, 0 = unbounded), n (line count, for the empty text), stick
// (the user is at the bottom). onSwap runs after every htmx append:
// apply the filter to the new lines, trim to max keeping the viewport
// still unless following, then scroll to the bottom only when following
// and already there.
func (p Props) xData() string {
	return fmt.Sprintf(`{
follow: %t, q: '', max: %d, n: %d, stick: true,
atBottom() { const b = this.$refs.box; return b.scrollHeight - b.scrollTop - b.clientHeight <= 4 },
onScroll() { this.stick = this.atBottom() },
apply() { const q = this.q.trim().toLowerCase(); const l = this.$refs.lines; if (!l) return; for (const el of l.children) { el.hidden = q !== '' && !el.textContent.toLowerCase().includes(q) } },
trim() { if (this.max <= 0) return; const b = this.$refs.box, l = this.$refs.lines; const before = b.scrollHeight; while (l.children.length > this.max) { l.firstElementChild.remove() } if (!(this.follow && this.stick)) { b.scrollTop = Math.max(0, b.scrollTop - (before - b.scrollHeight)) } },
toBottom() { const b = this.$refs.box; b.scrollTop = b.scrollHeight; this.stick = true },
onSwap() { this.apply(); this.trim(); this.n = this.$refs.lines.children.length; if (this.follow && this.stick) this.toBottom() },
init() { if (this.follow) this.$nextTick(() => this.toBottom()) }
}`, p.Follow, p.MaxLines, len(p.visible()))
}

// download is the Download action with a toolbar-sized default
// (button.SizeSm) when the caller left Size empty.
func (p Props) download() action.Action {
	a := *p.Download
	if a.Size == "" {
		a.Size = button.SizeSm
	}
	return a
}

// maxLinesAttr is the data-wf-log-max attribute value.
func (p Props) maxLinesAttr() string {
	return strconv.Itoa(p.MaxLines)
}

func toneClass(l Level) string {
	switch l {
	case LevelInfo:
		return "text-info"
	case LevelWarn:
		return "text-warning"
	case LevelError:
		return "text-destructive"
	}
	return ""
}

func lineClass(l Line) string {
	return utils.TwMerge("wf-log-line flex gap-2 px-3", toneClass(l.Level))
}

// atAttr is the <time> datetime value (RFC 3339).
func atAttr(t time.Time) string { return t.Format(time.RFC3339) }

// atText is the visible time, HH:MM:SS in the time's own location.
func atText(t time.Time) string { return t.Format("15:04:05") }

// lineText keeps an empty line one line tall under white-space: pre.
func lineText(s string) string {
	if s == "" {
		return " "
	}
	return s
}
