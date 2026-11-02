// Package diffview renders file diffs as tables: one <details> per file with
// a status badge and the path(s) in its summary, one table per hunk with
// old/new line-number gutters and a +/-/space marker column.
//
// Build Files by hand or call Parse on the text `git diff` (or any unified
// diff) produced. Mode selects the layout: ModeInline (the default) is one
// row per line; ModeSideBySide pairs each deletion with the addition that
// follows it, old text left, new text right.
//
// Side-by-side rendering strategy: the component renders the hunk twice,
// an inline table shown below md (`md:hidden`) and a side-by-side table
// shown from md up (inside a `hidden overflow-x-auto md:block` wrapper).
// Two renderings were chosen over one
// CSS-grid table that collapses because the two layouts do not share a row
// order (side-by-side pairs a deletion with the following addition; inline
// keeps the diff's own order), so one DOM could not serve both without
// JavaScript. The cost is duplicated markup for side-by-side diffs only.
//
// Phone width (390px): every diff is inline, the number gutters are narrow
// (w-8, enough for four digits), the file header wraps (badge, then the
// path), and a long line scrolls inside its hunk (overflow-x-auto on the
// hunk, whitespace-pre on the text cell) rather than widening the page.
package diffview

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/badge"
)

// Status is a file's change type, shown as a badge in the file header.
type Status string

const (
	// StatusModified is the zero value: the file exists on both sides.
	StatusModified Status = ""
	StatusAdded    Status = "added"
	StatusDeleted  Status = "deleted"
	StatusRenamed  Status = "renamed"
)

// Kind is a line's role in a hunk.
type Kind string

const (
	// KindContext is the zero value: an unchanged line present on both sides.
	KindContext Kind = ""
	KindAdd     Kind = "add"
	KindDel     Kind = "del"
)

// Mode selects the hunk layout.
type Mode string

const (
	// ModeInline is the zero value: one row per line, old and new numbers
	// side by side in the gutter.
	ModeInline Mode = ""
	// ModeSideBySide shows old text left and new text right from md up and
	// falls back to ModeInline below md.
	ModeSideBySide Mode = "side-by-side"
)

// Line is one line of a hunk. OldNo and NewNo are 1-based; a deletion has
// no NewNo and an addition no OldNo (0 renders as an empty gutter cell).
type Line struct {
	Kind  Kind
	OldNo int
	NewNo int
	Text  string
}

// Hunk is one contiguous block of lines with its "@@ -a,b +c,d @@" header
// (any trailing function context is kept in Header as-is).
type Hunk struct {
	Header string
	Lines  []Line
}

// File is one file's diff. Path is the new path; OldPath is set for renames
// (and for deletions, where it equals Path). Collapsed closes this file's
// <details> on first render.
type File struct {
	Path      string
	OldPath   string
	Status    Status
	Hunks     []Hunk
	Collapsed bool
}

// Props configures a DiffView.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes
	Files      []File
	Mode       Mode
	// Collapsed closes every file's <details> on first render; a File's
	// own Collapsed closes just that file.
	Collapsed bool
}

// pair is one side-by-side row: a left (old) and a right (new) cell, either
// of which may be absent (nil) when a change has no counterpart.
type pair struct {
	Left  *Line
	Right *Line
}

// pairLines builds side-by-side rows from a hunk's lines: context lines
// occupy both cells; a run of deletions followed by a run of additions is
// zipped index by index, and the longer run's tail gets an empty
// counterpart.
func pairLines(lines []Line) []pair {
	var out []pair
	var dels, adds []*Line
	flush := func() {
		n := len(dels)
		if len(adds) > n {
			n = len(adds)
		}
		for i := 0; i < n; i++ {
			var p pair
			if i < len(dels) {
				p.Left = dels[i]
			}
			if i < len(adds) {
				p.Right = adds[i]
			}
			out = append(out, p)
		}
		dels, adds = nil, nil
	}
	for i := range lines {
		l := &lines[i]
		switch l.Kind {
		case KindDel:
			if len(adds) > 0 {
				flush()
			}
			dels = append(dels, l)
		case KindAdd:
			adds = append(adds, l)
		default:
			flush()
			out = append(out, pair{Left: l, Right: l})
		}
	}
	flush()
	return out
}

// isCollapsed reports whether f renders closed.
func (p Props) isCollapsed(f File) bool {
	return p.Collapsed || f.Collapsed
}

// statusLabel is the badge text for a status.
func statusLabel(s Status) string {
	switch s {
	case StatusAdded:
		return "Added"
	case StatusDeleted:
		return "Deleted"
	case StatusRenamed:
		return "Renamed"
	default:
		return "Modified"
	}
}

// statusVariant picks the badge tone for a status.
func statusVariant(s Status) badge.Variant {
	switch s {
	case StatusAdded:
		return badge.VariantSuccess
	case StatusDeleted:
		return badge.VariantDestructive
	case StatusRenamed:
		return badge.VariantInfo
	default:
		return badge.VariantSecondary
	}
}

// modeAttr, statusAttr and kindAttr name the zero values for data-*
// attributes so tests and CSS can target them.
func modeAttr(m Mode) string {
	if m == ModeSideBySide {
		return string(ModeSideBySide)
	}
	return "inline"
}

func statusAttr(s Status) string {
	if s == StatusModified {
		return "modified"
	}
	return string(s)
}

func kindAttr(k Kind) string {
	if k == KindContext {
		return "context"
	}
	return string(k)
}

// marker is the +/-/space column for a line kind.
func marker(k Kind) string {
	switch k {
	case KindAdd:
		return "+"
	case KindDel:
		return "-"
	default:
		return " "
	}
}

// rowClass tints an inline row by kind.
func rowClass(k Kind) string {
	switch k {
	case KindAdd:
		return "bg-success/10"
	case KindDel:
		return "bg-destructive/10"
	default:
		return ""
	}
}

// markerClass tones the marker column by kind.
func markerClass(k Kind) string {
	switch k {
	case KindAdd:
		return "text-success"
	case KindDel:
		return "text-destructive"
	default:
		return "text-muted-foreground"
	}
}

// sideClass tints one side-by-side text cell: nil is a gap, a deletion on
// the left and an addition on the right are tinted, context is plain.
func sideClass(l *Line) string {
	if l == nil {
		return "bg-muted/40"
	}
	return rowClass(l.Kind)
}

// num formats a gutter number; 0 is blank.
func num(n int) string {
	if n <= 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// gutterNo is the number shown in a side-by-side gutter for one side.
func gutterNo(l *Line, old bool) string {
	if l == nil {
		return ""
	}
	if old {
		return num(l.OldNo)
	}
	return num(l.NewNo)
}

// text returns a line's text or "" for a gap.
func text(l *Line) string {
	if l == nil {
		return ""
	}
	return l.Text
}

// summaryID gives each file header an id for the collapse toggle's aria.
func summaryID(base string, i int) string {
	if base == "" {
		return ""
	}
	return base + "-file-" + strconv.Itoa(i)
}

// ErrNoFile is returned by Parse when a hunk or content line appears before
// any file header.
var ErrNoFile = errors.New("diffview: hunk before any file header")

// Parse reads a unified diff (as written by `git diff`, `diff -u` or `git
// format-patch`) into Files. It understands `diff --git` headers with
// `rename from`/`rename to`, `new file mode`, `deleted file mode`,
// `similarity index` and `index` lines, the `---`/`+++` pair (with the a/
// and b/ prefixes stripped and /dev/null meaning added or deleted), `@@`
// hunk headers with optional counts, and `\ No newline at end of file`
// markers (dropped). Diffs without `diff --git` lines (plain `---`/`+++`
// pairs) parse too. Any other line outside a hunk is ignored; a content
// line before any file, or a malformed `@@` header, is an error.
func Parse(unified string) ([]File, error) {
	var files []File
	var cur *File
	var hunk *Hunk
	var oldNo, newNo, oldLeft, newLeft int
	inHunk := false
	oldSeen := false // a --- line was read for cur (plain ---/+++ diffs)

	startFile := func() {
		files = append(files, File{})
		cur = &files[len(files)-1]
		hunk = nil
		inHunk = false
		oldSeen = false
	}

	lines := strings.Split(strings.TrimSuffix(unified, "\n"), "\n")
	for i, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		if strings.HasPrefix(line, "diff --git ") {
			startFile()
			if a, b, ok := splitGitPaths(strings.TrimPrefix(line, "diff --git ")); ok {
				cur.OldPath = a
				cur.Path = b
			}
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			if cur == nil {
				return nil, fmt.Errorf("line %d: %w", i+1, ErrNoFile)
			}
			h, err := parseHunkHeader(line)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			cur.Hunks = append(cur.Hunks, Hunk{Header: line})
			hunk = &cur.Hunks[len(cur.Hunks)-1]
			oldNo, newNo, oldLeft, newLeft = h.oldStart, h.newStart, h.oldCount, h.newCount
			inHunk = oldLeft > 0 || newLeft > 0
			continue
		}
		if inHunk {
			if line == "" {
				// A context line whose single leading space was stripped
				// (some mailers do that).
				line = " "
			}
			switch line[0] {
			case ' ':
				hunk.Lines = append(hunk.Lines, Line{Kind: KindContext, OldNo: oldNo, NewNo: newNo, Text: line[1:]})
				oldNo++
				newNo++
				oldLeft--
				newLeft--
			case '+':
				hunk.Lines = append(hunk.Lines, Line{Kind: KindAdd, NewNo: newNo, Text: line[1:]})
				newNo++
				newLeft--
			case '-':
				hunk.Lines = append(hunk.Lines, Line{Kind: KindDel, OldNo: oldNo, Text: line[1:]})
				oldNo++
				oldLeft--
			case '\\':
				// "\ No newline at end of file": dropped, counts untouched.
			default:
				return nil, fmt.Errorf("line %d: diffview: unexpected %q inside a hunk", i+1, line)
			}
			if oldLeft <= 0 && newLeft <= 0 {
				inHunk = false
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "--- "):
			if cur == nil || oldSeen {
				// A ---/+++ pair without a diff --git line starts a file.
				startFile()
			}
			oldSeen = true
			p := stripPrefix(strings.TrimPrefix(line, "--- "))
			if p == "/dev/null" {
				cur.Status = StatusAdded
			} else {
				cur.OldPath = p
			}
		case strings.HasPrefix(line, "+++ "):
			if cur == nil {
				return nil, fmt.Errorf("line %d: %w", i+1, ErrNoFile)
			}
			p := stripPrefix(strings.TrimPrefix(line, "+++ "))
			if p == "/dev/null" {
				cur.Status = StatusDeleted
			} else {
				cur.Path = p
			}
		case strings.HasPrefix(line, "rename from "):
			if cur != nil {
				cur.OldPath = strings.TrimPrefix(line, "rename from ")
				cur.Status = StatusRenamed
			}
		case strings.HasPrefix(line, "rename to "):
			if cur != nil {
				cur.Path = strings.TrimPrefix(line, "rename to ")
				cur.Status = StatusRenamed
			}
		case strings.HasPrefix(line, "new file mode"):
			if cur != nil {
				cur.Status = StatusAdded
			}
		case strings.HasPrefix(line, "deleted file mode"):
			if cur != nil {
				cur.Status = StatusDeleted
			}
		}
	}

	for i := range files {
		f := &files[i]
		switch f.Status {
		case StatusAdded:
			f.OldPath = ""
		case StatusDeleted:
			if f.Path == "" || f.OldPath != "" {
				f.Path = f.OldPath
			}
		case StatusRenamed:
		default:
			if f.OldPath != "" && f.Path != "" && f.OldPath != f.Path {
				f.Status = StatusRenamed
			} else {
				f.OldPath = ""
			}
		}
	}
	return files, nil
}

// splitGitPaths splits the "a/x b/y" tail of a diff --git line. Paths with
// spaces are ambiguous there; the ---/+++ and rename lines fix them up.
func splitGitPaths(s string) (string, string, bool) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return stripPrefix(parts[0]), stripPrefix(parts[1]), true
}

// stripPrefix removes git's a/ or b/ prefix and any "\tTIMESTAMP" tail
// diff -u appends.
func stripPrefix(p string) string {
	if i := strings.IndexByte(p, '\t'); i >= 0 {
		p = p[:i]
	}
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "a/") || strings.HasPrefix(p, "b/") {
		return p[2:]
	}
	return p
}

// hunkRange is the parsed "@@ -a,b +c,d @@" header.
type hunkRange struct {
	oldStart, oldCount, newStart, newCount int
}

// parseHunkHeader reads the ranges from "@@ -a[,b] +c[,d] @@ ..."; an
// omitted count is 1.
func parseHunkHeader(h string) (hunkRange, error) {
	var r hunkRange
	rest := strings.TrimPrefix(h, "@@ ")
	end := strings.Index(rest, " @@")
	if end < 0 {
		return r, fmt.Errorf("diffview: malformed hunk header %q", h)
	}
	ranges := strings.Fields(rest[:end])
	if len(ranges) != 2 || !strings.HasPrefix(ranges[0], "-") || !strings.HasPrefix(ranges[1], "+") {
		return r, fmt.Errorf("diffview: malformed hunk header %q", h)
	}
	var err error
	if r.oldStart, r.oldCount, err = parseRange(ranges[0][1:]); err != nil {
		return r, fmt.Errorf("diffview: malformed hunk header %q: %w", h, err)
	}
	if r.newStart, r.newCount, err = parseRange(ranges[1][1:]); err != nil {
		return r, fmt.Errorf("diffview: malformed hunk header %q: %w", h, err)
	}
	return r, nil
}

// parseRange reads "start[,count]"; count defaults to 1.
func parseRange(s string) (start, count int, err error) {
	count = 1
	if i := strings.IndexByte(s, ','); i >= 0 {
		if count, err = strconv.Atoi(s[i+1:]); err != nil {
			return 0, 0, err
		}
		s = s[:i]
	}
	start, err = strconv.Atoi(s)
	return start, count, err
}
