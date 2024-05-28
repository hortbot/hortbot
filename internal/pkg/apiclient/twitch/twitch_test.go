package twitch_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

var (
	tokenCmp = cmp.Comparer(func(x, y oauth2.Token) bool {
		return oauth2x.Equals(&x, &y)
	})

	expectedErrors = map[int]error{
		401: apiclient.NewStatusError("twitch", 401),
		404: apiclient.NewStatusError("twitch", 404),
		418: apiclient.NewStatusError("twitch", 418),
		500: apiclient.NewStatusError("twitch", 500),
	}
)

const (
	clientID     = "client-id"
	clientSecret = "client-secret"
	redirectURL  = "http://localhost/auth/twitch/callback"
)

func TestNewPanic(t *testing.T) {
	checkPanic := func(fn func()) any {
		var recovered any

		func() {
			defer func() {
				recovered = recover()
			}()

			fn()
		}()

		return recovered
	}

	assert.Equal(t, checkPanic(func() {
		twitch.New(clientID, clientSecret, redirectURL, http.DefaultClient)
	}), nil)

	assert.Equal(t, checkPanic(func() {
		twitch.New("", clientSecret, redirectURL, http.DefaultClient)
	}), "empty clientID")

	assert.Equal(t, checkPanic(func() {
		twitch.New(clientID, "", redirectURL, http.DefaultClient)
	}), "empty clientSecret")

	assert.Equal(t, checkPanic(func() {
		twitch.New(clientID, clientSecret, "", http.DefaultClient)
	}), "empty redirectURL")

	assert.Equal(t, checkPanic(func() {
		twitch.New(clientID, clientSecret, redirectURL, nil)
	}), "nil http.Client")
}

func TestAuthExchange(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	const state = "some-state"

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	assert.Equal(t,
		tw.AuthCodeURL(state, nil),
		fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?access_type=offline&client_id=%s&force_verify=true&redirect_uri=%s&response_type=code&state=%s", clientID, url.QueryEscape(redirectURL), state),
	)

	code := ft.codeForUser(1234)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	assert.Equal(t,
		tw.AuthCodeURL(state, []string{"user_follows_edit"}),
		fmt.Sprintf("https://id.twitch.tv/oauth2/authorize?access_type=offline&client_id=%s&force_verify=true&redirect_uri=%s&response_type=code&scope=user_follows_edit&state=%s", clientID, url.QueryEscape(redirectURL), state),
	)
}
