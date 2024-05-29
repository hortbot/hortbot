package twitch_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestGetUserForToken(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	c := &twitch.Channel{
		ID:    1234,
		Name:  "someone",
		Title: "What a cool stream!",
		Game:  "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(int64(c.ID))

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	user, newToken, err := tw.GetUserByToken(ctx, tok)
	assert.NilError(t, err)
	assert.Equal(t, user.ID, c.ID)
	assert.Equal(t, user.Name, c.Name)
	assert.Assert(t, newToken == nil)
}

func TestGetUserForTokenServerError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	c := &twitch.Channel{
		ID:    503,
		Title: "What a cool stream!",
		Game:  "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(int64(c.ID))

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	_, _, err = tw.GetUserByToken(ctx, tok)
	assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/users: unexpected status: 503")
}

func TestGetUserForTokenDecodeError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	c := &twitch.Channel{
		ID:   777,
		Game: "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(int64(c.ID))

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	_, _, err = tw.GetUserByToken(ctx, tok)
	assert.Error(t, err, "twitch: ErrHandler: invalid character '}' looking for beginning of value")
}

func TestGetUserForTokenRequestError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	_, _, err := tw.GetUserByToken(ctx, &oauth2.Token{AccessToken: "requesterror"})
	assert.ErrorContains(t, err, errTestBadRequest.Error())
}

func TestGetUserForUsername(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	u, err := tw.GetUserByUsername(ctx, "foobar")
	assert.NilError(t, err)
	assert.DeepEqual(t, u, &twitch.User{
		ID:          1234,
		Name:        "foobar",
		DisplayName: "Foobar",
	})
}

func TestGetUserForUsernameServerError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	_, err := tw.GetUserByUsername(ctx, "servererror")
	assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/users?login=servererror: unexpected status: 500")
}

func TestGetUserForUsernameNotFound(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	_, err := tw.GetUserByUsername(ctx, "notfound")
	assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/users?login=notfound: unexpected status: 404")
}

func TestGetUserForUsernameNotFoundEmpty(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	_, err := tw.GetUserByUsername(ctx, "notfound2")
	assert.Error(t, err, "twitch: unexpected status: 404")
}

func TestGetUserForUsernameDecodeError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	_, err := tw.GetUserByUsername(ctx, "requesterror")
	assert.ErrorContains(t, err, errTestBadRequest.Error())
}

func TestGetUserForUsernameRequestError(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	_, err := tw.GetUserByUsername(ctx, "decodeerror")
	assert.Error(t, err, "twitch: ErrHandler: invalid character '}' looking for beginning of value")
}

func TestGetUserForID(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	u, err := tw.GetUserByID(ctx, 1234)
	assert.NilError(t, err)
	assert.DeepEqual(t, u, &twitch.User{
		ID:          1234,
		Name:        "foobar",
		DisplayName: "Foobar",
	})
}

func TestUserDispName(t *testing.T) {
	u := &twitch.User{
		Name: "somename",
	}

	assert.Equal(t, u.DispName(), u.Name)

	u.DisplayName = "SomeName"
	assert.Equal(t, u.DispName(), u.DisplayName)
}

func TestGetModeratedChannelsOK(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	const id = 123
	tok := tokFor(ctx, t, tw, ft, id)

	mods := []*twitch.ModeratedChannel{
		{
			ID:    1234,
			Name:  "Mod2",
			Login: "mod2",
		},
		{
			ID:    4141,
			Name:  "Mod1",
			Login: "mod1",
		},
		{
			ID:    999,
			Name:  "mod3",
			Login: "mod3",
		},
	}

	ft.setModerated(id, mods)

	got, newToken, err := tw.GetModeratedChannels(ctx, id, tok)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.DeepEqual(t, got, mods)
}

func TestGetModeratedChannelsErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	id := int64(777)
	tok := tokFor(ctx, t, tw, ft, id)
	ft.setModerated(id, []*twitch.ModeratedChannel{})

	_, _, err := tw.GetModeratedChannels(ctx, 777, tok)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)
		ft.setModerated(id, []*twitch.ModeratedChannel{})

		_, newToken, err := tw.GetModeratedChannels(ctx, id, tok)
		assert.ErrorContains(t, err, fmt.Sprintf("status: %d", status))
		assert.Assert(t, newToken == nil)
	}

	id = 888
	tok = tokFor(ctx, t, tw, ft, id)
	ft.setModerated(id, []*twitch.ModeratedChannel{})

	_, newToken, err := tw.GetModeratedChannels(ctx, id, tok)
	assert.Error(t, err, "twitch: ErrHandler: unexpected EOF")
	assert.Assert(t, newToken == nil)
}
