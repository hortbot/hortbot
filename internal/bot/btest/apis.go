package btest

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apis/extralife"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
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
