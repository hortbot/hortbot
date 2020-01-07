package btest

import (
	"testing"

	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife/extralifefakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm/lastfmfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam/steamfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl/tinyurlfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch/twitchfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/urban/urbanfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd/xkcdfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube/youtubefakes"
	"github.com/hortbot/hortbot/internal/pkg/fakex"
)

func newFakeRand(t testing.TB) *botfakes.FakeRand {
	f := &botfakes.FakeRand{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newFakeLastFM(t testing.TB) *lastfmfakes.FakeAPI {
	f := &lastfmfakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newFakeYouTube(t testing.TB) *youtubefakes.FakeAPI {
	f := &youtubefakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newFakeXKCD(t testing.TB) *xkcdfakes.FakeAPI {
	f := &xkcdfakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newFakeExtraLife(t testing.TB) *extralifefakes.FakeAPI {
	f := &extralifefakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newFakeTwitch(t testing.TB) *twitchfakes.FakeAPI {
	f := &twitchfakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newFakeSteam(t testing.TB) *steamfakes.FakeAPI {
	f := &steamfakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newTinyURL(t testing.TB) *tinyurlfakes.FakeAPI {
	f := &tinyurlfakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}

func newUrban(t testing.TB) *urbanfakes.FakeAPI {
	f := &urbanfakes.FakeAPI{}
	fakex.StubNotImplemented(t, f)
	return f
}
