package wflint

import (
	"fmt"
	"slices"
	"strings"

	"github.com/a-h/templ/parser/v2"
)

// AllowedElements are the text-level HTML elements app code may use directly.
// An image is a sized block of content, not a word in a sentence, so <img> is
// not one: the app declares it through the image component, and webframe
// decides how it is sized and loaded.
var AllowedElements = []string{"p", "a", "span", "strong", "em", "b", "i", "code", "kbd", "br", "ul", "ol", "li", "time", "small", "sup", "sub"}

// AllowedAttrs are the attributes app code may set on allowed elements, in
// addition to any attribute starting with one of allowedAttrPrefixes.
// download is allowed because a link can sit inside a sentence and
// downloading is behavior, not design. role and tabindex stay forbidden:
// something focusable with a role is an interactive thing, and interactive
// things are components. width, height and loading belong to the image
// component.
var AllowedAttrs = []string{"href", "src", "alt", "title", "id", "name", "datetime", "target", "rel", "download"}

var allowedAttrPrefixes = []string{"aria-", "data-", "hx-", "x-"}

// classStyleBindings are the Alpine attributes that set class or style; they
// are forbidden even though x-* is otherwise allowed.
var classStyleBindings = []string{":class", "x-bind:class", ":style", "x-bind:style"}

// Hints name the webframe component to use instead of a disallowed element.
var Hints = map[string]string{
	"div":        "layout.Stack or layout.Inline",
	"section":    "layout.Section",
	"header":     "appshell or pageheader",
	"nav":        "appshell",
	"aside":      "layout.TwoColumn",
	"main":       "appshell",
	"footer":     "appshell",
	"table":      "datatable",
	"form":       "form.Form or formpage",
	"h1":         "pageheader",
	"h2":         "layout.Section",
	"h3":         "panel",
	"h4":         "panel",
	"h5":         "panel",
	"h6":         "panel",
	"hr":         "layout.Divider",
	"details":    "panel (Collapsible)",
	"dialog":     "dialog",
	"button":     "action.Button",
	"input":      "formfield",
	"select":     "selectbox",
	"textarea":   "textarea",
	"label":      "label",
	"img":        "image",
	"figure":     "figure",
	"figcaption": "figure",
	"dl":         "descriptionlist",
	"dt":         "descriptionlist",
	"dd":         "descriptionlist",
}

func (w *walker) checkElement(e *parser.Element) {
	name := strings.ToLower(e.Name)
	if !slices.Contains(AllowedElements, name) {
		// Attribute rules apply only to allowed elements: the element
		// itself is the violation, and replacing it removes its attributes.
		w.report(e.Range, "element",
			fmt.Sprintf("<%s> is not allowed in app code; use a webframe component", name),
			Hints[name])
		return
	}
	w.checkAttrs(e.Range, e.Attributes, false)
}

func (w *walker) checkRawElement(e *parser.RawElement) {
	name := strings.ToLower(e.Name)
	if name == "style" {
		w.report(e.Range, "style-element",
			"<style> is not allowed in app code; styling belongs inside webframe components", "")
		return
	}
	w.report(e.Range, "element",
		fmt.Sprintf("<%s> is not allowed in app code; use a webframe component", name),
		Hints[name])
}

// checkAttrs applies the attribute rules. elem is the owning element's range,
// used when an attribute carries no position of its own. With onlyClassStyle
// (used for <script>), only attr-class and attr-style are checked.
func (w *walker) checkAttrs(elem parser.Range, attrs []parser.Attribute, onlyClassStyle bool) {
	for _, a := range attrs {
		switch a := a.(type) {
		case *parser.ConstantAttribute:
			w.checkKey(elem, a.Key, onlyClassStyle)
		case *parser.ExpressionAttribute:
			w.checkKey(elem, a.Key, onlyClassStyle)
		case *parser.BoolConstantAttribute:
			w.checkKey(elem, a.Key, onlyClassStyle)
		case *parser.BoolExpressionAttribute:
			w.checkKey(elem, a.Key, onlyClassStyle)
		case *parser.ConditionalAttribute:
			w.checkAttrs(elem, a.Then, onlyClassStyle)
			w.checkAttrs(elem, a.Else, onlyClassStyle)
		case *parser.SpreadAttributes:
			if !onlyClassStyle {
				w.report(pick(a.Range, elem), "attr-spread",
					"attribute spread is not allowed in app code; pass props to a webframe component", "")
			}
		}
	}
}

func (w *walker) checkKey(elem parser.Range, k parser.AttributeKey, onlyClassStyle bool) {
	switch k := k.(type) {
	case parser.ExpressionAttributeKey:
		if onlyClassStyle {
			return
		}
		w.report(pick(k.Expression.Range, elem), "attr-dynamic-key",
			"attribute names computed at runtime are not allowed in app code", "")
		return
	case *parser.ExpressionAttributeKey:
		if onlyClassStyle {
			return
		}
		w.report(pick(k.Expression.Range, elem), "attr-dynamic-key",
			"attribute names computed at runtime are not allowed in app code", "")
		return
	}
	r := elem
	switch k := k.(type) {
	case parser.ConstantAttributeKey:
		r = pick(k.NameRange, elem)
	case *parser.ConstantAttributeKey:
		r = pick(k.NameRange, elem)
	}
	name := strings.ToLower(k.String())
	switch {
	case name == "class":
		w.report(r, "attr-class", "class is not allowed in app code; styling belongs inside webframe components", "")
	case name == "style":
		w.report(r, "attr-style", "style is not allowed in app code; styling belongs inside webframe components", "")
	case onlyClassStyle:
	case slices.Contains(classStyleBindings, name):
		// Checked before the x-* allowance: choosing a class is design,
		// not behavior.
		w.report(r, "attr-binding",
			fmt.Sprintf("Alpine %s is not allowed in app code; styling belongs inside webframe components", name), "")
	case slices.Contains(AllowedAttrs, name), hasAllowedPrefix(name):
	default:
		w.report(r, "attr-unknown",
			fmt.Sprintf("attribute %q is not allowed in app code", name),
			"allowed: "+strings.Join(AllowedAttrs, ", ")+", or aria-*, data-*, hx-*, x-*")
	}
}

func hasAllowedPrefix(name string) bool {
	for _, p := range allowedAttrPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// pick returns r unless it is the zero range, in which case it returns
// fallback.
func pick(r, fallback parser.Range) parser.Range {
	if r == (parser.Range{}) {
		return fallback
	}
	return r
}
