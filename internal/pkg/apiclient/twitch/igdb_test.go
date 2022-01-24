package twitch_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestGetGameLinks(t *testing.T) {
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

		want := []twitch.GameLink{
			{Type: twitch.GameLinkSteam, URL: "https://store.steampowered.com/app/1119980"},
			{Type: twitch.GameLinkEpic, URL: "https://www.epicgames.com/store/p/in-sound-mind"},
			{Type: twitch.GameLinkGOG, URL: "https://www.gog.com/game/in_sound_mind"},
		}

		links, err := tw.GetGameLinks(ctx, 518088)
		assert.NilError(t, err)

		assert.DeepEqual(t, links, want)
	})

	t.Run("Empty", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 4040)
		assert.Equal(t, err, twitch.ErrNotFound)
	})

	t.Run("Empty 404", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 404)
		assert.Equal(t, err, twitch.ErrNotFound)
	})

	t.Run("Empty 404 2", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 4041)
		assert.Equal(t, err, twitch.ErrNotFound)
	})

	t.Run("Empty 404 2", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 777)
		assert.Equal(t, err, twitch.ErrNotFound)
	})

	t.Run("Server error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 500)
		assert.Equal(t, err, twitch.ErrServerError)
	})

	t.Run("Decode error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 700)
		assert.Equal(t, err, twitch.ErrServerError)
	})

	t.Run("Request error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 599)
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})
}
