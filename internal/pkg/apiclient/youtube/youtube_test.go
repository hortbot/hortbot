package youtube_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/youtube"
	"github.com/hortbot/hortbot/internal/pkg/assertx"
	"github.com/hortbot/hortbot/internal/pkg/httpmockx"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
)

func TestNew(t *testing.T) {
	assertx.Panic(t, func() {
		_ = youtube.New("", nil)
	}, "empty apiKey")
}

func TestVideoTitle(t *testing.T) {
	const (
		apiKey    = "THISISTHEAPIKEY123456789"
		id        = "90X5NJleYJQ"
		baseURL   = "https://www.googleapis.com/youtube/v3/videos"
		query     = "part=snippet&key=" + apiKey + "&id=" + id
		wantTitle = "Strong Bad Email #58 - Dragon"
	)

	goodURLs := []string{
		"https://www.youtube.com/watch?v=" + id,
		"https://www.youtube.com/watch/?v=" + id,
		"https://youtube.com/watch?v=" + id,
		"https://youtube.com/watch?v=" + id,
		"https://youtu.be/" + id,
		"https://youtu.be/" + id + "/",
		"https://www.youtube.com/embed/" + id,
	}

	ctx := context.Background()

	for _, u := range goodURLs {
		t.Run(u, func(t *testing.T) {
			mt := httpmockx.NewMockTransport(t)
			mt.RegisterResponderWithQuery("GET", baseURL, query, httpmock.NewStringResponder(200, response))

			y := youtube.New(apiKey, &http.Client{Transport: mt})

			title := y.VideoTitle(ctx, parseURL(t, u))
			assert.Equal(t, title, wantTitle)
		})
	}

	badURLs := []string{
		"example.com",
		"https://yout.be/" + id,
		"https://youtu.be/",
		"https://youtube.com/watch?v=",
		"https://www.youtube.com/embed/",
	}

	for _, u := range badURLs {
		t.Run(u, func(t *testing.T) {
			mt := httpmockx.NewMockTransport(t)
			y := youtube.New(apiKey, &http.Client{Transport: mt})

			title := y.VideoTitle(ctx, parseURL(t, u))
			assert.Equal(t, title, "")
		})
	}

	t.Run("Not found", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", baseURL, query, httpmock.NewStringResponder(200, emptyResponse))

		y := youtube.New(apiKey, &http.Client{Transport: mt})

		title := y.VideoTitle(ctx, parseURL(t, "https://youtube.com/watch?v="+id))
		assert.Equal(t, title, "")
	})

	t.Run("Not found", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", baseURL, query, httpmock.NewErrorResponder(errors.New("testing error")))

		y := youtube.New(apiKey, &http.Client{Transport: mt})

		title := y.VideoTitle(ctx, parseURL(t, "https://youtube.com/watch?v="+id))
		assert.Equal(t, title, "")
	})

	t.Run("Decode error", func(t *testing.T) {
		mt := httpmockx.NewMockTransport(t)
		mt.RegisterResponderWithQuery("GET", baseURL, query, httpmock.NewStringResponder(200, "{"))

		y := youtube.New(apiKey, &http.Client{Transport: mt})

		title := y.VideoTitle(ctx, parseURL(t, "https://youtube.com/watch?v="+id))
		assert.Equal(t, title, "")
	})
}

func parseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	assert.NilError(t, err)
	return u
}

const response = `
{
    "kind": "youtube#videoListResponse",
    "etag": "\"p4VTdlkQv3HQeTEaXgvLePAydmU/fUCFI-cTagkd4b_Ea8SdISzvWxk\"",
    "pageInfo": {
        "totalResults": 1,
        "resultsPerPage": 1
    },
    "items": [
        {
            "kind": "youtube#video",
            "etag": "\"p4VTdlkQv3HQeTEaXgvLePAydmU/8CUXy0jsNniZOgqz5MlUHMrYE3M\"",
            "id": "90X5NJleYJQ",
            "snippet": {
                "publishedAt": "2009-03-31T21:02:18.000Z",
                "channelId": "UCMkbjxvwur30YrFWw8kpSaw",
                "title": "Strong Bad Email #58 - Dragon",
                "description": "Strong Bad teaches the world how to draw a dragon.",
                "thumbnails": {
                    "default": {
                        "url": "https://i.ytimg.com/vi/90X5NJleYJQ/default.jpg",
                        "width": 120,
                        "height": 90
                    },
                    "medium": {
                        "url": "https://i.ytimg.com/vi/90X5NJleYJQ/mqdefault.jpg",
                        "width": 320,
                        "height": 180
                    },
                    "high": {
                        "url": "https://i.ytimg.com/vi/90X5NJleYJQ/hqdefault.jpg",
                        "width": 480,
                        "height": 360
                    }
                },
                "channelTitle": "homestarrunnerdotcom",
                "tags": [
                    "Homestarrunner.com",
                    "Homestar",
                    "Homestar Runner",
                    "Strong Bad",
                    "Sbemail",
                    "Strong Bad Email",
                    "Strongbad",
                    "Trogdor",
                    "Dragon",
                    "The S is for Sucks",
                    "Strong Mad",
                    "Homsar",
                    "Strong Sad",
                    "Coach Z",
                    "Kerek",
                    "Beefy Arm",
                    "Consummate V's",
                    "Trogdor the Burninator",
                    "Majesty",
                    "His Majesty",
                    "Burninating",
                    "Thatched-Roof Cottages",
                    "Peasants Quest",
                    "Peasant",
                    "Fire",
                    "The Paper",
                    "Lappy 486",
                    "Compy 386",
                    "Tandy 400",
                    "Animation",
                    "Funny",
                    "Cartoon",
                    "Board Game",
                    "Drawing",
                    "How to Draw",
                    "Skills of an artist",
                    "Comedy",
                    "Flash Animation"
                ],
                "categoryId": "23",
                "liveBroadcastContent": "none",
                "localized": {
                    "title": "Strong Bad Email #58 - Dragon",
                    "description": "Strong Bad teaches the world how to draw a dragon."
                }
            }
        }
    ]
}
`

const emptyResponse = `
{
    "kind": "youtube#videoListResponse",
    "etag": "\"p4VTdlkQv3HQeTEaXgvLePAydmU/Rk41fm-2TD0VG1yv0-bkUvcBi9s\"",
    "pageInfo": {
        "totalResults": 0,
        "resultsPerPage": 0
    },
    "items": []
}
`
