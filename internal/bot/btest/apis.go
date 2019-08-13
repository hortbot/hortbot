package btest

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/xkcd"
	"gotest.tools/assert"
)

func (st *scriptTester) noLastFM(t testing.TB, _, _ string, _ int) {
	st.addAction(func(ctx context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable LastFM")
		st.bc.LastFM = nil
	})
}

func (st *scriptTester) lastFMRecentTracks(t testing.TB, _, args string, lineNum int) {
	var v map[string][]lastfm.Track

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.lastFM.RecentTracksCalls(func(user string, n int) ([]lastfm.Track, error) {
			x := v[user]

			if len(x) > n {
				x = x[:n]
			}

			return x, nil
		})
	})
}

func (st *scriptTester) noYouTube(t testing.TB, _, _ string, _ int) {
	st.addAction(func(ctx context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable YouTube")
		st.bc.YouTube = nil
	})
}

func (st *scriptTester) youtubeVideoTitles(t testing.TB, _, args string, lineNum int) {
	var v map[string]string

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.youtube.VideoTitleCalls(func(u *url.URL) string {
			return v[u.String()]
		})
	})
}

func (st *scriptTester) noXKCD(t testing.TB, _, _ string, _ int) {
	st.addAction(func(ctx context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable XKCD")
		st.bc.XKCD = nil
	})
}

func (st *scriptTester) xkcdComics(t testing.TB, _, args string, lineNum int) {
	var v map[string]*xkcd.Comic

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.xkcd.GetComicCalls(func(id int) (*xkcd.Comic, error) {
			c, ok := v[strconv.Itoa(id)]
			if !ok {
				return nil, xkcd.ErrNotFound
			}
			return c, nil
		})
	})
}

func (st *scriptTester) noExtraLife(t testing.TB, _, _ string, _ int) {
	st.addAction(func(ctx context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable Extra Life")
		st.bc.ExtraLife = nil
	})
}

func (st *scriptTester) extraLifeAmounts(t testing.TB, _, args string, lineNum int) {
	var v map[string]float64

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.extraLife.GetDonationAmountCalls(func(id int) (float64, error) {
			a, ok := v[strconv.Itoa(id)]
			if !ok {
				return 0, extralife.ErrNotFound
			}
			return a, nil
		})
	})
}

func (st *scriptTester) noSteam(t testing.TB, _, _ string, _ int) {
	st.addAction(func(ctx context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable Steam")
		st.bc.Steam = nil
	})
}

func steamErr(t testing.TB, lineNum int, e string) error {
	switch e {
	case "":
		return nil
	case "ErrNotFound":
		return steam.ErrNotFound
	case "ErrNotAuthorized":
		return steam.ErrNotAuthorized
	case "ErrServerError":
		return steam.ErrServerError
	case "ErrUnknown":
		return steam.ErrUnknown
	default:
		t.Fatalf("unknown error type %s: line %d", e, lineNum)
		return nil
	}
}

func (st *scriptTester) steamGetPlayerSummary(t testing.TB, directive, args string, lineNum int) {
	var call struct {
		ID string

		Summary *steam.Summary
		Err     string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.steam.GetPlayerSummaryCalls(func(_ context.Context, id string) (*steam.Summary, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Summary, steamErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) steamGetOwnedGames(t testing.TB, directive, args string, lineNum int) {
	var call struct {
		ID string

		Games []*steam.Game
		Err   string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.steam.GetOwnedGamesCalls(func(_ context.Context, id string) ([]*steam.Game, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Games, steamErr(t, lineNum, call.Err)
		})
	})
}
