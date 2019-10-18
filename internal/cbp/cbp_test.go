package cbp_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hortbot/hortbot/internal/cbp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestParse(t *testing.T) {
	runTest := func(name string, input string, malformed bool, result ...cbp.Node) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			c, gotMalformed := cbp.Parse(input)
			assert.Check(t, cmp.DeepEqual(result, c, cmpopts.EquateEmpty()))
			assert.Check(t, cmp.Equal(malformed, gotMalformed))
			assert.Check(t, cmp.Equal(input, cbp.NodesString(c)))
		})
	}

	runTest("Empty string", "", false)
	runTest("No actions", "this is a test", false, cbp.TextNode(`this is a test`))
	runTest("Special characters", "this is a (cool) test_1234", false, cbp.TextNode(`this is a (cool) test_1234`))

	runTest("Single action", "this is a (_USER_) test", false,
		cbp.TextNode(`this is a `),
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
		cbp.TextNode(` test`),
	)

	runTest("Single action at start", "(_USER_) test", false,
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
		cbp.TextNode(` test`),
	)

	runTest("Single action at end", "this is a (_USER_)", false,
		cbp.TextNode(`this is a `),
		cbp.ActionNode(
			cbp.TextNode(`USER`),
		),
	)

	runTest("Multiple actions", "this is a (_USER_) test (_FOO BAR_) thing", false,
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

	runTest("Nested action", "this is a (_USER (_NAME_) WOW_) test", false,
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

	runTest("RIP counter", "RIP count reset to (_VARS_RIP-(_GAME_CLEAN_)_SET_0_), neat.", false,
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

	runTest("Open paren at end with action", "how (_ok_) cool(", false,
		cbp.TextNode(`how `),
		cbp.ActionNode(
			cbp.TextNode("ok"),
		),
		cbp.TextNode(` cool(`),
	)

	runTest("Underscore at end with action", "how (_ok_) cool_", false,
		cbp.TextNode(`how `),
		cbp.ActionNode(
			cbp.TextNode("ok"),
		),
		cbp.TextNode(` cool_`),
	)

	runTest("Underscore next to open", "(_HEY_(_TESTING_ERROR_)_)", false,
		cbp.ActionNode(
			cbp.TextNode(`HEY_`),
			cbp.ActionNode(
				cbp.TextNode("TESTING_ERROR"),
			),
		),
	)

	runTest("Underscore next to close", "(_(_TESTING_ERROR_)_HEY_)", false,
		cbp.ActionNode(
			cbp.ActionNode(
				cbp.TextNode("TESTING_ERROR"),
			),
			cbp.TextNode(`_HEY`),
		),
	)

	// Malformed inputs

	runTest("Unclosed action", "this action (_ is not closed", true,
		cbp.TextNode(`this action (_ is not closed`),
	)

	runTest("Unexpected close", "this action has a _) inside", true,
		cbp.TextNode(`this action has a _) inside`),
	)

	runTest("Bad nesting", "foo (_ bar (_ baz _) fob", true,
		cbp.TextNode(`foo (_ bar `),
		cbp.ActionNode(
			cbp.TextNode(" baz "),
		),
		cbp.TextNode(` fob`),
	)

	runTest("ASCII art", "(_USER_) rubs on (_PARAMETER_) 's booty! PogChamp / (_(_|", true,
		cbp.ActionNode(
			cbp.TextNode("USER"),
		),
		cbp.TextNode(` rubs on `),
		cbp.ActionNode(
			cbp.TextNode("PARAMETER"),
		),
		cbp.TextNode(` 's booty! PogChamp / (_(_|`),
	)

	runTest("Underscore URL", "follow Necomi on twitter (http://twitter.com/_necomie_) (_ONLINE_CHECK_)", true,
		cbp.TextNode(`follow Necomi on twitter (http://twitter.com/_necomie_) `),
		cbp.ActionNode(
			cbp.TextNode("ONLINE_CHECK"),
		),
	)
}

func TestString(t *testing.T) {
	n := cbp.TextNode("foo")
	assert.Check(t, cmp.Equal("foo", n.String()))

	n2 := cbp.ActionNode(
		cbp.TextNode("hello"),
		cbp.ActionNode(
			cbp.TextNode("what"),
		),
		cbp.TextNode("there"),
	)
	assert.Check(t, cmp.Equal("(_hello(_what_)there_)", n2.String()))
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
	runBenchmark("ASCII art", "(_USER_) rubs on (_PARAMETER_) 's booty! PogChamp / (_(_|")
}
