// Package youtube proivdes a YouTube API client.
package youtube

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out youtubemocks/mocks.go -pkg youtubemocks . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	VideoTitle(ctx context.Context, u *url.URL) string
}

// YouTube is a YouTube API client.
type YouTube struct {
	apiKey string
	cli    httpx.Client
}

var _ API = &YouTube{}

// New creates a new YouTube client.
func New(apiKey string, opts ...Option) *YouTube {
	if apiKey == "" {
		panic("empty apiKey")
	}

	yt := &YouTube{
		apiKey: apiKey,
		cli: httpx.Client{
			Name: "youtube",
		},
	}

	for _, o := range opts {
		o(yt)
	}

	return yt
}

// Option controls client functionality.
type Option func(*YouTube)

// HTTPClient sets the YouTube client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(y *YouTube) {
		y.cli.Client = cli
	}
}

// VideoTitle returns the title for the specified YouTUbe video, or an empty
// string if a failure occurs.
func (y *YouTube) VideoTitle(ctx context.Context, u *url.URL) string {
	id := extractVideoID(u)
	if id == "" {
		return ""
	}

	url := "https://www.googleapis.com/youtube/v3/videos?part=snippet&key=" + url.QueryEscape(y.apiKey) + "&id=" + url.QueryEscape(id)

	resp, err := y.cli.Get(ctx, url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var body struct {
		Items []struct {
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := jsonx.DecodeSingle(resp.Body, &body); err != nil {
		return ""
	}

	if len(body.Items) == 0 {
		return ""
	}

	return body.Items[0].Snippet.Title
}

// From https://github.com/rylio/ytdl, MIT licensed.
func extractVideoID(u *url.URL) string {
	switch u.Host {
	case "www.youtube.com", "youtube.com":
		if path.Clean(u.Path) == "/watch" {
			return u.Query().Get("v")
		}
		if strings.HasPrefix(u.Path, "/embed/") {
			return u.Path[7:]
		}
	case "youtu.be":
		if len(u.Path) > 1 {
			return path.Clean(u.Path[1:])
		}
	}
	return ""
}
