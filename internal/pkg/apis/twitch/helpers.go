package twitch

import (
	"encoding/json"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
)

func statusToError(code int) error {
	if code >= 200 && code < 300 {
		return nil
	}

	switch code {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrNotAuthorized
	}

	if code >= 500 {
		return ErrServerError
	}

	return ErrUnknown
}

// Go 1.13's http.Header.Clone.
func cloneHeader(h http.Header) http.Header {
	// Find total number of values.
	nv := 0
	for _, vv := range h {
		nv += len(vv)
	}
	sv := make([]string, nv) // shared backing array for headers' values
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		n := copy(sv, vv)
		h2[k] = sv[:n:n]
		sv = sv[n:]
	}
	return h2
}

func setToken(newToken **oauth2.Token) func(tok *oauth2.Token, err error) {
	return func(tok *oauth2.Token, err error) {
		if err == nil {
			*newToken = tok
		}
	}
}

// IDStr is an int64 that is represented as a string in JSON, but can be
// parsed as either a string or a raw integer.
//
// https://stackoverflow.com/a/31625512
type IDStr int64

func (v IDStr) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(v), 10))
}

func (v *IDStr) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}

	var tmp int64
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	*v = IDStr(tmp)
	return nil
}

func (v IDStr) AsInt64() int64 {
	return int64(v)
}