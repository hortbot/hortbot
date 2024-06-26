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
	t.Parallel()
	ft := newFakeTwitch(t)
	cli := ft.client()

	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	ft.setClientTokens(tok)

	tw := twitch.New(clientID, clientSecret, redirectURL, cli)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		want := []twitch.GameLink{
			{Type: twitch.GameLinkSteam, URL: "https://store.steampowered.com/app/1119980"},
			{Type: twitch.GameLinkEpic, URL: "https://www.epicgames.com/store/p/in-sound-mind"},
			{Type: twitch.GameLinkGOG, URL: "https://www.gog.com/game/in_sound_mind"},
			{Type: twitch.GameLinkOfficial, URL: "https://modusgames.com/in-sound-mind"},
		}

		links, err := tw.GetGameLinks(ctx, 518088)
		assert.NilError(t, err)

		assert.DeepEqual(t, links, want)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 4040)
		assert.Error(t, err, "twitch: unexpected status: 404")
	})

	t.Run("Empty 404", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 404)
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.igdb.com/v4/games: unexpected status: 404")
	})

	t.Run("Empty 404 1", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 4041)
		assert.Error(t, err, "twitch: unexpected status: 404")
	})

	t.Run("Empty 404 2", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 777)
		assert.Error(t, err, "twitch: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 500)
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.igdb.com/v4/games: unexpected status: 500")
	})

	t.Run("Decode error", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 700)
		assert.Error(t, err, "twitch: ErrHandler: unexpected EOF")
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.GetGameLinks(ctx, 599)
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})
}
