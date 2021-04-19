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

func TestSetChannelStatus(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

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

	got, err := twitch.GetChannelByID(tw, ctx, c.ID.AsInt64())
	assert.NilError(t, err)
	assert.Equal(t, got.Status, newStatus)
}

func TestSetChannelStatusNilToken(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

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
	ctx, cancel := testContext(t)
	defer cancel()

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

	got, err := twitch.GetChannelByID(tw, ctx, c.ID.AsInt64())
	assert.NilError(t, err)
	assert.Equal(t, got.Game, newGame)
}

func TestSetChannelGameNilToken(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

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
	ctx, cancel := testContext(t)
	defer cancel()

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
	ctx, cancel := testContext(t)
	defer cancel()

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

func TestGetChannelModeratorsOK(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const id = 123
	tok := tokFor(ctx, t, tw, ft, id)

	mods := []*twitch.ChannelModerator{
		{
			ID:   1234,
			Name: "mod2",
		},
		{
			ID:   4141,
			Name: "mod1",
		},
		{
			ID:   999,
			Name: "mod3",
		},
	}

	ft.setMods(id, mods)

	got, newToken, err := tw.GetChannelModerators(ctx, id, tok)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.DeepEqual(t, got, mods)
}

func TestGetChannelModeratorsErrors(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	id := int64(777)
	tok := tokFor(ctx, t, tw, ft, id)
	ft.setMods(id, []*twitch.ChannelModerator{})

	_, _, err := tw.GetChannelModerators(ctx, 777, tok)
	assert.ErrorContains(t, err, errTestBadRequest.Error())

	for status, expected := range expectedErrors {
		id := int64(status)
		tok := tokFor(ctx, t, tw, ft, id)
		ft.setMods(id, []*twitch.ChannelModerator{})

		_, newToken, err := tw.GetChannelModerators(ctx, id, tok)
		assert.Equal(t, err, expected, "%d", status)
		assert.Assert(t, newToken == nil)
	}

	id = 888
	tok = tokFor(ctx, t, tw, ft, id)
	ft.setMods(id, []*twitch.ChannelModerator{})

	_, newToken, err := tw.GetChannelModerators(ctx, id, tok)
	assert.Equal(t, err, twitch.ErrServerError)
	assert.Assert(t, newToken == nil)
}

func TestGetChannelModeratorsEsoteric(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	const id = 999
	tok := tokFor(ctx, t, tw, ft, id)

	mods := []*twitch.ChannelModerator{
		{
			ID:   1234,
			Name: "mod2",
		},
		{
			ID:   4141,
			Name: "mod1",
		},
		{
			ID:   999,
			Name: "mod3",
		},
	}

	ft.setMods(id, mods)

	got, newToken, err := tw.GetChannelModerators(ctx, id, tok)
	assert.NilError(t, err)
	assert.Assert(t, newToken == nil)
	assert.DeepEqual(t, got, mods)
}

func TestModifyChannel(t *testing.T) {
	t.Run("Success title", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)

		const id = 1234
		tok := tokFor(ctx, t, tw, ft, id)
		newToken, err := tw.ModifyChannel(ctx, id, tok, strPtr("some new title"), nil)
		assert.NilError(t, err)
		assert.Assert(t, newToken == nil)
	})

	t.Run("Success game", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		const id = 5678
		tok := tokFor(ctx, t, tw, ft, id)
		newToken, err := tw.ModifyChannel(ctx, id, tok, nil, int64Ptr(9876))
		assert.NilError(t, err)
		assert.Assert(t, newToken == nil)
	})

	t.Run("Server error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		const id = 500
		tok := tokFor(ctx, t, tw, ft, id)
		_, err := tw.ModifyChannel(ctx, id, tok, strPtr("some new title"), nil)
		assert.Equal(t, err, twitch.ErrServerError)
	})

	t.Run("Request error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		const id = 900
		tok := tokFor(ctx, t, tw, ft, id)
		_, err := tw.ModifyChannel(ctx, id, tok, strPtr("some new title"), nil)
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})

	t.Run("Nil token", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, tw := createTester(t)
		const id = 900
		_, err := tw.ModifyChannel(ctx, id, nil, strPtr("some new title"), nil)
		assert.Equal(t, err, twitch.ErrNotAuthorized)
	})

	t.Run("Bad request", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		const id = 900
		tok := tokFor(ctx, t, tw, ft, id)
		_, err := tw.ModifyChannel(ctx, id, tok, nil, nil)
		assert.Equal(t, err, twitch.ErrBadRequest)

		_, err = tw.ModifyChannel(ctx, id, tok, strPtr(""), nil)
		assert.Equal(t, err, twitch.ErrBadRequest)
	})
}

func tokFor(ctx context.Context, t *testing.T, tw *twitch.Twitch, ft *fakeTwitch, id int64) *oauth2.Token { //nolint:thelper
	t.Helper()

	code := ft.codeForUser(id)

	tok, err := tw.Exchange(ctx, code)
	assert.NilError(t, err)
	assert.DeepEqual(t, tok, ft.tokenForCode(code), tokenCmp)

	return tok
}

func TestGetHelixChannelByID(t *testing.T) {
	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))

	t.Run("Success", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		channel, err := tw.GetHelixChannelByID(ctx, 1234)
		assert.NilError(t, err)

		assert.DeepEqual(t, channel, &twitch.HelixChannel{
			ID:     1234,
			Name:   "foobar",
			Game:   "PUBG MOBILE",
			GameID: 58730284,
			Title:  "This is the title.",
		})
	})

	t.Run("Empty", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetHelixChannelByID(ctx, 444)
		assert.Equal(t, err, twitch.ErrNotFound)
	})

	t.Run("Empty 404", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetHelixChannelByID(ctx, 404)
		assert.Equal(t, err, twitch.ErrNotFound)
	})

	t.Run("Server error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetHelixChannelByID(ctx, 500)
		assert.Equal(t, err, twitch.ErrServerError)
	})

	t.Run("Decode error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetHelixChannelByID(ctx, 900)
		assert.Equal(t, err, twitch.ErrServerError)
	})

	t.Run("Request error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetHelixChannelByID(ctx, 901)
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})
}
