package redis

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestBuildKey(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{
			input: []string{"first"},
			want:  "first",
		},
		{
			input: []string{"first", "second"},
			want:  "first:second",
		},
		{
			input: []string{"first", "second", "third", "ok"},
			want:  "first:second:third:ok",
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
		buildKey("this", "is:bad")
	}()

	assert.Assert(t, recovered != nil)

}
