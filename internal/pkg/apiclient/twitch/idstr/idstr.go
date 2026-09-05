package idstr

import (
	"encoding/json/jsontext"
	"fmt"
	"strconv"
)

// IDStr is an int64 that is represented as a string in JSON, but can be
// parsed as either a string or a raw integer.
//
// https://stackoverflow.com/a/31625512
type IDStr int64 //nolint:recvcheck // JSON encoding uses a value receiver while decoding must mutate a pointer.

// MarshalJSONTo implements json.MarshalerTo for IDStr.
func (v IDStr) MarshalJSONTo(out *jsontext.Encoder) error {
	return out.WriteToken(jsontext.String(strconv.FormatInt(int64(v), 10))) //nolint:wrapcheck
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for IDStr.
func (v *IDStr) UnmarshalJSONFrom(in *jsontext.Decoder) error {
	token, err := in.ReadToken()
	if err != nil {
		return err //nolint:wrapcheck
	}
	if token.Kind() == jsontext.KindNull {
		return nil
	}
	if token.Kind() != jsontext.KindString && token.Kind() != jsontext.KindNumber {
		return fmt.Errorf("cannot unmarshal %s into IDStr", token.Kind())
	}
	value, err := strconv.ParseInt(token.String(), 10, 64)
	if err != nil {
		return fmt.Errorf("parse IDStr: %w", err)
	}
	*v = IDStr(value)
	return nil
}
