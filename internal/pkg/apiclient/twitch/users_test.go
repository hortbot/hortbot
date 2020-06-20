package twitch_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestGetUserForToken(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     1234,
		Name:   "someone",
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID.AsInt64())

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	user, newToken, err := tw.GetUserForToken(ctx, tok)
	assert.NilError(t, err)
	assert.Equal(t, user.ID, c.ID)
	assert.Equal(t, user.Name, c.Name)
	assert.Assert(t, newToken == nil)
}

func TestGetUserForTokenServerError(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     503,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID.AsInt64())

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	_, _, err = tw.GetUserForToken(ctx, tok)
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestGetUserForTokenDecodeError(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     777,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID.AsInt64())

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	_, _, err = tw.GetUserForToken(ctx, tok)
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestGetUserForTokenRequestError(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, _, err := tw.GetUserForToken(ctx, &oauth2.Token{AccessToken: "requesterror"})
	assert.ErrorContains(t, err, errTestBadRequest.Error())
}

func TestGetUserForUsername(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	u, err := tw.GetUserForUsername(ctx, "foobar")
	assert.NilError(t, err)
	assert.DeepEqual(t, u, &twitch.User{
		ID:          1234,
		Name:        "foobar",
		DisplayName: "Foobar",
	})
}

func TestGetUserForUsernameServerError(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, err := tw.GetUserForUsername(ctx, "servererror")
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestGetUserForUsernameNotFound(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, err := tw.GetUserForUsername(ctx, "notfound")
	assert.Equal(t, err, twitch.ErrNotFound)
}

func TestGetUserForUsernameNotFoundEmpty(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, err := tw.GetUserForUsername(ctx, "notfound2")
	assert.Equal(t, err, twitch.ErrNotFound)
}

func TestGetUserForUsernameDecodeError(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, err := tw.GetUserForUsername(ctx, "requesterror")
	assert.ErrorContains(t, err, errTestBadRequest.Error())
}

func TestGetUserForUsernameRequestError(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	_, err := tw.GetUserForUsername(ctx, "decodeerror")
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestFollowChannel(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID: 1234,
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID.AsInt64())

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	newToken, err := tw.FollowChannel(ctx, c.ID.AsInt64(), tok, 200)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)

	newToken, err = tw.FollowChannel(ctx, c.ID.AsInt64(), tok, 401)
	assert.Equal(t, err, twitch.ErrNotAuthorized)
	assert.Assert(t, newToken == nil)

	newToken, err = tw.FollowChannel(ctx, c.ID.AsInt64(), tok, 404)
	assert.Equal(t, err, twitch.ErrNotFound)
	assert.Assert(t, newToken == nil)

	newToken, err = tw.FollowChannel(ctx, c.ID.AsInt64(), tok, 422)
	assert.Equal(t, err, twitch.ErrUnknown)
	assert.Assert(t, newToken == nil)

	newToken, err = tw.FollowChannel(ctx, c.ID.AsInt64(), tok, 500)
	assert.Equal(t, err, twitch.ErrServerError)
	assert.Assert(t, newToken == nil)

	newToken, err = tw.FollowChannel(ctx, c.ID.AsInt64(), nil, 500)
	assert.Equal(t, err, twitch.ErrNotAuthorized)
	assert.Assert(t, newToken == nil)

	newToken, err = tw.FollowChannel(ctx, c.ID.AsInt64(), tok, 901)
	assert.ErrorContains(t, err, errTestBadRequest.Error())
	assert.Assert(t, newToken == nil)
}

func TestUserDispName(t *testing.T) {
	u := &twitch.User{
		Name: "somename",
	}

	assert.Equal(t, u.DispName(), u.Name)

	u.DisplayName = "SomeName"
	assert.Equal(t, u.DispName(), u.DisplayName)
}
