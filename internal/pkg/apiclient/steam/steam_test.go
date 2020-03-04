package steam_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	assertx.Panic(t, func() {
		steam.New("")
	}, "empty apiKey")
}

func TestGetPlayerSummary(t *testing.T) {
	const (
		apiKey = "THISISTHEAPIKEY123456789"
		id     = "12730127017230123"
		url    = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/"
		query  = "key=" + apiKey + "&format=json&steamids=" + id
	)

	ctx := context.Background()

	t.Run("Good", func(t *testing.T) {
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

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

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
		const response = `{
			"response": {
				"players": []
			}
		}`

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, response))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 404})
	})

	t.Run("Bad response", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, "{"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("Not found", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(404, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 404})
	})

	t.Run("Server error", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(500, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 500})
	})

	t.Run("Not authorized", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(403, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 403})
	})

	t.Run("Unknown", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(418, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 418})
	})

	t.Run("Request error", func(t *testing.T) {
		testErr := errors.New("testing error")
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewErrorResponder(testErr))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetPlayerSummary(ctx, id)
		assert.ErrorContains(t, err, testErr.Error())
	})
}

func TestGetOwnedGames(t *testing.T) {
	const (
		apiKey = "THISISTHEAPIKEY123456789"
		id     = "12730127017230123"
		url    = "https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/"
		query  = "key=" + apiKey + "&format=json&steamid=" + id + "&include_appinfo=1"
	)

	ctx := context.Background()

	t.Run("Good", func(t *testing.T) {
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

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

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
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, "{"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetOwnedGames(ctx, id)
		assert.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("Not found", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(404, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetOwnedGames(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 404})
	})

	t.Run("Server error", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(500, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetOwnedGames(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 500})
	})

	t.Run("Not authorized", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(403, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetOwnedGames(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 403})
	})

	t.Run("Unknown", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(418, "{}"))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetOwnedGames(ctx, id)
		assert.DeepEqual(t, err, &apiclient.Error{API: "steam", StatusCode: 418})
	})

	t.Run("Request error", func(t *testing.T) {
		testErr := errors.New("testing error")
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewErrorResponder(testErr))

		s := steam.New(apiKey, steam.HTTPClient(&http.Client{Transport: mt}))

		_, err := s.GetOwnedGames(ctx, id)
		assert.ErrorContains(t, err, testErr.Error())
	})
}
