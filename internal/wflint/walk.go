package wflint

import (
	"strings"

	"github.com/a-h/templ/parser/v2"
)

// walker collects violations for one file.
type walker struct {
	file   string
	out    []Violation
	fences []Fence
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
// escape hatch, where every rule except fence-reason is suppressed.
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
		if kind, ok := fenceOf(n.Expression.Value); ok {
			w.fence(n, kind)
			w.nodes(n.Children, true)
			return
		}
		if !inFence {
			w.reportGo(n.Expression, true)
		}
		w.nodes(n.Children, inFence)
	case *parser.CallTemplateExpression:
		// The older {! expr } call syntax: same checks as @expr.
		if !inFence {
			w.reportGo(n.Expression, true)
		}
	case *parser.GoCode:
		if !inFence {
			w.reportGo(n.Expression, false)
		}
	case *parser.StringExpression:
		if !inFence {
			w.reportGo(n.Expression, false)
		}
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
		// Text, Whitespace, HTMLComment, ChildrenExpression and the like
		// carry nothing these rules check.
	}
}

// fence records a CustomPage/CustomBlock use and reports fence-reason when
// its Reason is absent or empty. It applies inside fences too: a nested fence
// is still a fence.
func (w *walker) fence(n *parser.TemplElementExpression, kind string) {
	r := pick(n.Expression.Range, n.Range)
	reason, ok := fenceReason(n.Expression.Value)
	w.fences = append(w.fences, Fence{File: w.file, Line: int(r.From.Line) + 1, Kind: kind, Reason: reason})
	if !ok {
		w.report(r, "fence-reason",
			"Custom"+strings.ToUpper(kind[:1])+kind[1:]+" needs a Reason given as a non-empty double-quoted string literal explaining why components cannot be used", "")
	}
}
