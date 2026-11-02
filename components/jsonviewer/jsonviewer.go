// Package jsonviewer renders any Go value as a server-rendered, collapsible
// JSON tree.
//
// The value is first marshalled through encoding/json, so maps, structs
// (honoring their json tags), slices and json.RawMessage all render the same
// way, then re-read as a token stream so object keys keep the order the
// marshaller wrote them in (struct field order; map keys are already sorted
// by encoding/json). Every object and array is a native <details>/<summary>
// pair, so expanding and collapsing needs no JavaScript; Alpine is used only
// by the optional copy-path buttons (through copybutton).
//
// On narrow screens long strings break anywhere and the indent stops
// growing after six levels, so deep trees still fit; the frame scrolls
// horizontally only when a single line cannot.
package jsonviewer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/a-h/templ"
)

// maxIndentDepth is the deepest nesting level that still adds its own
// indent; children of nodes at this depth or deeper sit flush with their
// parent's children and rely on the summary line to show nesting.
const maxIndentDepth = 6

// Props configures a JSONViewer. Value is the only required field.
type Props struct {
	ID         string
	Class      string
	Attributes templ.Attributes
	// Value is any Go value encoding/json can marshal. Marshalling errors
	// render as a destructive-toned message in place of the tree.
	Value any
	// Collapsed is the depth from which objects and arrays start closed:
	// 1 keeps the root open and closes its children, 2 also opens the
	// children, and so on. 0 renders every node open.
	Collapsed int
	// MaxDepth elides objects and arrays nested deeper than this many
	// levels below the root: an elided node renders as "…" with its size
	// instead of its children. 0 renders every level. Scalars inside the
	// deepest rendered container are still printed.
	MaxDepth int
	// CopyPath adds a copy button to every node that copies its JSON path
	// (for example $.items[0].name) to the clipboard.
	CopyPath bool
	// Sort orders every object's keys alphabetically instead of keeping
	// the order the marshaller wrote them in.
	Sort bool
}

// kind is the JSON type of a node.
type kind int

const (
	kindNull kind = iota
	kindBool
	kindNumber
	kindString
	kindArray
	kindObject
)

// node is one JSON value, with ordered children for containers. keys is set
// for objects only and runs parallel to children.
type node struct {
	kind     kind
	text     string // the literal for scalars (strings unquoted)
	keys     []string
	children []*node
}

// isContainer reports whether the node is an object or array.
func (n *node) isContainer() bool {
	return n.kind == kindArray || n.kind == kindObject
}

// size is the number of entries in a container.
func (n *node) size() int {
	return len(n.children)
}

// sizeLabel is the "{n}" or "[n]" shown on a container's summary line.
func (n *node) sizeLabel() string {
	if n.kind == kindObject {
		return "{" + strconv.Itoa(n.size()) + "}"
	}
	return "[" + strconv.Itoa(n.size()) + "]"
}

// emptyLabel is the literal for an empty container.
func (n *node) emptyLabel() string {
	if n.kind == kindObject {
		return "{}"
	}
	return "[]"
}

// scalarText is the display text for a scalar node: strings quoted,
// everything else as the JSON literal.
func (n *node) scalarText() string {
	if n.kind == kindString {
		return strconv.Quote(n.text)
	}
	return n.text
}

// scalarClass is the tone for a scalar node, from theme tokens only.
func (n *node) scalarClass() string {
	switch n.kind {
	case kindString:
		return "text-success break-all"
	case kindNumber:
		return "text-info"
	case kindBool:
		return "text-warning"
	default:
		return "text-muted-foreground italic"
	}
}

// build marshals v and parses the result into a node tree, sorting object
// keys when sortKeys is set.
func build(v any, sortKeys bool) (*node, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	n, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("jsonviewer: trailing data after value")
	}
	if sortKeys {
		n.sortKeys()
	}
	return n, nil
}

// parseValue reads one JSON value from the decoder's token stream.
func parseValue(dec *json.Decoder) (*node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return parseFromToken(dec, tok)
}

// parseFromToken builds the node for an already-read token, consuming the
// rest of a container's tokens when tok opens one.
func parseFromToken(dec *json.Decoder, tok json.Token) (*node, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			n := &node{kind: kindObject}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("jsonviewer: object key %v is not a string", keyTok)
				}
				child, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				n.keys = append(n.keys, key)
				n.children = append(n.children, child)
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, err
			}
			return n, nil
		case '[':
			n := &node{kind: kindArray}
			for dec.More() {
				child, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				n.children = append(n.children, child)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return n, nil
		}
		return nil, fmt.Errorf("jsonviewer: unexpected delimiter %q", t)
	case nil:
		return &node{kind: kindNull, text: "null"}, nil
	case bool:
		return &node{kind: kindBool, text: strconv.FormatBool(t)}, nil
	case json.Number:
		return &node{kind: kindNumber, text: t.String()}, nil
	case string:
		return &node{kind: kindString, text: t}, nil
	}
	return nil, fmt.Errorf("jsonviewer: unexpected token %T", tok)
}

// sortKeys orders every object's keys alphabetically, recursively.
func (n *node) sortKeys() {
	if n.kind == kindObject {
		idx := make([]int, len(n.keys))
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool { return n.keys[idx[a]] < n.keys[idx[b]] })
		keys := make([]string, len(idx))
		children := make([]*node, len(idx))
		for i, j := range idx {
			keys[i] = n.keys[j]
			children[i] = n.children[j]
		}
		n.keys, n.children = keys, children
	}
	for _, c := range n.children {
		c.sortKeys()
	}
}

// childPath extends a JSON path with an object key or array index. Keys
// that are identifiers use dot notation; anything else is bracketed and
// quoted so the path stays unambiguous.
func childPath(parent string, key string, index int, isObject bool) string {
	if !isObject {
		return parent + "[" + strconv.Itoa(index) + "]"
	}
	if isIdentifier(key) {
		return parent + "." + key
	}
	return parent + "[" + strconv.Quote(key) + "]"
}

// isIdentifier reports whether s can follow a dot in a JSON path.
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}

// isOpen reports whether a container at depth renders expanded.
func (p Props) isOpen(depth int) bool {
	return p.Collapsed <= 0 || depth < p.Collapsed
}

// isElided reports whether a container at depth renders as "…" with its
// size instead of its children.
func (p Props) isElided(depth int) bool {
	return p.MaxDepth > 0 && depth > p.MaxDepth
}

// childrenClass is the indent for the children of a container at depth;
// it stops growing past maxIndentDepth so deep trees keep fitting.
func childrenClass(depth int) string {
	if depth < maxIndentDepth {
		return "flex flex-col pl-4"
	}
	return "flex flex-col"
}

// keyLabel is the "key: " prefix for an object member; array elements and
// the root have none.
func keyLabel(key string, isMember bool) string {
	if !isMember {
		return ""
	}
	return strconv.Quote(key) + ":"
}

// errorText is the message rendered when the value cannot be marshalled.
func errorText(err error) string {
	return "cannot render value: " + strings.TrimSpace(err.Error())
}
