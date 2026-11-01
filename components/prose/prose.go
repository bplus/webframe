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
// Sizing: Size scales the whole block (the spacing is in em, so it
// follows); MaxWidth caps the measure at 65ch by default, so a long article
// stays readable on a wide screen. Colors come from the theme tokens, so
// the block follows the light and dark themes like every component.
//
// Phone width (390px): the max width yields to the viewport (the block is
// w-full and min-w-0, the cap is only a maximum), images never exceed the
// width, long words and URLs in paragraphs and inline code wrap
// (overflow-wrap: anywhere), <pre> blocks and tables scroll horizontally
// inside their own box (overflow-x: auto; a table is display: block for
// that) instead of widening the page.
package prose

import (
	"regexp"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// HTML is trusted markup: content the app authored or already sanitized,
// such as Markdown rendered server-side through a sanitizer. Prose writes
// it verbatim (templ.Raw) inside the .wf-prose scope, so whoever builds an
// HTML value is responsible for what it contains. For text from users or
// any other untrusted source use Props.Text, which is escaped.
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
