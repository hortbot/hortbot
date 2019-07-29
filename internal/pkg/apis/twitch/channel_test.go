package twitch_test

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/assert"
)

func TestGetChannelByID(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	c := &twitch.Channel{
		ID:     1234,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	_, err := tw.GetChannelByID(ctx, 999)
	assert.Equal(t, err, twitch.ErrNotFound)

	got, err := tw.GetChannelByID(ctx, c.ID)
	assert.NilError(t, err)
	assert.DeepEqual(t, c, got)
}

func TestGetChannel(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     1234,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	got, newToken, err := tw.GetChannel(ctx, tok)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.DeepEqual(t, c, got)
}

func TestSetChannelStatus(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     1234,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	const newStatus = "This is the new status."

	newToken, err := tw.SetChannelStatus(ctx, c.ID, tok, newStatus)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)

	got, newToken, err := tw.GetChannel(ctx, tok)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.Equal(t, got.Status, newStatus)
}

func TestSetChannelStatusNilToken(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     1234,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	newToken, err := tw.SetChannelStatus(ctx, c.ID, nil, "something")
	assert.Equal(t, err, twitch.ErrNotAuthorized)
	assert.Assert(t, newToken == nil)
}

func TestSetChannelGame(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     1234,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	const newGame = "Just Chatting"

	newToken, err := tw.SetChannelGame(ctx, c.ID, tok, newGame)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)

	got, newToken, err := tw.GetChannel(ctx, tok)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.Equal(t, got.Game, newGame)
}

func TestSetChannelGameNilToken(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:     1234,
		Status: "What a cool stream!",
		Game:   "Garry's Mod",
	}

	ft.setChannel(c)

	newToken, err := tw.SetChannelGame(ctx, c.ID, nil, "something")
	assert.Equal(t, err, twitch.ErrNotAuthorized)
	assert.Assert(t, newToken == nil)
}

func TestSetChannelStatusErrors(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	for status, expected := range expectedErrors {
		id := int64(status)

		code := ft.codeForUser(id)

		tok, err := tw.Exchange(ctx, code)
		assert.NilError(t, err)
		assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

		newToken, err := tw.SetChannelStatus(ctx, id, tok, "something")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}

func TestSetChannelGameErrors(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	for status, expected := range expectedErrors {
		id := int64(status)

		code := ft.codeForUser(id)

		tok, err := tw.Exchange(ctx, code)
		assert.NilError(t, err)
		assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

		newToken, err := tw.SetChannelGame(ctx, id, tok, "something")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}
}
