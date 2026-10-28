package wflint

import "github.com/a-h/templ/parser/v2"

// walker collects violations for one file.
type walker struct {
	file string
	out  []Violation
}

func (w *walker) report(r parser.Range, rule, msg, hint string) {
	w.out = append(w.out, Violation{
		File:    w.file,
		Line:    int(r.From.Line) + 1,
		Col:     int(r.From.Col) + 1,
		Rule:    rule,
		Message: msg,
		Hint:    hint,
	})
}

// nodes walks a node list. inFence is true inside a CustomPage/CustomBlock
// escape hatch, where the element and attribute rules do not apply. Fence
// detection arrives in a later task; until then inFence is always false.
func (w *walker) nodes(ns []parser.Node, inFence bool) {
	for _, n := range ns {
		w.node(n, inFence)
	}
}

func (w *walker) node(n parser.Node, inFence bool) {
	switch n := n.(type) {
	case *parser.Element:
		if !inFence {
			w.checkElement(n)
		}
		w.nodes(n.Children, inFence)
	case *parser.RawElement:
		if !inFence {
			w.checkRawElement(n)
		}
	case *parser.ScriptElement:
		// <script> is allowed in app code (behavior is not design); only
		// class and style are checked on it.
		if !inFence {
			w.checkAttrs(n.Range, n.Attributes, true)
		}
	case *parser.TemplElementExpression:
		// Fence hook: a later task inspects n.Expression for CustomPage /
		// CustomBlock and walks the children with inFence = true.
		w.nodes(n.Children, inFence)
	case *parser.IfExpression:
		w.nodes(n.Then, inFence)
		for _, ei := range n.ElseIfs {
			w.nodes(ei.Then, inFence)
		}
		w.nodes(n.Else, inFence)
	case *parser.ForExpression:
		w.nodes(n.Children, inFence)
	case *parser.SwitchExpression:
		for _, c := range n.Cases {
			w.nodes(c.Children, inFence)
		}
	default:
		// Text, Whitespace, HTMLComment, GoCode, StringExpression,
		// ChildrenExpression, CallTemplateExpression and the like carry no
		// elements or attributes for these rules.
	}
}
