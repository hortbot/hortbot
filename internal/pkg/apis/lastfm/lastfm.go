// Package lastfm provides a LastFM client.
package lastfm

import (
	"time"

	"github.com/Kovensky/go-lastfm"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Track represents a specific LastFM track.
type Track struct {
	NowPlaying bool
	Name       string
	Artist     string
	URL        string
	Time       time.Time
}

//counterfeiter:generate . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	RecentTracks(user string, n int) ([]Track, error)
}

// TODO: Fork LastFM package to expose internal client.

// LastFM is a LastFM API client.
type LastFM struct {
	api lastfm.LastFM
}

var _ API = (*LastFM)(nil)

// New creates a new LastFM client.
func New(apiKey string) *LastFM {
	if apiKey == "" {
		panic("empty apiKey")
	}

	return &LastFM{
		api: lastfm.New(apiKey),
	}
}

// RecentTracks gets the most recently played tracks for the user, limited to
// the n most recent tracks.
func (l *LastFM) RecentTracks(user string, n int) ([]Track, error) {
	resp, err := l.api.GetRecentTracks(user, n)
	if err != nil {
		return nil, err
	}

	tracks := resp.Tracks

	ts := make([]Track, len(tracks))

	for i, t := range tracks {
		ts[i] = Track{
			NowPlaying: t.NowPlaying,
			Name:       t.Name,
			Artist:     t.Artist.Name,
			Time:       t.Date,
		}
	}

	return ts, nil
}
