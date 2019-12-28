// Package youtube proivdes a YouTube API client.
package youtube

import (
	"net/url"

	"github.com/rylio/ytdl"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	VideoTitle(u *url.URL) string
}

// YouTube is a YouTube API client.
type YouTube struct{}

var _ API = &YouTube{}

// New creates a new YouTube client.
func New() *YouTube {
	return &YouTube{}
}

// VideoTitle returns the title for the specified YouTUbe video, or an empty
// string if a failure occurs.
func (*YouTube) VideoTitle(u *url.URL) string {
	if info, _ := ytdl.GetVideoInfoFromURL(u); info != nil {
		return info.Title
	}
	return ""
}
