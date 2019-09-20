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

func TestCachedGetUserForUsername(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.NewCached(
		twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli)),
	)

	u, err := tw.GetUserForUsername(ctx, "foobar")
	assert.NilError(t, err)
	assert.DeepEqual(t, u, &twitch.User{
		ID:          1234,
		Name:        "foobar",
		DisplayName: "Foobar",
	})

	u2, err2 := tw.GetUserForUsername(ctx, "foobar")
	assert.NilError(t, err2)
	assert.Equal(t, u, u2)
}

func TestCachedGetChannelByID(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.NewCached(
		twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli)),
	)

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

	got, err := tw.GetChannelByID(ctx, c.ID.AsInt64())
	assert.NilError(t, err)
	assert.DeepEqual(t, c, got)

	got2, err2 := tw.GetChannelByID(ctx, c.ID.AsInt64())
	assert.Equal(t, err2, err)
	assert.Equal(t, got2, got)

	tw.Flush()

	got2, err2 = tw.GetChannelByID(ctx, c.ID.AsInt64())
	assert.Equal(t, err2, err)
	assert.Assert(t, got2 != got)
	assert.DeepEqual(t, got2, got)
}

func TestCachedGetCurrentStream(t *testing.T) {
	ctx := context.Background()

	ft := newFakeTwitch(t)
	cli := ft.client()

	tw := twitch.NewCached(
		twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli)),
	)

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	id := int64(1234)

	s := &twitch.Stream{
		ID:        12345678,
		Game:      "Garry's Mod",
		Viewers:   311,
		CreatedAt: time.Now().Add(-time.Hour).Round(time.Second),
	}

	ft.setStream(id, s)

	got, err := tw.GetCurrentStream(ctx, id)
	assert.NilError(t, err)
	assert.DeepEqual(t, got, s)

	got2, err2 := tw.GetCurrentStream(ctx, id)
	assert.Equal(t, err2, err)
	assert.Equal(t, got2, got)

	tw.Flush()

	got2, err2 = tw.GetCurrentStream(ctx, id)
	assert.Equal(t, err2, err)
	assert.Assert(t, got2 != got)
	assert.DeepEqual(t, got2, got)
}
