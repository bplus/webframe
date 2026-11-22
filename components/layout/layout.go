// Package layout provides the spacing and grouping primitives app code
// uses to place content next to or under other content without writing
// raw block elements or classes: Stack, Inline, Section, Divider, TwoColumn
// and ThreeColumn place content; Scroll is a box with a height limit that
// scrolls a long list of components inside the page.
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

// Side is which edge of a TwoColumn the aside sits on. The zero value
// is right.
type Side string

const (
	SideRight Side = ""
	SideLeft  Side = "left"
)

// AsideWidth is the width of a side column from the md breakpoint up.
// The zero value is medium.
type AsideWidth string

const (
	AsideM AsideWidth = ""
	AsideS AsideWidth = "s"
	AsideL AsideWidth = "l"
)

func asideWidthClass(w AsideWidth) string {
	switch w {
	case AsideS:
		return "md:w-64"
	case AsideL:
		return "md:w-96"
	default:
		return "md:w-80"
	}
}
