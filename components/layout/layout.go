// Package layout provides the spacing and grouping primitives app code
// uses to place content next to or under other content without writing
// raw block elements or classes.
package layout

// Gap is the spacing between children. The zero value is medium.
type Gap string

const (
	GapM    Gap = ""
	GapNone Gap = "none"
	GapXS   Gap = "xs"
	GapS    Gap = "s"
	GapL    Gap = "l"
	GapXL   Gap = "xl"
)

// Align is cross-axis alignment. The zero value stretches.
type Align string

const (
	AlignStretch Align = ""
	AlignStart   Align = "start"
	AlignCenter  Align = "center"
	AlignEnd     Align = "end"
)

// Justify is main-axis distribution. The zero value packs at the start.
type Justify string

const (
	JustifyStart   Justify = ""
	JustifyCenter  Justify = "center"
	JustifyEnd     Justify = "end"
	JustifyBetween Justify = "between"
)

func gapClass(g Gap) string {
	switch g {
	case GapNone:
		return "gap-0"
	case GapXS:
		return "gap-1"
	case GapS:
		return "gap-2"
	case GapL:
		return "gap-6"
	case GapXL:
		return "gap-10"
	default:
		return "gap-4"
	}
}

func alignClass(a Align) string {
	switch a {
	case AlignStart:
		return "items-start"
	case AlignCenter:
		return "items-center"
	case AlignEnd:
		return "items-end"
	default:
		return "items-stretch"
	}
}

func justifyClass(j Justify) string {
	switch j {
	case JustifyCenter:
		return "justify-center"
	case JustifyEnd:
		return "justify-end"
	case JustifyBetween:
		return "justify-between"
	default:
		return "justify-start"
	}
}
