// Package jsonx extends the encoding/json API.
package jsonx

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
)

// ErrMoreThanOne is returned when DecodeOne decodes more than a single value.
var ErrMoreThanOne = errors.New("jsonx: more than one value")

// DecodeSingle decodes a single JSON value, returning an error if the reader
// contains more than that single value. See golang.org/issues/36225.
func DecodeSingle(r io.Reader, v any) error {
	d := jsontext.NewDecoder(r)
	if err := json.UnmarshalDecode(d, v); err != nil {
		return err //nolint:wrapcheck
	}
	if _, err := d.ReadValue(); err != io.EOF { //nolint:errorlint
		return ErrMoreThanOne
	}
	return nil
}

// ErrUnmarshallable is returned by the unmarshalable type's MarshalJSON function.
var ErrUnmarshallable = errors.New("unmarshallable")

// Unmarshallable returns a json.Marshaler which always returns an error.
func Unmarshallable() json.Marshaler {
	return &unmarshallable{}
}

type unmarshallable struct{}

func (*unmarshallable) MarshalJSON() ([]byte, error) {
	return nil, ErrUnmarshallable
}
