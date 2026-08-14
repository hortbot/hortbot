package hltb_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

const (
	initURL   = `=~^https://howlongtobeat\.com/api/bleed/init\?t=[0-9]+$`
	searchURL = "https://howlongtobeat.com/api/bleed"
)

func TestSearchGame(t *testing.T) {
	t.Parallel()

	mt := httpmockx.NewMockTransport(t)
	var userAgent string
	mt.RegisterResponder("GET", initURL, httpmockx.ResponderFunc(func(req *http.Request) (*http.Response, error) {
		userAgent = req.Header.Get("User-Agent")
		assert.Assert(t, len(userAgent) > 0)
		return httpmock.NewJsonResponse(http.StatusOK, map[string]string{
			"token": "token",
			"hpKey": "challenge",
			"hpVal": "answer",
		})
	}))
	mt.RegisterResponder("POST", searchURL, httpmockx.ResponderFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, req.Header.Get("User-Agent"), userAgent)
		assert.Equal(t, req.Header.Get("X-Auth-Token"), "token")
		assert.Equal(t, req.Header.Get("X-Hp-Key"), "challenge")
		assert.Equal(t, req.Header.Get("X-Hp-Val"), "answer")

		var body map[string]json.RawMessage
		assert.NilError(t, json.NewDecoder(req.Body).Decode(&body))
		assert.Equal(t, string(body["challenge"]), `"answer"`)
		assert.Equal(t, string(body["searchTerms"]), `["Half-Life","Alyx"]`)
		assert.Equal(t, string(body["size"]), `1`)

		return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
			"data": []map[string]any{{
				"game_id":   72067,
				"game_name": "Half-Life: Alyx",
				"comp_main": 43950,
				"comp_plus": 51647,
				"comp_100":  68214,
			}},
		})
	}))

	client := hltb.New(&http.Client{Transport: mt})
	game, err := client.SearchGame(t.Context(), "Half-Life Alyx")
	assert.NilError(t, err)
	assert.DeepEqual(t, game, &hltb.Game{
		Title:         "Half-Life: Alyx",
		URL:           "https://howlongtobeat.com/game/72067",
		MainStory:     "12 hours",
		MainPlusExtra: "14.5 hours",
		Completionist: "19 hours",
	})
}

func TestSearchGameRefreshesExpiredToken(t *testing.T) {
	t.Parallel()

	mt := httpmockx.NewMockTransport(t)
	initCalls := 0
	mt.RegisterResponder("GET", initURL, httpmockx.ResponderFunc(func(_ *http.Request) (*http.Response, error) {
		initCalls++
		return httpmock.NewJsonResponse(http.StatusOK, map[string]string{
			"token": "token-" + strconv.Itoa(initCalls),
			"hpKey": "challenge",
			"hpVal": "answer",
		})
	}))

	searchCalls := 0
	mt.RegisterResponder("POST", searchURL, httpmockx.ResponderFunc(func(req *http.Request) (*http.Response, error) {
		searchCalls++
		if searchCalls == 1 {
			assert.Equal(t, req.Header.Get("X-Auth-Token"), "token-1")
			return httpmock.NewStringResponse(http.StatusForbidden, `{"message":"expired"}`), nil
		}

		assert.Equal(t, req.Header.Get("X-Auth-Token"), "token-2")
		return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
			"data": []map[string]any{{
				"game_id":   1,
				"game_name": "Game",
			}},
		})
	}))

	client := hltb.New(&http.Client{Transport: mt})
	_, err := client.SearchGame(t.Context(), "Game")
	assert.NilError(t, err)
	assert.Equal(t, initCalls, 2)
	assert.Equal(t, searchCalls, 2)
}

func TestSearchGameNotFound(t *testing.T) {
	t.Parallel()

	mt := httpmockx.NewMockTransport(t)
	mt.RegisterResponder("GET", initURL, httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]string{
		"token": "token",
	}))
	mt.RegisterResponder("POST", searchURL, httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
		"data": []any{},
	}))

	client := hltb.New(&http.Client{Transport: mt})
	_, err := client.SearchGame(t.Context(), "missing")

	apiErr, ok := apiclient.AsError(err)
	assert.Assert(t, ok)
	assert.Equal(t, apiErr.StatusCode, http.StatusNotFound)
}

func TestSearchGameCachesAuth(t *testing.T) {
	t.Parallel()

	mt := httpmockx.NewMockTransport(t)
	initCalls := 0
	mt.RegisterResponder("GET", initURL, httpmockx.ResponderFunc(func(_ *http.Request) (*http.Response, error) {
		initCalls++
		return httpmock.NewJsonResponse(http.StatusOK, map[string]string{
			"token": "token",
		})
	}))

	searchCalls := 0
	mt.RegisterResponder("POST", searchURL, httpmockx.ResponderFunc(func(_ *http.Request) (*http.Response, error) {
		searchCalls++
		return httpmock.NewJsonResponse(http.StatusOK, map[string]any{
			"data": []map[string]any{{
				"game_id":   searchCalls,
				"game_name": "Game",
			}},
		})
	}))

	client := hltb.New(&http.Client{Transport: mt})
	_, err := client.SearchGame(t.Context(), "Game")
	assert.NilError(t, err)
	_, err = client.SearchGame(t.Context(), "Game")
	assert.NilError(t, err)

	assert.Equal(t, initCalls, 1)
	assert.Equal(t, searchCalls, 2)
}

func TestSearchGameErrors(t *testing.T) {
	t.Parallel()

	t.Run("Empty query", func(t *testing.T) {
		t.Parallel()

		client := hltb.New(&http.Client{Transport: httpmockx.NewMockTransport(t)})
		_, err := client.SearchGame(t.Context(), " \t ")
		assert.ErrorContains(t, err, "search term cannot be empty")
	})

	t.Run("Auth request", func(t *testing.T) {
		t.Parallel()

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", initURL, httpmock.NewStringResponder(http.StatusBadGateway, ""))

		client := hltb.New(&http.Client{Transport: mt})
		_, err := client.SearchGame(t.Context(), "Game")
		assertStatusError(t, err, http.StatusBadGateway)
	})

	t.Run("Empty auth token", func(t *testing.T) {
		t.Parallel()

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", initURL, httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]string{}))

		client := hltb.New(&http.Client{Transport: mt})
		_, err := client.SearchGame(t.Context(), "Game")
		assert.ErrorContains(t, err, "empty search token")
	})

	t.Run("Search request", func(t *testing.T) {
		t.Parallel()

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder("GET", initURL, httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]string{
			"token": "token",
		}))
		mt.RegisterResponder("POST", searchURL, httpmock.NewStringResponder(http.StatusBadGateway, ""))

		client := hltb.New(&http.Client{Transport: mt})
		_, err := client.SearchGame(t.Context(), "Game")
		assertStatusError(t, err, http.StatusBadGateway)
	})

	t.Run("Auth refresh", func(t *testing.T) {
		t.Parallel()

		mt := httpmockx.NewMockTransport(t)
		initCalls := 0
		mt.RegisterResponder("GET", initURL, httpmockx.ResponderFunc(func(_ *http.Request) (*http.Response, error) {
			initCalls++
			if initCalls == 1 {
				return httpmock.NewJsonResponse(http.StatusOK, map[string]string{
					"token": "expired",
				})
			}
			return httpmock.NewStringResponse(http.StatusBadGateway, ""), nil
		}))
		mt.RegisterResponder("POST", searchURL, httpmock.NewStringResponder(http.StatusForbidden, ""))

		client := hltb.New(&http.Client{Transport: mt})
		_, err := client.SearchGame(t.Context(), "Game")
		assertStatusError(t, err, http.StatusBadGateway)
	})
}

func assertStatusError(t *testing.T, err error, status int) {
	t.Helper()

	apiErr, ok := apiclient.AsError(err)
	assert.Assert(t, ok)
	assert.Equal(t, apiErr.StatusCode, status)
}
