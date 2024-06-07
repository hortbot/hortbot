package twitch

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestSetToken(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	var id idstr.IDStr
	err := json.Unmarshal([]byte("true"), &id)
	assert.ErrorContains(t, err, "cannot unmarshal")
}
