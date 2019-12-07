package bot_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/bot"
	"gotest.tools/v3/assert"
)

var splitFirstSepTests = []struct {
	input string
	sep   byte
	a     string
	b     string
}{
	{"", ' ', "", ""},
	{"foo", ' ', "foo", ""},
	{"foo bar", ' ', "foo", "bar"},
	{"foo bar baz", ' ', "foo", "bar baz"},
	{"foo  bar  baz", ' ', "foo", " bar  baz"},
	{"foo/bar baz", '/', "foo", "bar baz"},
}

func TestHelperSplitFirstSep(t *testing.T) {
	for _, test := range splitFirstSepTests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			gotA, gotB := bot.SplitFirstSep(test.input, test.sep)
			assert.Equal(t, test.a, gotA)
			assert.Equal(t, test.b, gotB)
		})
	}
}

var strSink1, strSink2 string

func BenchmarkHelperSplitFirstSep(b *testing.B) {
	for _, test := range splitFirstSepTests {
		test := test
		b.Run(test.input, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				strSink1, strSink2 = bot.SplitFirstSep(test.input, test.sep)
			}
		})
	}
}
