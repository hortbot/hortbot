package cbp_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/cbp"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		nodes []cbp.Node
	}{
		{
			name: "Empty string",
		},
		{
			name:  "No actions",
			input: `this is a test`,
			nodes: []cbp.Node{cbp.TextNode(`this is a test`)},
		},
		{
			name:  "Special characters",
			input: `this is a (cool) test_1234`,
			nodes: []cbp.Node{cbp.TextNode(`this is a (cool) test_1234`)},
		},
		{
			name:  "Single action",
			input: `this is a (_USER_) test`,
			nodes: []cbp.Node{
				cbp.TextNode(`this is a `),
				cbp.ActionNode(
					cbp.TextNode(`USER`),
				),
				cbp.TextNode(` test`),
			},
		},
		{
			name:  "Single action at start",
			input: `(_USER_) test`,
			nodes: []cbp.Node{
				cbp.ActionNode(
					cbp.TextNode(`USER`),
				),
				cbp.TextNode(` test`),
			},
		},
		{
			name:  "Single action at end",
			input: `this is a (_USER_)`,
			nodes: []cbp.Node{
				cbp.TextNode(`this is a `),
				cbp.ActionNode(
					cbp.TextNode(`USER`),
				),
			},
		},
		{
			name:  "Multiple actions",
			input: `this is a (_USER_) test (_FOO BAR_) thing`,
			nodes: []cbp.Node{
				cbp.TextNode(`this is a `),
				cbp.ActionNode(
					cbp.TextNode(`USER`),
				),
				cbp.TextNode(` test `),
				cbp.ActionNode(
					cbp.TextNode(`FOO BAR`),
				),
				cbp.TextNode(` thing`),
			},
		},
		{
			name:  "Nested action",
			input: `this is a (_USER (_NAME_) WOW_) test`,
			nodes: []cbp.Node{
				cbp.TextNode(`this is a `),
				cbp.ActionNode(
					cbp.TextNode(`USER `),
					cbp.ActionNode(
						cbp.TextNode(`NAME`),
					),
					cbp.TextNode(` WOW`),
				),
				cbp.TextNode(` test`),
			},
		},
		{
			name:  "RIP counter",
			input: `RIP count reset to (_VARS_RIP-(_GAME_CLEAN_)_SET_0_), neat.`,
			nodes: []cbp.Node{
				cbp.TextNode(`RIP count reset to `),
				cbp.ActionNode(
					cbp.TextNode(`VARS_RIP-`),
					cbp.ActionNode(
						cbp.TextNode(`GAME_CLEAN`),
					),
					cbp.TextNode(`_SET_0`),
				),
				cbp.TextNode(`, neat.`),
			},
		},
		{
			name:  "Open paren at end with action",
			input: `how (_ok_) cool(`,
			nodes: []cbp.Node{
				cbp.TextNode(`how `),
				cbp.ActionNode(
					cbp.TextNode("ok"),
				),
				cbp.TextNode(` cool(`),
			},
		},
		{
			name:  "Underscore at end with action",
			input: `how (_ok_) cool_`,
			nodes: []cbp.Node{
				cbp.TextNode(`how `),
				cbp.ActionNode(
					cbp.TextNode("ok"),
				),
				cbp.TextNode(` cool_`),
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c, err := cbp.Parse(test.input)
			assert.Check(t, err)
			assert.Check(t, cmp.DeepEqual(test.nodes, c, cmpopts.EquateEmpty()))
		})
	}
}

func TestParseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		err    cbp.Error
		errMsg string
	}{
		{
			name:  "Unclosed action",
			input: "this action (_ is not closed",
			err: cbp.Error{
				Pos:  28,
				Code: cbp.ErrorMissingClose,
			},
			errMsg: "syntax error at position 28; input ended unexpectedly",
		},
		{
			name:  "Unexpected close",
			input: "this action has a _) inside",
			err: cbp.Error{
				Pos:  18,
				Code: cbp.ErrorUnexpectedClose,
			},
			errMsg: "syntax error at position 18; unexpected action close",
		},
		{
			name:  "Bad nesting",
			input: "foo (_ bar (_ baz _) fob",
			err: cbp.Error{
				Pos:  24,
				Code: cbp.ErrorMissingClose,
			},
			errMsg: "syntax error at position 24; input ended unexpectedly",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c, err := cbp.Parse(test.input)
			assert.Check(t, cmp.Nil(c))
			assert.Check(t, cmp.Equal(test.err, err))
			assert.Check(t, cmp.Error(err, test.errMsg))
		})
	}
}

func TestBadErrorCode(t *testing.T) {
	t.Parallel()

	err := cbp.Error{
		Pos:  123,
		Code: -1,
	}

	assert.Check(t, cmp.Error(err, "syntax error at position 123; unknown error code -1"))
}

func BenchmarkParse(b *testing.B) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "No actions",
			input: `This is just a cool test.`,
		},
		{
			name:  "Parens",
			input: `This is just a cool test. (Or is it?)`,
		},
		{
			name:  "Single action",
			input: `Hello, (_USER_)!`,
		},
		{
			name:  "Nested",
			input: `RIP count reset to (_VARS_RIP-(_GAME_CLEAN_)_SET_0_), neat.`,
		},
	}

	for _, test := range tests {
		test := test
		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = cbp.Parse(test.input)
			}
		})
	}
}
