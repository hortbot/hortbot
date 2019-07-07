package lastfm

import (
	"time"

	"github.com/Kovensky/go-lastfm"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

type Track struct {
	NowPlaying bool
	Name       string
	Artist     string
	URL        string
	Time       time.Time
}

//counterfeiter:generate . API
type API interface {
	RecentTracks(user string, n int) ([]Track, error)
}

type LastFM struct {
	api lastfm.LastFM
}

var _ API = (*LastFM)(nil)

func New(apiKey string) *LastFM {
	return &LastFM{
		api: lastfm.New(apiKey),
	}
}

func (l *LastFM) RecentTracks(user string, n int) ([]Track, error) {
	resp, err := l.api.GetRecentTracks(user, n)
	if err != nil {
		return nil, err
	}

	tracks := resp.Tracks

	if len(tracks) > n {
		tracks = tracks[:n]
	}

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
