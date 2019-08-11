package btest

import (
	"context"
	"net/url"
	"testing"

	"github.com/hortbot/hortbot/internal/bot/botfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/extralife/extralifefakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm/lastfmfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch/twitchfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd/xkcdfakes"
	"github.com/hortbot/hortbot/internal/pkg/apis/youtube/youtubefakes"
	"golang.org/x/oauth2"
)

func newFakeRand(t testing.TB) *botfakes.FakeRand {
	f := &botfakes.FakeRand{}

	f.IntnCalls(func(_ int) int {
		t.Fatal("IntnCalls not implemented")
		return 0
	})

	f.Float64Calls(func() float64 {
		t.Fatal("Float64Calls not implemented")
		return 0
	})

	return f
}

func newFakeLastFM(t testing.TB) *lastfmfakes.FakeAPI {
	f := &lastfmfakes.FakeAPI{}

	f.RecentTracksCalls(func(_ string, _ int) ([]lastfm.Track, error) {
		t.Fatal("RecentTracksCalls not implemented")
		return nil, nil
	})

	return f
}

func newFakeYouTube(t testing.TB) *youtubefakes.FakeAPI {
	f := &youtubefakes.FakeAPI{}

	f.VideoTitleCalls(func(_ *url.URL) string {
		t.Fatal("VideoTitleCalls not implemented")
		return ""
	})

	return f
}

func newFakeXKCD(t testing.TB) *xkcdfakes.FakeAPI {
	f := &xkcdfakes.FakeAPI{}

	f.GetComicCalls(func(_ int) (*xkcd.Comic, error) {
		t.Fatal("GetComicCalls not implemented")
		return nil, nil
	})

	return f
}

func newFakeExtraLife(t testing.TB) *extralifefakes.FakeAPI {
	f := &extralifefakes.FakeAPI{}

	f.GetDonationAmountCalls(func(_ int) (float64, error) {
		t.Fatal("GetDonationAmountCalls not implemented")
		return 0, nil
	})

	return f
}

func newFakeTwitch(t testing.TB) *twitchfakes.FakeAPI {
	f := &twitchfakes.FakeAPI{}

	f.GetChattersCalls(func(context.Context, string) (*twitch.Chatters, error) {
		t.Fatal("GetChattersCalls not implemented")
		return nil, nil
	})

	f.GetCurrentStreamCalls(func(context.Context, int64) (*twitch.Stream, error) {
		t.Fatal("GetCurrentStreamCalls not implemented")
		return nil, nil
	})

	f.GetIDForTokenCalls(func(context.Context, *oauth2.Token) (int64, *oauth2.Token, error) {
		t.Fatal("GetIDForTokenCalls not implemented")
		return 0, nil, nil
	})

	f.GetIDForUsernameCalls(func(context.Context, string) (int64, error) {
		t.Fatal("GetIDForUsernameCalls not implemented")
		return 0, nil
	})

	f.SetChannelGameCalls(func(context.Context, int64, *oauth2.Token, string) (string, *oauth2.Token, error) {
		t.Fatal("SetChannelGameCalls not implemented")
		return "", nil, nil
	})

	f.SetChannelStatusCalls(func(context.Context, int64, *oauth2.Token, string) (string, *oauth2.Token, error) {
		t.Fatal("SetChannelStatusCalls not implemented")
		return "", nil, nil
	})

	f.GetChannelByIDCalls(func(context.Context, int64) (*twitch.Channel, error) {
		t.Fatal("GetChannelByIDCalls not implemented")
		return nil, nil
	})

	return f
}
