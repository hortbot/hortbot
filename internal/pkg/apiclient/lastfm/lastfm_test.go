package lastfm_test

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()
	assertx.Panic(t, func() {
		lastfm.New("", nil)
	}, "empty apiKey")
}

func TestRecentTracks(t *testing.T) {
	t.Parallel()
	const (
		apiKey = "this_is_the_api_key"
		user   = "1"
		url    = "https://ws.audioscrobbler.com/2.0/"
		limit  = 2
	)

	query := "api_key=" + apiKey + "&limit=" + strconv.Itoa(limit) + "&method=user.getRecentTracks&user=" + user

	ctx := context.Background()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, apiResponse))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		tracks, err := lp.RecentTracks(ctx, user, limit)
		assert.NilError(t, err)
		assert.DeepEqual(t, tracks, []lastfm.Track{
			{
				NowPlaying: true,
				Name:       "Scarlet M∞N",
				Artist:     "MEIKO・巡音ルカ",
				URL:        "https://www.last.fm/music/MEIKO%E3%83%BB%E5%B7%A1%E9%9F%B3%E3%83%AB%E3%82%AB/_/Scarlet+M%E2%88%9EN",
				Time:       time.Unix(1404235898, 0),
			},
			{
				NowPlaying: false,
				Name:       "Daydream Flight",
				Artist:     "蒼姫ラピス",
				URL:        "https://www.last.fm/music/%E8%92%BC%E5%A7%AB%E3%83%A9%E3%83%94%E3%82%B9/_/Daydream+Flight",
				Time:       time.Unix(1404235556, 0),
			},
		})
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(404, api404))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		_, err := lp.RecentTracks(ctx, user, limit)
		assert.Error(t, err, "lastfm: ErrValidator: response error for https://ws.audioscrobbler.com/2.0/?api_key=REDACTED0&limit=2&method=user.getRecentTracks&user=1: unexpected status: 404")
	})

	t.Run("Bad response", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(200, "<"))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		_, err := lp.RecentTracks(ctx, user, limit)
		assert.ErrorContains(t, err, "XML syntax error")
	})

	t.Run("Server error", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(500, "{}"))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		_, err := lp.RecentTracks(ctx, user, limit)
		assert.Error(t, err, "lastfm: ErrValidator: response error for https://ws.audioscrobbler.com/2.0/?api_key=REDACTED0&limit=2&method=user.getRecentTracks&user=1: unexpected status: 500")
	})

	t.Run("Not authorized", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(403, "{}"))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		_, err := lp.RecentTracks(ctx, user, limit)
		assert.Error(t, err, "lastfm: ErrValidator: response error for https://ws.audioscrobbler.com/2.0/?api_key=REDACTED0&limit=2&method=user.getRecentTracks&user=1: unexpected status: 403")
	})

	t.Run("Unknown", func(t *testing.T) {
		t.Parallel()
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewStringResponder(418, "{}"))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		_, err := lp.RecentTracks(ctx, user, limit)
		assert.Error(t, err, "lastfm: ErrValidator: response error for https://ws.audioscrobbler.com/2.0/?api_key=REDACTED0&limit=2&method=user.getRecentTracks&user=1: unexpected status: 418")
	})

	t.Run("Request error", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("testing error")
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", url, query, httpmock.NewErrorResponder(testErr))

		lp := lastfm.New(apiKey, &http.Client{Transport: mt})

		_, err := lp.RecentTracks(ctx, user, limit)
		assert.ErrorContains(t, err, testErr.Error())
	})
}

const (
	apiResponse = `<?xml version="1.0" encoding="UTF-8"?>
	<lfm status="ok">
		<recenttracks user="zikaeroh" page="1" perPage="2" totalPages="533" total="1066">
			<track nowplaying="true">
				<artist mbid="">MEIKO・巡音ルカ</artist>
				<name>Scarlet M∞N</name>
				<streamable>0</streamable>
				<mbid></mbid>
				<album mbid=""></album>
				<url>https://www.last.fm/music/MEIKO%E3%83%BB%E5%B7%A1%E9%9F%B3%E3%83%AB%E3%82%AB/_/Scarlet+M%E2%88%9EN</url>
				<image size="small"></image>
				<image size="medium"></image>
				<image size="large"></image>
				<image size="extralarge"></image>
				<date uts="1404235898">01 Jul 2014, 17:31</date>
			</track>
			<track>
				<artist mbid="">蒼姫ラピス</artist>
				<name>Daydream Flight</name>
				<streamable>0</streamable>
				<mbid></mbid>
				<album mbid=""></album>
				<url>https://www.last.fm/music/%E8%92%BC%E5%A7%AB%E3%83%A9%E3%83%94%E3%82%B9/_/Daydream+Flight</url>
				<image size="small"></image>
				<image size="medium"></image>
				<image size="large"></image>
				<image size="extralarge"></image>
				<date uts="1404235556">01 Jul 2014, 17:25</date>
			</track>
		</recenttracks>
	</lfm>`
	api404 = `<?xml version="1.0" encoding="UTF-8"?>
	<lfm status="failed">
		<error code="6">User not found</error>
	</lfm>`
)
