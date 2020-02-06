// Package cbp implements a CoeBot-style command parser.
package cbp

import "strings"

const sliceCap = 3

// Parse parses a string as a CoeBot command. Parsing always succeeds, but if
// the input was malformed and needed to be fixed (via recovery), malformed
// will be true.
func Parse(s string) (nodes []Node, malformed bool) {
	if s == "" {
		return nil, false
	}

	switch {
	case strings.Contains(s, "(_"):
	case strings.Contains(s, "_)"):
	default:
		return []Node{TextNode(s)}, false
	}

	sc := scanner{input: s}

	stack := make([]Node, 1, sliceCap)
	stack[0].Children = make([]Node, 0, sliceCap)
	curr := &stack[0].Children

	pop := func() Node {
		end := len(stack) - 1
		last := stack[end]

		curr, stack = &stack[end-1].Children, stack[:end]
		return last
	}

	for sc.scan() {
		if sc.sep {
			if sc.open {
				stack = append(stack, Node{
					Children: make([]Node, 0, sliceCap),
				})
				curr = &stack[len(stack)-1].Children
			} else {
				if len(stack) <= 1 {
					malformed = true
					appendText(curr, "_)")
					continue
				}

				last := pop()
				*curr = append(*curr, last)
			}
		} else if sc.text != "" {
			appendText(curr, sc.text)
		}
	}

	for len(stack) > 1 {
		malformed = true

		last := pop()
		appendText(curr, "(_")

		for _, child := range last.Children {
			if child.Text != "" {
				appendText(curr, child.Text)
			} else {
				*curr = append(*curr, child)
			}
		}
	}

	return *curr, malformed
}

func appendText(curr *[]Node, text string) {
	if end := len(*curr) - 1; end >= 0 && (*curr)[end].Text != "" {
		(*curr)[end].Text += text
		return
	}

	*curr = append(*curr, Node{
		Text: text,
	})
}

// Node is a CoeBot command node. If Text is set, then the node is a text node.
// Otherwise, Children will contain a list of subnodes.
type Node struct {
	Text     string
	Children []Node
}

// TextNode creates a new text-only Node.
func TextNode(s string) Node {
	return Node{
		Text: s,
	}
}

// ActionNode creates a new action node, with the given child nodes.
func ActionNode(nodes ...Node) Node {
	return Node{
		Children: nodes,
	}
}

func (n *Node) String() string {
	var b strings.Builder
	n.string(&b)
	return b.String()
}

func (n *Node) string(b *strings.Builder) {
	if n.Text != "" {
		b.WriteString(n.Text)
		return
	}

	b.WriteString("(_")
	nodesString(b, n.Children)
	b.WriteString("_)")
}

func nodesString(b *strings.Builder, nodes []Node) {
	for _, child := range nodes {
		child.string(b)
	}
}

// NodesString returns the nodes as a single string, as they'd
// appear before parsing. The string returned is never malformed.
func NodesString(nodes []Node) string {
	var b strings.Builder
	nodesString(&b, nodes)
	return b.String()
}

type scanner struct {
	input string
	idx   int

	text string

	sepNext bool
	sep     bool
	open    bool
}

func (s *scanner) scan() bool {
	if s.input == "" {
		return false
	}

	if s.sepNext {
		s.sepNext = false
		s.sep = true
		s.text, s.input = s.input[:2], s.input[2:]
		s.idx += 2
		return true
	}

	s.sep = false

	var end int
	s.sepNext, s.open, end = findSep(s.input)

	s.text, s.input = s.input[:end], s.input[end:]
	s.idx += len(s.text)
	return true
}

func findSep(s string) (found bool, open bool, end int) {
	var prev rune

	for i, r := range s {
		switch {
		case prev == '(' && r == '_':
			return true, true, i - 1
		case prev == '_' && r == ')':
			return true, false, i - 1
		}

		prev = r
	}

	return false, false, len(s)
}
