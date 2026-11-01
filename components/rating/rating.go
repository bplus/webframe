// Package rating renders a row of stars, either read-only (a value out of
// Max, announced as "3 of 5") or as an input: a fieldset of visually hidden
// radios whose labels are the stars, so the browser owns the selection and
// the keyboard (arrow keys move between radios natively) and the fill follows
// the checked radio and the hovered star through CSS alone; there is no
// script.
//
// Form contract: input mode submits Name=<n> with n in 1..Max as a plain
// radio group; the handler reads strconv.Atoi(r.FormValue(name)). Nothing is
// submitted when no star is chosen (Value 0 and the user left it), which
// Required turns into a browser validation error inside a form.
//
// Phone width (390px): each star is its own tap target, 24px at SizeSm, 32px
// at SizeMd and 44px at SizeLg (the recommended minimum for touch, so a
// rating a user is asked to give should be SizeLg). Five stars at SizeLg are
// 236px wide and fit; a wider Max wraps onto a second row.
package rating

import (
	"fmt"
	"strconv"

	"github.com/a-h/templ"
)

// Size sets the tap target and the star glyph. The zero value is SizeMd.
type Size string

const (
	// SizeSm is a 24px target with a 16px star: table cells, metadata rows.
	SizeSm Size = "sm"
	// SizeMd is a 32px target with a 20px star (zero value).
	SizeMd Size = ""
	// SizeLg is a 44px target with a 28px star: the size for a rating the
	// user is asked to give on a phone.
	SizeLg Size = "lg"
)

// DefaultMax is the number of stars when Max is zero or negative.
const DefaultMax = 5

// Props configures Rating. Class merges after the built-in classes;
// Attributes spread onto the root (the <span role="img"> when ReadOnly, the
// <fieldset> otherwise), after the built-in attributes.
type Props struct {
	// ID is the root's id. When empty, ControlID derives one from Name.
	ID         string
	Class      string
	Attributes templ.Attributes

	// Name is the radio group name in input mode (submits Name=<n>).
	Name string
	// Value is the number of lit stars (read-only) or the checked radio
	// (input). 0 lights nothing; a value above Max lights every star when
	// read-only and checks nothing when an input.
	Value int
	// Max is the number of stars; DefaultMax (5) when zero or negative.
	Max int
	// ReadOnly renders static stars with aria-label="<Value> of <Max>".
	ReadOnly bool
	Size     Size
	// Label is the fieldset legend in input mode (a visually hidden
	// "Rating" when empty) and prefixes the aria-label when read-only
	// ("Quality: 3 of 5").
	Label string
	// Required puts `required` on the radio group, so a form will not
	// submit without a star chosen. Input mode only.
	Required bool
	// Disabled disables the fieldset (every radio with it). Input mode only.
	Disabled bool
}

// max returns the effective star count.
func (p Props) max() int {
	if p.Max <= 0 {
		return DefaultMax
	}
	return p.Max
}

// ControlID is the id of the element that receives the field wiring
// (aria-describedby, aria-invalid): the fieldset in input mode, the span in
// read-only mode. It is ID when set, else "rating-"+Name, else "rating".
func (p Props) ControlID() string {
	if p.ID != "" {
		return p.ID
	}
	if p.Name != "" {
		return "rating-" + p.Name
	}
	return "rating"
}

// radioID is the id of the n-th radio: ControlID()+"-"+n.
func (p Props) radioID(n int) string {
	return p.ControlID() + "-" + strconv.Itoa(n)
}

// ariaLabel is the read-only announcement: "3 of 5", prefixed with Label
// when set.
func (p Props) ariaLabel() string {
	s := fmt.Sprintf("%d of %d", p.Value, p.max())
	if p.Label != "" {
		return p.Label + ": " + s
	}
	return s
}

// starText is the visually hidden name of the n-th radio.
func starText(n int) string {
	if n == 1 {
		return "1 star"
	}
	return strconv.Itoa(n) + " stars"
}

// iconSize is the star glyph size in px for a Size.
func (p Props) iconSize() int {
	switch p.Size {
	case SizeSm:
		return 16
	case SizeLg:
		return 28
	default:
		return 20
	}
}

// targetClass is the tap target box per Size: 24px, 32px, 44px.
func (p Props) targetClass() string {
	switch p.Size {
	case SizeSm:
		return "h-6 w-6"
	case SizeLg:
		return "h-11 w-11"
	default:
		return "h-8 w-8"
	}
}

// Star colors: lit stars use the warning token, unlit a faded muted
// foreground. The read-only mode applies them to the icon directly; input
// mode applies them to the star labels through the fill rules below.
const (
	filledClass = "text-warning"
	emptyClass  = "text-muted-foreground/40"
)

// starLabelClass is the class of one star label in input mode. The fill is
// pure CSS, in DOM order <input><label><input><label>…:
//
//   - a label is lit when its own radio is checked (input:checked + label)
//     or when a later radio is checked (label:has(~ input:checked)), i.e.
//     every star up to the chosen one;
//   - while the pointer is over the row (.wf-rating:hover), that fill is
//     reset and the hovered label plus every label before it
//     (label:has(~ label:hover)) light instead, so the user previews the
//     value they are about to pick;
//   - the label of the focused radio shows a focus ring, since the radio
//     itself is visually hidden.
//
// The reset rule's specificity (.wf-rating:hover .x) beats the checked
// rules (input:checked + .x) and loses to the hover rules
// (.wf-rating:hover .x:hover), which is what makes the preview win while
// hovering and the checked fill return when the pointer leaves.
//
// Every class below is written out in full (not assembled from filledClass
// and emptyClass) because the stylesheet's scanner only compiles complete
// literal tokens.
func (p Props) starLabelClass() string {
	return "inline-flex cursor-pointer items-center justify-center rounded-sm transition-colors " +
		p.targetClass() +
		" text-muted-foreground/40" +
		" [input:checked+&]:text-warning" +
		" [&:has(~input:checked)]:text-warning" +
		" [.wf-rating:hover_&]:text-muted-foreground/40" +
		" [.wf-rating:hover_&:hover]:text-warning" +
		" [.wf-rating:hover_&:has(~label:hover)]:text-warning" +
		" [input:focus-visible+&]:ring-2 [input:focus-visible+&]:ring-ring [input:focus-visible+&]:ring-offset-2 [input:focus-visible+&]:ring-offset-background"
}

// starIconClass is the class of a read-only star: lit or not.
func starIconClass(lit bool) string {
	if lit {
		return filledClass
	}
	return emptyClass
}
