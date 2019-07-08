package youtube

import (
	"net/url"
	"strings"

	"github.com/rylio/ytdl"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API
type API interface {
	VideoTitle(u *url.URL) string
}

type YouTube struct{}

var _ API = &YouTube{}

func New() *YouTube {
	return &YouTube{}
}

func (_ YouTube) VideoTitle(u *url.URL) string {
	short := false
	switch {
	case u.Host == "youtu.be":
		short = true
	case u.Host == "youtube.com":
	case strings.HasSuffix(u.Host, ".youtu.be"):
		short = true
	case strings.HasSuffix(u.Host, ".youtube.com"):
	default:
		return ""
	}

	var info *ytdl.VideoInfo
	if short {
		info, _ = ytdl.GetVideoInfoFromShortURL(u)
	} else {
		info, _ = ytdl.GetVideoInfoFromURL(u)
	}

	if info == nil {
		return ""
	}

	return info.Title
}
