package lastfm_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestRecentTracks(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	const apiKey = "this_is_the_api_key"

	errTest := errors.New("test error")

	httpmock.RegisterResponder(
		"GET",
		"http://ws.audioscrobbler.com/2.0/?api_key=this_is_the_api_key&extended=1&limit=2&method=user.getRecentTracks&user=zikaeroh",
		httpmock.NewStringResponder(200, apiResponse),
	)

	httpmock.RegisterResponder(
		"GET",
		"http://ws.audioscrobbler.com/2.0/?api_key=this_is_the_api_key&extended=1&limit=2&method=user.getRecentTracks&user=thisuserdoesnotexistreally",
		httpmock.NewStringResponder(404, api404),
	)

	httpmock.RegisterResponder(
		"GET",
		"http://ws.audioscrobbler.com/2.0/?api_key=this_is_the_api_key&extended=1&limit=2&method=user.getRecentTracks&user=clienterror",
		func(_ *http.Request) (*http.Response, error) {
			return nil, errTest
		},
	)

	t.Run("OK", func(t *testing.T) {
		lp := lastfm.New(apiKey)

		tracks, err := lp.RecentTracks("zikaeroh", 2)
		assert.NilError(t, err)
		assert.DeepEqual(t, tracks, []lastfm.Track{
			{
				NowPlaying: false,
				Name:       "Scarlet M∞N",
				Artist:     "MEIKO・巡音ルカ",
				Time:       time.Unix(1404235898, 0),
			},
			{
				NowPlaying: false,
				Name:       "Daydream Flight",
				Artist:     "蒼姫ラピス",
				Time:       time.Unix(1404235556, 0),
			},
		})
	})

	t.Run("Not found", func(t *testing.T) {
		lp := lastfm.New(apiKey)

		_, err := lp.RecentTracks("thisuserdoesnotexistreally", 2)

		assert.ErrorContains(t, err, "User not found")
	})

	t.Run("Client error", func(t *testing.T) {
		lp := lastfm.New(apiKey)

		_, err := lp.RecentTracks("clienterror", 2)
		assert.ErrorContains(t, err, errTest.Error())
	})
}

const (
	apiResponse = `<?xml version="1.0" encoding="UTF-8"?>
<lfm status="ok">
    <recenttracks user="zikaeroh" page="1" perPage="2" totalPages="533" total="1066">
        <track>
            <artist>
                <name>MEIKO・巡音ルカ</name>
                <mbid></mbid>
                <url>https://www.last.fm/music/MEIKO%E3%83%BB%E5%B7%A1%E9%9F%B3%E3%83%AB%E3%82%AB</url>
                <image size="small">https://lastfm-img2.akamaized.net/i/u/34s/2a96cbd8b46e442fc41c2b86b821562f.png</image>
                <image size="medium">https://lastfm-img2.akamaized.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png</image>
                <image size="large">https://lastfm-img2.akamaized.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png</image>
                <image size="extralarge">https://lastfm-img2.akamaized.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png</image>
            </artist>
            <loved>0</loved>
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
            <artist>
                <name>蒼姫ラピス</name>
                <mbid></mbid>
                <url>https://www.last.fm/music/%E8%92%BC%E5%A7%AB%E3%83%A9%E3%83%94%E3%82%B9</url>
                <image size="small">https://lastfm-img2.akamaized.net/i/u/34s/2a96cbd8b46e442fc41c2b86b821562f.png</image>
                <image size="medium">https://lastfm-img2.akamaized.net/i/u/64s/2a96cbd8b46e442fc41c2b86b821562f.png</image>
                <image size="large">https://lastfm-img2.akamaized.net/i/u/174s/2a96cbd8b46e442fc41c2b86b821562f.png</image>
                <image size="extralarge">https://lastfm-img2.akamaized.net/i/u/300x300/2a96cbd8b46e442fc41c2b86b821562f.png</image>
            </artist>
            <loved>0</loved>
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
