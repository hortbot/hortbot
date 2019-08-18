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

func TestCachedGetIDForUsername(t *testing.T) {
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

	id, err := tw.GetIDForUsername(ctx, "foobar")
	assert.NilError(t, err)
	assert.Equal(t, id, int64(1234))

	id2, err2 := tw.GetIDForUsername(ctx, "foobar")
	assert.NilError(t, err2)
	assert.Equal(t, id2, int64(1234))
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
