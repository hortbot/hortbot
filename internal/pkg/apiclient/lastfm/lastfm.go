// Package lastfm provides a LastFM client.
package lastfm

import (
	"context"
	"encoding/xml"
	"net/http"
	"strconv"
	"time"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go run github.com/matryer/moq -fmt goimports -out lastfmmocks/mocks.go -pkg lastfmmocks . API

// Track represents a specific LastFM track.
type Track struct {
	NowPlaying bool      `json:"now_playing"`
	Name       string    `json:"name"`
	Artist     string    `json:"artist"`
	URL        string    `json:"url"`
	Time       time.Time `json:"time"`
}

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	RecentTracks(ctx context.Context, user string, n int) ([]Track, error)
}

// LastFM is a LastFM API client.
type LastFM struct {
	apiKey string
	cli    httpx.Client
}

var _ API = (*LastFM)(nil)

// New creates a new LastFM client.
func New(apiKey string, cli *http.Client) *LastFM {
	if apiKey == "" {
		panic("empty apiKey")
	}

	return &LastFM{
		apiKey: apiKey,
		cli:    httpx.NewClient(cli, "lastfm", false),
	}
}

// RecentTracks gets the most recently played tracks for the user, limited to
// the n most recent tracks.
func (l *LastFM) RecentTracks(ctx context.Context, user string, n int) ([]Track, error) {
	var body struct {
		XMLName      xml.Name `xml:"lfm"`
		RecentTracks struct {
			Tracks []struct {
				NowPlaying bool   `xml:"nowplaying,attr"`
				Artist     string `xml:"artist"`
				Name       string `xml:"name"`
				URL        string `xml:"url"`
				Date       struct {
					UTS int64 `xml:"uts,attr"`
				} `xml:"date"`
			} `xml:"track"`
		} `xml:"recenttracks"`
	}

	req := l.cli.NewRequest("https://ws.audioscrobbler.com/2.0/").
		Param("api_key", l.apiKey).
		Param("limit", strconv.Itoa(n)).
		Param("method", "user.getRecentTracks").
		Param("user", user).
		Handle(func(r *http.Response) error {
			return xml.NewDecoder(r.Body).Decode(&body)
		})

	if err := req.Fetch(ctx); err != nil {
		return nil, apiclient.WrapRequestErr("lastfm", err, []string{l.apiKey})
	}

	tracks := body.RecentTracks.Tracks

	ts := make([]Track, len(tracks))

	for i, t := range tracks {
		ts[i] = Track{
			NowPlaying: t.NowPlaying,
			Name:       t.Name,
			Artist:     t.Artist,
			URL:        t.URL,
			Time:       time.Unix(t.Date.UTS, 0),
		}
	}

	return ts, nil
}
