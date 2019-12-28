package youtube

import (
	"net/url"

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

func (*YouTube) VideoTitle(u *url.URL) string {
	if info, _ := ytdl.GetVideoInfoFromURL(u); info != nil {
		return info.Title
	}
	return ""
}
