// Package codeblock renders a block of source text in a monospace frame:
// optional language label and copy button in a header row, optional
// CSS-counter line numbers, soft wrap or horizontal scroll, a capped
// height that scrolls inside, and a caption.
//
// It has no syntax highlighter: Language is a label on the header and a
// "language-<name>" class on the <code> element, nothing more. The text is
// HTML-escaped. Long lines scroll inside the block (or wrap, with Wrap),
// so it never widens the page.
package codeblock

import (
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/bplus/webframe/utils"
)

// Height caps the block; the <pre> scrolls inside it. The zero value is
// no cap: the block is as tall as its text.
type Height string

const (
	HeightNone Height = ""
	HeightSm   Height = "sm" // 240px
	HeightMd   Height = "md" // 400px
	HeightLg   Height = "lg" // 600px
)

// Props configures CodeBlock.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes

	// Code is the text shown, escaped. One trailing newline is dropped so
	// a file's last line is not followed by an empty numbered line; the
	// copy button still copies Code exactly as given.
	Code string
	// Language is the header label and the "language-<Language>" class on
	// the <code> element. Empty hides the label.
	Language string
	// LineNumbers numbers every line with a CSS counter (no JS); the
	// numbers are not selectable, so copying by selection copies code only.
	// Without Wrap the number gutter is sticky: it stays in view while a
	// long line is scrolled sideways (data-wf-code-scroll on the figure).
	LineNumbers bool
	// Copy adds a copybutton in the header that copies Code.
	Copy bool
	// Wrap soft-wraps long lines instead of scrolling them horizontally.
	Wrap bool
	// MaxHeight caps the <pre>, which then scrolls vertically.
	MaxHeight Height
	// Caption is a <figcaption> under the code.
	Caption string
}

func (p Props) hasHeader() bool {
	return p.Language != "" || p.Copy
}

// lines splits text() for line numbering.
func (p Props) lines() []string {
	return strings.Split(p.text(), "\n")
}

// text is Code with "\r\n" normalized to "\n" and one trailing newline
// dropped.
func (p Props) text() string {
	return strings.TrimSuffix(strings.ReplaceAll(p.Code, "\r\n", "\n"), "\n")
}

// digitsStyle sets --wf-code-digits to the digit count of the last line
// number; the .wf-code rules in input.css size the gutter from it. SafeCSS is
// sound because the value is computed here, never caller input.
func (p Props) digitsStyle() templ.SafeCSS {
	return templ.SafeCSS("--wf-code-digits:" + strconv.Itoa(len(strconv.Itoa(len(p.lines())))) + ";")
}

func (p Props) codeClass() string {
	if p.Language == "" {
		return ""
	}
	return "language-" + p.Language
}

func maxHeightClass(h Height) string {
	switch h {
	case HeightSm:
		return "max-h-[240px] overflow-y-auto"
	case HeightMd:
		return "max-h-[400px] overflow-y-auto"
	case HeightLg:
		return "max-h-[600px] overflow-y-auto"
	}
	return ""
}

func (p Props) rootClass() string {
	return utils.TwMerge(
		"wf-code min-w-0 overflow-hidden rounded-md border border-border bg-muted/40 text-sm",
		p.Class,
	)
}

func (p Props) preClass() string {
	return utils.TwMerge(
		"m-0 min-w-0 overflow-x-auto p-3 font-mono text-xs leading-relaxed text-foreground",
		"focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50",
		utils.IfElse(p.Wrap, "whitespace-pre-wrap break-words", "whitespace-pre"),
		maxHeightClass(p.MaxHeight),
	)
}

// scrollable reports whether the <pre> can scroll (no wrap, or a height
// cap), so it gets tabindex="0" for keyboard users.
func (p Props) scrollable() bool {
	return !p.Wrap || p.MaxHeight != HeightNone
}
