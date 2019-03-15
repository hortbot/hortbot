package cbp

import (
	"fmt"
	"strings"
)

const sliceCap = 3

func Parse(s string) ([]Node, error) {
	if s == "" {
		return nil, nil
	}

	switch {
	case strings.Contains(s, "(_"):
	case strings.Contains(s, "_)"):
	default:
		return []Node{TextNode(s)}, nil
	}

	sc := scanner{input: s}

	stack := make([]Node, 1, sliceCap)
	stack[0].Children = make([]Node, 0, sliceCap)
	curr := &stack[0].Children

	for sc.scan() {
		if sc.sep {
			if sc.open {
				stack = append(stack, Node{
					Children: make([]Node, 0, sliceCap),
				})
				curr = &stack[len(stack)-1].Children
			} else {
				if len(stack) <= 1 {
					return nil, Error{Pos: sc.idx - 2, Code: ErrorUnexpectedClose}
				}

				end := len(stack) - 1
				last := stack[end]

				curr, stack = &stack[end-1].Children, stack[:end]
				*curr = append(*curr, last)
			}
		} else if sc.text != "" {
			*curr = append(*curr, Node{
				Text: sc.text,
			})
		}
	}

	if len(stack) != 1 {
		return nil, Error{Pos: sc.idx, Code: ErrorMissingClose}
	}

	return *curr, nil
}

// Node is a CoeBot command node. If Text is set, then the node is a text node.
// Otherwise, Children will contain a list of subnodes.
type Node struct {
	Text     string
	Children []Node
}

func TextNode(s string) Node {
	return Node{
		Text: s,
	}
}

func ActionNode(nodes ...Node) Node {
	return Node{
		Children: nodes,
	}
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
	skipNext := false

	for i, r := range s {
		if skipNext {
			skipNext = false
			continue
		}

		if r == '(' || r == '_' {
			skipNext = true

			nextI := i + 1
			if nextI >= len(s) {
				break
			}
			r2 := s[nextI]

			if r == '(' && r2 == '_' {
				return true, true, i
			}

			if r == '_' && r2 == ')' {
				return true, false, i
			}
		}
	}

	return false, false, len(s)
}

type ErrorCode int

const (
	ErrorUnexpectedClose ErrorCode = iota
	ErrorMissingClose
)

// Error is returned from Parse if the command does not parse.
type Error struct {
	Pos  int
	Code ErrorCode
}

func (e Error) Error() string {
	switch e.Code {
	case ErrorUnexpectedClose:
		return fmt.Sprintf("syntax error at position %d; unexpected action close", e.Pos)
	case ErrorMissingClose:
		return fmt.Sprintf("syntax error at position %d; input ended unexpectedly", e.Pos)
	default:
		return fmt.Sprintf("syntax error at position %d; unknown error code %v", e.Pos, e.Code)
	}
}
