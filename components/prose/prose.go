// Package prose renders free-form content (rendered Markdown, a CMS body,
// a help page, item notes) inside a scoped typography block, so headings,
// paragraphs, lists, links, code, tables, blockquotes, images and rules
// look right without a single class on the content. It is the one
// sanctioned route for content HTML: app code may not call templ.Raw
// (wflint rejects it), and Prose is where that call lives.
//
// Two inputs, one of which is set:
//
//   - HTML is trusted markup, written verbatim. The caller is responsible
//     for its safety: HTML is for content the app authored itself or
//     already sanitized (Markdown rendered server-side with a sanitizer,
//     a hand-written help page). Never put user input in HTML without
//     sanitizing it first; use Text for anything untrusted.
//   - Text is plain text, escaped. Blank lines separate paragraphs
//     (<p>), a single newline inside a paragraph becomes <br>. A <script>
//     tag in Text is text on the page.
//
// With both set, HTML wins and Text is ignored.
//
// Size scales the whole block; MaxWidth caps the measure (65ch by default).
//
// On narrow screens images never exceed the width, long words wrap, and
// <pre> blocks and tables scroll horizontally inside their own box. For a
// table that box is a <div class="wf-prose-table"> Prose puts around every
// top-level <table> in HTML, so the table itself keeps display: table and
// with it its table role: a screen reader still offers row and column
// navigation. The HTML needs nothing extra for this.
package prose

import (
	"regexp"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// HTML is trusted markup, such as Markdown rendered server-side through a
// sanitizer. Prose writes it verbatim, so whoever builds an HTML value is
// responsible for its safety. Use Props.Text for untrusted text.
type HTML string

// Size scales the block's font size; spacing follows in em. The zero value
// is SizeMd.
type Size string

const (
	// SizeSm is 0.875rem: sidebars, captions, dense help text.
	SizeSm Size = "sm"
	// SizeMd is 1rem (zero value).
	SizeMd Size = ""
	// SizeLg is 1.125rem: a long-form article.
	SizeLg Size = "lg"
)

// MaxWidth caps the measure. The zero value is MaxWidthProse.
type MaxWidth string

const (
	// MaxWidthProse caps the block at 65ch, the classic readable measure
	// (zero value).
	MaxWidthProse MaxWidth = ""
	// MaxWidthWide caps the block at 100ch: content with tables or wide
	// code blocks.
	MaxWidthWide MaxWidth = "wide"
	// MaxWidthNone lets the block fill its container.
	MaxWidthNone MaxWidth = "none"
)

// Props configures Prose. Set HTML or Text; with both, HTML wins.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	// HTML is trusted markup written verbatim. See the HTML type.
	HTML HTML
	// Text is plain text, escaped: blank lines separate paragraphs, a
	// single newline becomes <br>.
	Text string
	// Size scales the block. Zero is SizeMd.
	Size Size
	// MaxWidth caps the measure. Zero is MaxWidthProse (65ch).
	MaxWidth MaxWidth
}

// rootClass is the block's class list: the .wf-prose scope the stylesheet
// styles, its size modifier, the measure cap, then the caller's Class.
func (p Props) rootClass() string {
	return utils.TwMerge(
		"wf-prose w-full min-w-0",
		p.sizeClass(),
		p.maxWidthClass(),
		p.Class,
	)
}

// sizeClass is the modifier the .wf-prose CSS keys the font size on.
func (p Props) sizeClass() string {
	switch p.Size {
	case SizeSm:
		return "wf-prose--sm"
	case SizeLg:
		return "wf-prose--lg"
	default:
		return ""
	}
}

// maxWidthClass is the measure cap: Tailwind's max-w-prose (65ch) by
// default, 100ch for Wide, nothing for None.
func (p Props) maxWidthClass() string {
	switch p.MaxWidth {
	case MaxWidthWide:
		return "max-w-[100ch]"
	case MaxWidthNone:
		return ""
	default:
		return "max-w-prose"
	}
}

// paragraphBreak matches a blank line (a newline, optional horizontal
// whitespace, one or more newlines): the paragraph separator in Text.
var paragraphBreak = regexp.MustCompile(`\n[ \t]*\n+`)

// paragraphs splits Text into paragraphs on blank lines and each paragraph
// into its lines; surrounding whitespace is trimmed, empty paragraphs are
// dropped. The templ renders each inner slice as one <p> with <br> between
// its lines.
func (p Props) paragraphs() [][]string {
	text := strings.ReplaceAll(p.Text, "\r\n", "\n")
	var out [][]string
	for _, para := range paragraphBreak.Split(text, -1) {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		lines := strings.Split(para, "\n")
		for i, l := range lines {
			lines[i] = strings.TrimSpace(l)
		}
		out = append(out, lines)
	}
	return out
}

// paragraphHTML is the body of one Text paragraph: every line through
// templ.EscapeString, joined with <br>. The only markup in the result is
// the <br> this function writes, so the templ may render it with Raw.
func paragraphHTML(lines []string) string {
	esc := make([]string, len(lines))
	for i, l := range lines {
		esc[i] = templ.EscapeString(l)
	}
	return strings.Join(esc, "<br>")
}

// tableWrapOpen and tableWrapClose surround each top-level table in HTML.
// The .wf-prose-table rules in input.css make the div the horizontal
// scroller, so the table can stay display: table.
const (
	tableWrapOpen  = `<div class="wf-prose-table">`
	tableWrapClose = `</div>`
)

// wrapTables returns html with every top-level <table>...</table> wrapped
// in tableWrapOpen/tableWrapClose. A table nested in another table's cell
// is left alone (the outer wrapper already scrolls). Tag names match
// case-insensitively, text inside <!-- --> comments is skipped, and a table
// with no closing tag is wrapped up to the end of html. Everything else is
// copied byte for byte.
func wrapTables(html string) string {
	lower := strings.ToLower(html)
	if !strings.Contains(lower, "<table") {
		return html
	}
	var b strings.Builder
	b.Grow(len(html) + 64)
	depth := 0
	for i := 0; i < len(html); {
		switch {
		case strings.HasPrefix(lower[i:], "<!--"):
			end := strings.Index(lower[i+4:], "-->")
			if end < 0 {
				b.WriteString(html[i:])
				i = len(html)
				continue
			}
			end = i + 4 + end + 3
			b.WriteString(html[i:end])
			i = end
		case isTag(lower, i, "<table"):
			if depth == 0 {
				b.WriteString(tableWrapOpen)
			}
			depth++
			b.WriteByte(html[i])
			i++
		case depth > 0 && isTag(lower, i, "</table"):
			end := strings.IndexByte(html[i:], '>')
			if end < 0 {
				end = len(html) - 1 - i
			}
			end = i + end + 1
			b.WriteString(html[i:end])
			i = end
			depth--
			if depth == 0 {
				b.WriteString(tableWrapClose)
			}
		default:
			b.WriteByte(html[i])
			i++
		}
	}
	if depth > 0 {
		b.WriteString(tableWrapClose)
	}
	return b.String()
}

// isTag reports whether lower[i:] starts with name (such as "<table") as a
// whole tag name: followed by whitespace, ">" or "/", or the end of input,
// so "<tablet>" is not a table.
func isTag(lower string, i int, name string) bool {
	if !strings.HasPrefix(lower[i:], name) {
		return false
	}
	j := i + len(name)
	if j == len(lower) {
		return true
	}
	switch lower[j] {
	case ' ', '\t', '\n', '\r', '\f', '>', '/':
		return true
	}
	return false
}
