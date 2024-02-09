package stringsx_test

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/stringsx"
	"gotest.tools/v3/assert"
)

var splitByteTests = []struct {
	input string
	sep   byte
	a, b  string
}{
	{"", ' ', "", ""},
	{"foo", ' ', "foo", ""},
	{"foo bar", ' ', "foo", "bar"},
	{"foo bar baz", ' ', "foo", "bar baz"},
	{"foo  bar  baz", ' ', "foo", " bar  baz"},
	{"foo/bar baz", '/', "foo", "bar baz"},
}

func TestSplitByte(t *testing.T) {
	for _, test := range splitByteTests {
		t.Run(test.input, func(t *testing.T) {
			gotA, gotB := stringsx.SplitByte(test.input, test.sep)
			assert.Equal(t, test.a, gotA)
			assert.Equal(t, test.b, gotB)
		})
	}
}

func TestSplitOnByte(t *testing.T) {
	for _, test := range splitByteTests {
		t.Run(test.input, func(t *testing.T) {
			gotA, gotB := stringsx.Split(test.input, string(test.sep))
			assert.Equal(t, test.a, gotA)
			assert.Equal(t, test.b, gotB)
		})
	}
}

var splitTests = []struct {
	input string
	sep   string
	a, b  string
}{
	{"", " ", "", ""},
	{"foo", "", "foo", ""},
	{"what", " ", "what", ""},
	{"foo  bar", "  ", "foo", "bar"},
	{"https://something.com", "://", "https", "something.com"},
	{"huh what", ",,", "huh what", ""},
}

func TestSplit(t *testing.T) {
	for _, test := range splitTests {
		t.Run(test.input, func(t *testing.T) {
			gotA, gotB := stringsx.Split(test.input, test.sep)
			assert.Equal(t, test.a, gotA)
			assert.Equal(t, test.b, gotB)
		})
	}
}
