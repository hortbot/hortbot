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

func readerFor(v any) io.Reader {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		panic(err)
	}
	return &buf
}

func TestDecodeSingleOK(t *testing.T) {
	t.Parallel()
	want := map[string]any{
		"foo": "bar",
		"x":   true,
	}

	var got map[string]any
	err := jsonx.DecodeSingle(readerFor(want), &got)
	assert.NilError(t, err)
	assert.DeepEqual(t, want, got)
}

func TestDecodeSingleTwo(t *testing.T) {
	t.Parallel()
	var got any
	err := jsonx.DecodeSingle(strings.NewReader("{}{}"), &got)
	assert.Equal(t, err, jsonx.ErrMoreThanOne)
}

func TestDecodeSingleIncomplete(t *testing.T) {
	t.Parallel()
	var got any
	err := jsonx.DecodeSingle(strings.NewReader("{"), &got)
	assert.ErrorContains(t, err, "unexpected EOF")
}

func TestUnmarshallable(t *testing.T) {
	t.Parallel()
	v := jsonx.Unmarshallable()

	_, err := v.MarshalJSON()
	assert.Equal(t, err, jsonx.ErrUnmarshallable)

	_, err = json.Marshal(v)
	assert.ErrorContains(t, err, jsonx.ErrUnmarshallable.Error())
}
