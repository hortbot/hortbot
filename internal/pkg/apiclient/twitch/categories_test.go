package twitch_test

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
)

func TestSearchCategories(t *testing.T) {
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
		ctx, cancel := testContext(t)
		defer cancel()

		cats, err := tw.SearchCategories(ctx, "pubg")
		assert.NilError(t, err)

		assert.DeepEqual(t, cats, []*twitch.Category{
			{ID: 287491, Name: "PLAYERUNKNOWN's BATTLEGROUNDS"},
			{ID: 58730284, Name: "PUBG MOBILE"},
		})
	})

	t.Run("Empty", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.SearchCategories(ctx, "notfound")
		assert.Error(t, err, "twitch: unexpected status: 404")
	})

	t.Run("Empty 404", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.SearchCategories(ctx, "notfound2")
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/search/categories?query=notfound2: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.SearchCategories(ctx, "servererror")
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/search/categories?query=servererror: unexpected status: 500")
	})

	t.Run("Decode error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.SearchCategories(ctx, "decodeerror")
		assert.Error(t, err, "twitch: ErrHandler: invalid character '}' looking for beginning of value")
	})

	t.Run("Request error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		_, err := tw.SearchCategories(ctx, "requesterror")
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})
}

func TestGetGame(t *testing.T) {
	tok := &oauth2.Token{
		AccessToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:      time.Now().Add(time.Hour).Round(time.Second),
		TokenType:   "bearer",
	}

	t.Run("Success name", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		ft.setClientTokens(tok)

		game, err := tw.GetGameByName(ctx, "PLAYERUNKNOWN's BATTLEGROUNDS")
		assert.NilError(t, err)

		assert.DeepEqual(t, game, &twitch.Category{ID: 287491, Name: "PLAYERUNKNOWN's BATTLEGROUNDS"})
	})

	t.Run("Success name", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		ft.setClientTokens(tok)

		game, err := tw.GetGameByID(ctx, 287491)
		assert.NilError(t, err)

		assert.DeepEqual(t, game, &twitch.Category{ID: 287491, Name: "PLAYERUNKNOWN's BATTLEGROUNDS"})
	})

	t.Run("Not found", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		ft.setClientTokens(tok)

		_, err := tw.GetGameByName(ctx, "notfound")
		assert.Error(t, err, "twitch: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		ft.setClientTokens(tok)

		_, err := tw.GetGameByName(ctx, "servererror")
		assert.Error(t, err, "twitch: ErrValidator: response error for https://api.twitch.tv/helix/games?name=servererror: unexpected status: 500")
	})

	t.Run("Decode error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		ft.setClientTokens(tok)

		_, err := tw.GetGameByName(ctx, "decodeerror")
		assert.Error(t, err, "twitch: ErrHandler: invalid character '}' looking for beginning of value")
	})

	t.Run("Request error", func(t *testing.T) {
		ctx, cancel := testContext(t)
		defer cancel()

		ft, tw := createTester(t)
		ft.setClientTokens(tok)

		_, err := tw.GetGameByName(ctx, "requesterror")
		assert.ErrorContains(t, err, errTestBadRequest.Error())
	})
}
