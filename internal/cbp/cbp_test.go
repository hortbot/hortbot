package cbp_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/cbp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestParse(t *testing.T) {
	runTest := func(name string, input string, result ...cbp.Node) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			c, err := cbp.Parse(input)
			assert.Check(t, err)
			assert.Check(t, cmp.DeepEqual(result, c, cmpopts.EquateEmpty()))
		})
	}

	runTest("Empty string", "")
	runTest("No actions", "this is a test", cbp.TextNode(`this is a test`))
	runTest("Special characters", "this is a (cool) test_1234", cbp.TextNode(`this is a (cool) test_1234`))

	runTest("Single action", "this is a (_USER_) test",
		cbp.TextNode(`this is a `),
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
		cbp.TextNode(` test`),
	)

	runTest("Single action at start", "(_USER_) test",
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
		cbp.TextNode(` test`),
	)

	runTest("Single action at end", "this is a (_USER_)",
		cbp.TextNode(`this is a `),
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
	)

	runTest("Multiple actions", "this is a (_USER_) test (_FOO BAR_) thing",
		cbp.TextNode(`this is a `),
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
		cbp.TextNode(` test `),
		cbp.ActionNode(
			cbp.TextNode(`FOO BAR`),
		),
		cbp.TextNode(` thing`),
	)

	runTest("Nested action", "this is a (_USER (_NAME_) WOW_) test",
		cbp.TextNode(`this is a `),
		cbp.ActionNode(
			cbp.TextNode(`USER `),
			cbp.ActionNode(
				cbp.TextNode(`NAME`),
			),
			cbp.TextNode(` WOW`),
		),
		cbp.TextNode(` test`),
	)

	runTest("RIP counter", "RIP count reset to (_VARS_RIP-(_GAME_CLEAN_)_SET_0_), neat.",
		cbp.TextNode(`RIP count reset to `),
		cbp.ActionNode(
			cbp.TextNode(`VARS_RIP-`),
			cbp.ActionNode(
				cbp.TextNode(`GAME_CLEAN`),
			),
			cbp.TextNode(`_SET_0`),
		),
		cbp.TextNode(`, neat.`),
	)

	runTest("Open paren at end with action", "how (_ok_) cool(",
		cbp.TextNode(`how `),
		cbp.ActionNode(
			cbp.TextNode("ok"),
		),
		cbp.TextNode(` cool(`),
	)

	runTest("Underscore at end with action", "how (_ok_) cool_",
		cbp.TextNode(`how `),
		cbp.ActionNode(
			cbp.TextNode("ok"),
		),
		cbp.TextNode(` cool_`),
	)

	runTest("Underscore next to open", "(_HEY_(_TESTING_ERROR_)_)",
		cbp.ActionNode(
			cbp.TextNode(`HEY_`),
			cbp.ActionNode(
				cbp.TextNode("TESTING_ERROR"),
			),
		),
	)

	runTest("Underscore next to close", "(_(_TESTING_ERROR_)_HEY_)",
		cbp.ActionNode(
			cbp.ActionNode(
				cbp.TextNode("TESTING_ERROR"),
			),
			cbp.TextNode(`_HEY`),
		),
	)
}

func TestParseError(t *testing.T) {
	runTest := func(name string, input string, want cbp.Error, errMsg string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			c, err := cbp.Parse(input)
			assert.Check(t, cmp.Equal(want, err))
			assert.Check(t, cmp.Error(err, errMsg))
			assert.Check(t, cmp.DeepEqual([]cbp.Node{}, c, cmpopts.EquateEmpty()))
		})
	}

	runTest(
		"Unclosed action",
		"this action (_ is not closed",
		cbp.Error{
			Pos:  28,
			Code: cbp.ErrorMissingClose,
		},
		"syntax error at position 28; input ended unexpectedly",
	)

	runTest(
		"Unexpected close",
		"this action has a _) inside",
		cbp.Error{
			Pos:  18,
			Code: cbp.ErrorUnexpectedClose,
		},
		"syntax error at position 18; unexpected action close",
	)

	runTest(
		"Bad nesting",
		"foo (_ bar (_ baz _) fob",
		cbp.Error{
			Pos:  24,
			Code: cbp.ErrorMissingClose,
		},
		"syntax error at position 24; input ended unexpectedly",
	)
}

func TestBadErrorCode(t *testing.T) {
	err := cbp.Error{
		Pos:  123,
		Code: -1,
	}

	assert.Check(t, cmp.Error(err, "syntax error at position 123; unknown error code -1"))
}

func BenchmarkParse(b *testing.B) {
	runBenchmark := func(name string, input string) {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = cbp.Parse(input)
			}
		})
	}

	runBenchmark("No actions", "This is just a cool test.")
	runBenchmark("Parens", "This is just a cool test. (Or is it?)")
	runBenchmark("Single action", "Hello, (_USER_)!")
	runBenchmark("Nested", "RIP count reset to (_VARS_RIP-(_GAME_CLEAN_)_SET_0_), neat.")
}
