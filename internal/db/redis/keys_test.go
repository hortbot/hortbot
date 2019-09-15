package redis

import (
	"testing"

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
		got := buildKey(test.input...)
		assert.Equal(t, test.want, got)
	}
}

func TestBuildKeyPanic(t *testing.T) {
	var recovered interface{}

	func() {
		defer func() {
			recovered = recover()
		}()
		buildKey()
	}()

	assert.Assert(t, recovered != nil)

	recovered = nil

	func() {
		defer func() {
			recovered = recover()
		}()
		buildKey(keyStr("this").is("bad:value"))
	}()

	assert.Assert(t, recovered != nil)

}
