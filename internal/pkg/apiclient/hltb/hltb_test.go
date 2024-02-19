package hltb_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/pkg/useragent"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestSearchGame(t *testing.T) {
	ctx := context.Background()

	t.Run("OK", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			httpmock.NewStringResponder(200, `{
				"color": "blue",
				"title": "",
				"category": "games",
				"count": 1,
				"pageCurrent": 1,
				"pageTotal": 1,
				"pageSize": 20,
				"data": [
					{
						"count": 1,
						"game_id": 72067,
						"game_name": "Half-Life: Alyx",
						"game_name_date": 0,
						"game_alias": "",
						"game_type": "game",
						"game_image": "72067_Half-Life_Alyx.jpg",
						"comp_lvl_combine": 0,
						"comp_lvl_sp": 1,
						"comp_lvl_co": 0,
						"comp_lvl_mp": 0,
						"comp_lvl_spd": 1,
						"comp_main": 43464,
						"comp_plus": 50837,
						"comp_100": 66013,
						"comp_all": 47867,
						"comp_main_count": 406,
						"comp_plus_count": 314,
						"comp_100_count": 73,
						"comp_all_count": 793,
						"invested_co": 0,
						"invested_mp": 0,
						"invested_co_count": 0,
						"invested_mp_count": 0,
						"count_comp": 1193,
						"count_speedrun": 0,
						"count_backlog": 1161,
						"count_review": 343,
						"review_score": 95,
						"count_playing": 67,
						"count_retired": 34,
						"profile_dev": "Valve",
						"profile_popular": 285,
						"profile_steam": 546560,
						"profile_platform": "PC",
						"release_world": 2020
					}
				],
				"displayModifier": null
			}`))

		h := hltb.New(&http.Client{Transport: mt})

		game, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.NilError(t, err)
		assert.DeepEqual(t, game, &hltb.Game{
			Title:         "Half-Life: Alyx",
			URL:           "https://howlongtobeat.com/game/72067",
			MainStory:     "12 hours",
			MainPlusExtra: "14 hours",
			Completionist: "18.5 hours",
		})
	})

	t.Run("All no times", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			httpmock.NewStringResponder(200, `{
				"color": "blue",
				"title": "",
				"category": "games",
				"count": 1,
				"pageCurrent": 1,
				"pageTotal": 1,
				"pageSize": 20,
				"data": [
					{
						"count": 1,
						"game_id": 72067,
						"game_name": "Half-Life: Alyx",
						"game_name_date": 0,
						"game_alias": "",
						"game_type": "game",
						"game_image": "72067_Half-Life_Alyx.jpg",
						"comp_lvl_combine": 0,
						"comp_lvl_sp": 1,
						"comp_lvl_co": 0,
						"comp_lvl_mp": 0,
						"comp_lvl_spd": 1,
						"comp_main": 0,
						"comp_plus": 0,
						"comp_100": 0,
						"comp_all": 47867,
						"comp_main_count": 406,
						"comp_plus_count": 314,
						"comp_100_count": 73,
						"comp_all_count": 793,
						"invested_co": 0,
						"invested_mp": 0,
						"invested_co_count": 0,
						"invested_mp_count": 0,
						"count_comp": 1193,
						"count_speedrun": 0,
						"count_backlog": 1161,
						"count_review": 343,
						"review_score": 95,
						"count_playing": 67,
						"count_retired": 34,
						"profile_dev": "Valve",
						"profile_popular": 285,
						"profile_steam": 546560,
						"profile_platform": "PC",
						"release_world": 2020
					}
				],
				"displayModifier": null
			}`))

		h := hltb.New(&http.Client{Transport: mt})

		game, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.NilError(t, err)
		assert.DeepEqual(t, game, &hltb.Game{
			Title: "Half-Life: Alyx",
			URL:   "https://howlongtobeat.com/game/72067",
		})
	})

	t.Run("Passes query", func(t *testing.T) {
		const query = "Half-Life Alyx"

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			func(r *http.Request) (*http.Response, error) {
				assert.Assert(t, r.UserAgent() != useragent.Bot(), "wrong user agent: "+r.UserAgent())

				var body hltb.RequestBody
				assert.NilError(t, jsonx.DecodeSingle(r.Body, &body))
				assert.NilError(t, r.ParseForm())

				assert.DeepEqual(t, body.SearchTerms, []string{"Half-Life", "Alyx"})
				return httpmock.NewStringResponse(200, ""), nil
			},
		)

		h := hltb.New(&http.Client{Transport: mt})
		_, _ = h.SearchGame(ctx, query)
	})

	t.Run("No results", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			httpmock.NewStringResponder(200, `{"color":"blue","title":"","category":"games","pageCurrent":1,"pageTotal":null,"pageSize":20,"data":[],"displayModifier":null}`))

		h := hltb.New(&http.Client{Transport: mt})

		_, err := h.SearchGame(ctx, "This is a fake game ignore me")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 404})
	})

	t.Run("404 code", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			httpmock.NewStringResponder(404, `{"color":"blue","title":"","category":"games","pageCurrent":1,"pageTotal":null,"pageSize":20,"data":[],"displayModifier":null}`))

		h := hltb.New(&http.Client{Transport: mt})

		_, err := h.SearchGame(ctx, "This is a fake game ignore me")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 404})
	})

	t.Run("500 code", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			httpmock.NewStringResponder(500, `{}`))

		h := hltb.New(&http.Client{Transport: mt})

		_, err := h.SearchGame(ctx, "This is a fake game ignore me")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 500})
	})

	t.Run("Empty response", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			httpmock.NewStringResponder(200, ``))

		h := hltb.New(&http.Client{Transport: mt})

		_, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.Error(t, err, "hltb: error decoding response: EOF")
	})

	t.Run("Client error", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)

		errTest := errors.New("test error")

		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/api/search",
			func(_ *http.Request) (*http.Response, error) {
				return nil, errTest
			},
		)

		h := hltb.New(&http.Client{Transport: mt})
		_, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.ErrorContains(t, err, errTest.Error())
	})
}
