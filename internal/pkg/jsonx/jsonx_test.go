package jsonx_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"gotest.tools/v3/assert"
)

func readerFor(v interface{}) io.Reader {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		panic(err)
	}
	return &buf
}

func TestDecodeSingleOK(t *testing.T) {
	want := map[string]interface{}{
		"foo": "bar",
		"x":   true,
	}

	var got map[string]interface{}
	err := jsonx.DecodeSingle(readerFor(want), &got)
	assert.NilError(t, err)
	assert.DeepEqual(t, want, got)
}

func TestDecodeSingleTwo(t *testing.T) {
	var got interface{}
	err := jsonx.DecodeSingle(strings.NewReader("{}{}"), &got)
	assert.Equal(t, err, jsonx.ErrMoreThanOne)
}

func TestDecodeSingleIncomplete(t *testing.T) {
	var got interface{}
	err := jsonx.DecodeSingle(strings.NewReader("{"), &got)
	assert.ErrorContains(t, err, "unexpected EOF")
}
