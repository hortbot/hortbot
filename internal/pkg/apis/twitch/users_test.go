package twitch_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
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
	assert.Equal(t, user.ID, c.ID.AsInt64())
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

func TestGetIDForUsername(t *testing.T) {
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

	id, err := tw.GetIDForUsername(ctx, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, id, int64(1234))
}

func TestGetIDForUsernameServerError(t *testing.T) {
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

	_, err := tw.GetIDForUsername(ctx, "servererror")
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestGetIDForUsernameNotFound(t *testing.T) {
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

	_, err := tw.GetIDForUsername(ctx, "notfound")
	assert.Equal(t, err, twitch.ErrNotFound)
}

func TestGetIDForUsernameNotFoundEmpty(t *testing.T) {
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

	_, err := tw.GetIDForUsername(ctx, "notfound2")
	assert.Equal(t, err, twitch.ErrNotFound)
}

func TestGetIDForUsernameDecodeError(t *testing.T) {
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

	_, err := tw.GetIDForUsername(ctx, "decodeerror")
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
}
