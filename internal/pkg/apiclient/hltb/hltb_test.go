package hltb_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestSearchGame(t *testing.T) {
	ctx := context.Background()

	t.Run("OK", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(200, `
				<h3 class='global_padding shadow_box back_blue center'>We Found 1 Games for "Half-Life Alyx"</h3>
				<ul>
					<div class="clear"></div>
					<li class="back_darkish"
						style="background-image:linear-gradient(rgb(31, 31, 31), rgba(31, 31, 31, 0.9)), url('https://howlongtobeat.com/games/72067_Half-Life_Alyx.jpg')">
						<div class="search_list_image">
							<a aria-label="HalfLife Alyx" title="HalfLife Alyx" href="game?id=72067">
								<img alt="Box Art" src="https://howlongtobeat.com/games/72067_Half-Life_Alyx.jpg" />
							</a>
						</div>
						<div class="search_list_details">
							<h3 class="shadow_text">
								<a class="text_white" title="HalfLife Alyx" href="game?id=72067">Half-Life: Alyx</a>
							</h3>
							<div class="search_list_details_block">
								<div>
									<div class="search_list_tidbit text_white shadow_text">Main Story</div>
									<div class="search_list_tidbit center time_100">10 Hours </div>
									<div class="search_list_tidbit text_white shadow_text">Main + Extra</div>
									<div class="search_list_tidbit center time_100">12 Hours </div>
									<div class="search_list_tidbit text_white shadow_text">Completionist</div>
									<div class="search_list_tidbit center time_40">13&#189; Hours </div>
								</div>
							</div>
						</div>
					</li>
					<div class="clear"></div>
				</ul>`))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		game, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.NilError(t, err)
		assert.DeepEqual(t, game, &hltb.Game{
			Title:         "Half-Life: Alyx",
			MainStory:     "10 hours",
			MainPlusExtra: "12 hours",
			Completionist: "13.5 hours",
		})
	})

	t.Run("Dashes", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(200, `
				<h3 class='global_padding shadow_box back_blue center'>We Found 1 Games for "Half-Life Alyx"</h3>
				<ul>
					<div class="clear"></div>
					<li class="back_darkish"
						style="background-image:linear-gradient(rgb(31, 31, 31), rgba(31, 31, 31, 0.9)), url('https://howlongtobeat.com/games/72067_Half-Life_Alyx.jpg')">
						<div class="search_list_image">
							<a aria-label="HalfLife Alyx" title="HalfLife Alyx" href="game?id=72067">
								<img alt="Box Art" src="https://howlongtobeat.com/games/72067_Half-Life_Alyx.jpg" />
							</a>
						</div>
						<div class="search_list_details">
							<h3 class="shadow_text">
								<a class="text_white" title="HalfLife Alyx" href="game?id=72067">Half-Life: Alyx</a>
							</h3>
							<div class="search_list_details_block">
								<div>
									<div class="search_list_tidbit text_white shadow_text">Main Story</div>
									<div class="search_list_tidbit center time_100">10 Hours </div>
									<div class="search_list_tidbit text_white shadow_text">Main + Extra</div>
									<div class="search_list_tidbit center time_100">--</div>
									<div class="search_list_tidbit text_white shadow_text">Completionist</div>
									<div class="search_list_tidbit center time_40">--</div>
								</div>
							</div>
						</div>
					</li>
					<div class="clear"></div>
				</ul>`))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		game, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.NilError(t, err)
		assert.DeepEqual(t, game, &hltb.Game{
			Title:     "Half-Life: Alyx",
			MainStory: "10 hours",
		})
	})

	t.Run("Passes query", func(t *testing.T) {
		const query = "Half-Life Alyx"

		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			func(r *http.Request) (*http.Response, error) {
				assert.NilError(t, r.ParseForm())

				for k, v := range hltb.FormCommon {
					assert.DeepEqual(t, r.Form[k], v)
				}

				assert.Equal(t, r.FormValue("queryString"), query)
				return httpmock.NewStringResponse(200, ""), nil
			},
		)

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))
		_, _ = h.SearchGame(ctx, query)
	})

	t.Run("No results", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(200, `
			<li class='global_padding back_primary shadow_box'>No results for <strong>This is a fake game ignore me</strong> in
				<u>games</u>.</li>
			<div class='clear'></div>`))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		_, err := h.SearchGame(ctx, "This is a fake game ignore me")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 404})
	})

	t.Run("404 code", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(404, `
			<li class='global_padding back_primary shadow_box'>No results for <strong>This is a fake game ignore me</strong> in
				<u>games</u>.</li>
			<div class='clear'></div>`))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		_, err := h.SearchGame(ctx, "This is a fake game ignore me")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 404})
	})

	t.Run("500 code", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(500, `
			<li class='global_padding back_primary shadow_box'>No results for <strong>This is a fake game ignore me</strong> in
				<u>games</u>.</li>
			<div class='clear'></div>`))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		_, err := h.SearchGame(ctx, "This is a fake game ignore me")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 500})
	})

	t.Run("Empty response", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(200, ``))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		_, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 404})
	})

	t.Run("Incomplete results", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			httpmock.NewStringResponder(200, `
				<h3 class='global_padding shadow_box back_blue center'>We Found 1 Games for "Half-Life Alyx"</h3>
				<ul>
					<div class="clear"></div>
					<li class="back_darkish"
						style="background-image:linear-gradient(rgb(31, 31, 31), rgba(31, 31, 31, 0.9)), url('https://howlongtobeat.com/games/72067_Half-Life_Alyx.jpg')">
						<div class="search_list_image">
							<a aria-label="HalfLife Alyx" title="HalfLife Alyx" href="game?id=72067">
								<img alt="Box Art" src="https://howlongtobeat.com/games/72067_Half-Life_Alyx.jpg" />
							</a>
						</div>
						<div class="search_list_details">
							<h3 class="shadow_text">
								<a class="text_white" title="HalfLife Alyx" href="game?id=72067">Half-Life: Alyx</a>
							</h3>
							<div class="search_list_details_block">
								<div></div>
							</div>
						</div>
					</li>
					<div class="clear"></div>
				</ul>`))

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))

		_, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.DeepEqual(t, err, &apiclient.Error{API: "hltb", StatusCode: 404})
	})

	t.Run("Client error", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)

		errTest := errors.New("test error")

		mt.RegisterResponder(
			"POST",
			"https://howlongtobeat.com/search_results?page=1",
			func(_ *http.Request) (*http.Response, error) {
				return nil, errTest
			},
		)

		h := hltb.New(hltb.HTTPClient(&http.Client{Transport: mt}))
		_, err := h.SearchGame(ctx, "Half-Life Alyx")
		assert.ErrorContains(t, err, errTest.Error())
	})
}
