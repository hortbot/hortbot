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
		ID:          1234,
		DisplayName: "FooBar",
		Status:      "What a cool stream!",
		Game:        "Garry's Mod",
	}

	ft.setChannel(c)

	got, err := tw.GetChannelByID(ctx, c.ID.AsInt64())
	assert.NilError(t, err)
	assert.DeepEqual(t, c, got)
}

func TestGetChannelByIDErrors(t *testing.T) {
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

	_, err := tw.GetChannelByID(ctx, 999)
	assert.Equal(t, err, twitch.ErrNotFound)

	_, err = tw.GetChannelByID(ctx, 900)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	_, err = tw.GetChannelByID(ctx, 901)
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestSetChannelStatus(t *testing.T) {
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
		ID:          1234,
		DisplayName: "FooBar",
		Status:      "What a cool stream!",
		Game:        "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID.AsInt64())

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	const newStatus = "This is the new status."

	getStatus, newToken, err := tw.SetChannelStatus(ctx, c.ID.AsInt64(), tok, newStatus)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.Equal(t, getStatus, newStatus)

	got, err := tw.GetChannelByID(ctx, c.ID.AsInt64())
	assert.NilError(t, err)
	assert.Equal(t, got.Status, newStatus)
}

func TestSetChannelStatusNilToken(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	c := &twitch.Channel{
		ID:          1234,
		DisplayName: "FooBar",
		Status:      "What a cool stream!",
		Game:        "Garry's Mod",
	}

	ft.setChannel(c)

	_, newToken, err := tw.SetChannelStatus(ctx, c.ID.AsInt64(), nil, "something")
	assert.Equal(t, err, twitch.ErrNotAuthorized)
	assert.Assert(t, newToken == nil)
}

func TestSetChannelGame(t *testing.T) {
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
		ID:          1234,
		DisplayName: "FooBar",
		Status:      "What a cool stream!",
		Game:        "Garry's Mod",
	}

	ft.setChannel(c)

	code := ft.codeForUser(c.ID.AsInt64())

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	const newGame = "Just Chatting"

	setGame, newToken, err := tw.SetChannelGame(ctx, c.ID.AsInt64(), tok, newGame)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.Equal(t, setGame, newGame)

	got, err := tw.GetChannelByID(ctx, c.ID.AsInt64())
	assert.NilError(t, err)
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

	_, newToken, err := tw.SetChannelGame(ctx, c.ID.AsInt64(), nil, "something")
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
		tok := tokFor(ctx, t, tw, ft, id)

		_, newToken, err := tw.SetChannelStatus(ctx, id, tok, "something")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}

	id := int64(900)
	tok := tokFor(ctx, t, tw, ft, id)

	_, _, err := tw.SetChannelStatus(ctx, id, tok, "something")
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	id = 901
	tok = tokFor(ctx, t, tw, ft, id)

	_, _, err = tw.SetChannelStatus(ctx, id, tok, "something")
	assert.Equal(t, err, twitch.ErrServerError)
}

func TestSetChannelGameErrors(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)

		_, newToken, err := tw.SetChannelGame(ctx, id, tok, "something")
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}

	id := int64(900)
	tok := tokFor(ctx, t, tw, ft, id)

	_, _, err := tw.SetChannelGame(ctx, id, tok, "something")
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	id = 901
	tok = tokFor(ctx, t, tw, ft, id)

	_, _, err = tw.SetChannelGame(ctx, id, tok, "something")
	assert.Equal(t, err, twitch.ErrServerError)
}

func tokFor(ctx context.Context, t *testing.T, tw *twitch.Twitch, ft *fakeTwitch, id int64) *oauth2.Token {
	t.Helper()

	code := ft.codeForUser(id)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	return tok
}
