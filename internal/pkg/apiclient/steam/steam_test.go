package steam_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	assertx.Panic(t, func() {
		steam.New("", nil)
	}, "empty apiKey")
}

func TestGetPlayerSummary(t *testing.T) {
	t.Parallel()
	const (
		apiKey = "THISISTHEAPIKEY123456789"
		id     = "12730127017230123"
		url    = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
		query  = "key=" + apiKey + "&format=json&steamids=" + id
	)

	ctx := t.Context()

	t.Run("Good", func(t *testing.T) {
		t.Parallel()
		const response = `{
			"response": {
				"players": [
					{
						"personaname": "Steam User",
						"profileurl": "https://steamcommunity.com/id/fakeprofile",
						"gameextrainfo": "Project Winter",
						"gameid": "774861",
						"gameserverip": "127.0.0.1:8080"
					}
				]
			}
		}`

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, response))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		summary, err := s.GetPlayerSummary(ctx, id)
		assert.NilError(t, err)

		assert.DeepEqual(t, summary, &steam.Summary{
			Name:       "Steam User",
			ProfileURL: "https://steamcommunity.com/id/fakeprofile",
			Game:       "Project Winter",
			GameID:     "774861",
			GameServer: "127.0.0.1:8080",
		})
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		const response = `{
			"response": {
				"players": []
			}
		}`

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, response))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.Error(t, err, "steam: unexpected status: 404")
	})

	t.Run("Bad response", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, "{"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(404, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?format=json&key=REDACTED0&steamids=12730127017230123: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(500, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?format=json&key=REDACTED0&steamids=12730127017230123: unexpected status: 500")
	})

	t.Run("Not authorized", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(403, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?format=json&key=REDACTED0&steamids=12730127017230123: unexpected status: 403")
	})

	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(418, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?format=json&key=REDACTED0&steamids=12730127017230123: unexpected status: 418")
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("testing error")
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewErrorResponder(testErr))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetPlayerSummary(ctx, id)
		assert.ErrorContains(t, err, testErr.Error())
	})
}

func TestGetOwnedGames(t *testing.T) {
	t.Parallel()
	const (
		apiKey = "THISISTHEAPIKEY123456789"
		id     = "12730127017230123"
		url    = "https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/"
		query  = "key=" + apiKey + "&format=json&steamid=" + id + "&include_appinfo=1"
	)

	ctx := t.Context()

	t.Run("Good", func(t *testing.T) {
		t.Parallel()
		const response = `{
			"response": {
				"games": [
					{
						"appid": 10,
						"name": "Counter-Strike"
					},
					{
						"appid": 220,
						"name": "Half-Life 2"
					}
				]
			}
		}`

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, response))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		games, err := s.GetOwnedGames(ctx, id)
		assert.NilError(t, err)

		assert.DeepEqual(t, games, []*steam.Game{
			{
				ID:   10,
				Name: "Counter-Strike",
			},
			{
				ID:   220,
				Name: "Half-Life 2",
			},
		})
	})

	t.Run("Bad response", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, "{"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetOwnedGames(ctx, id)
		assert.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(404, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetOwnedGames(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?format=json&include_appinfo=1&key=REDACTED0&steamid=12730127017230123: unexpected status: 404")
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(500, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetOwnedGames(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?format=json&include_appinfo=1&key=REDACTED0&steamid=12730127017230123: unexpected status: 500")
	})

	t.Run("Not authorized", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(403, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetOwnedGames(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?format=json&include_appinfo=1&key=REDACTED0&steamid=12730127017230123: unexpected status: 403")
	})

	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(418, "{}"))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetOwnedGames(ctx, id)
		assert.Error(t, err, "steam: ErrValidator: response error for https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?format=json&include_appinfo=1&key=REDACTED0&steamid=12730127017230123: unexpected status: 418")
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("testing error")
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewErrorResponder(testErr))

		s := steam.New(apiKey, &http.Client{Transport: mt})

		_, err := s.GetOwnedGames(ctx, id)
		assert.ErrorContains(t, err, testErr.Error())
	})
}
