package redis

import (
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"gotest.tools/v3/assert"
)

func TestBuildKey(t *testing.T) {
	tests := []struct {
		input []keyPair
		want  string
	}{
		{
			input: []keyPair{keyStr("first").is("v")},
			want:  "first:v",
		},
		{
			input: []keyPair{keyStr("first").is(""), keyStr("second").is("sec")},
			want:  "first::second:sec",
		},
		{
			input: []keyPair{keyStr("first").is(""), keyStr("second").is("sec"), keyStr("third").is("ok")},
			want:  "first::second:sec:third:ok",
		},
	}

	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			got := buildKey(test.input...)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestBuildKeyPanic(t *testing.T) {
	assertx.Panic(t, func() {
		buildKey()
	}, "no key specified")

	assertx.Panic(t, func() {
		buildKey(keyStr("this").is("bad:value"))
	}, "key contains colon: bad:value")
}
