package idstr

import (
	"encoding/json"
	"strconv"
)

// IDStr is an int64 that is represented as a string in JSON, but can be
// parsed as either a string or a raw integer.
//
// https://stackoverflow.com/a/31625512
type IDStr int64

// MarshalJSON implements json.Marshaler for IDStr.
func (v IDStr) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(v), 10)) //nolint:wrapcheck
}

// UnmarshalJSON implements json.Unmarshaler for IDStr.
func (v *IDStr) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	return json.Unmarshal(data, (*int64)(v)) //nolint:wrapcheck
}
