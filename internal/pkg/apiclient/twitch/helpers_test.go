package twitch

import (
	json "encoding/json"
	"errors"
	"testing"

	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestStatusToError(t *testing.T) {
	tests := map[int]error{
		200: nil,
		204: nil,
		400: ErrBadRequest,
		401: ErrNotAuthorized,
		403: ErrNotAuthorized,
		404: ErrNotFound,
		418: ErrUnknown,
		429: ErrUnknown,
		500: ErrServerError,
		503: ErrServerError,
	}

	for code, expected := range tests {
		err := statusToError(code)
		assert.Equal(t, err, expected)
	}
}

func TestSetToken(t *testing.T) {
	var newToken *oauth2.Token
	tok := &oauth2.Token{}

	setToken(&newToken)(tok, errors.New("something bad"))
	assert.Assert(t, newToken == nil)

	setToken(&newToken)(tok, nil)
	assert.Equal(t, newToken, tok)

	setToken(&newToken)(nil, nil)
	assert.Assert(t, newToken == nil)
}

func TestIDStrUnmarshalError(t *testing.T) {
	var id IDStr
	err := json.Unmarshal([]byte("true"), &id)
	assert.ErrorContains(t, err, "cannot unmarshal")
}
