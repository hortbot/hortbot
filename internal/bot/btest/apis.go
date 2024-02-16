package btest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/hltb"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/urban"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/xkcd"
	"gotest.tools/v3/assert"
)

func (st *scriptTester) noLastFM(t testing.TB, _, _ string, _ int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable LastFM")
		st.bc.LastFM = nil
	})
}

func (st *scriptTester) lastFMRecentTracks(t testing.TB, _, args string, lineNum int) {
	var v map[string][]lastfm.Track

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.lastFM.RecentTracksFunc = func(_ context.Context, user string, n int) ([]lastfm.Track, error) {
			if user == "error" {
				return nil, &apiclient.Error{API: "lastfm", StatusCode: 500}
			}

			x := v[user]

			if len(x) > n {
				x = x[:n]
			}

			return x, nil
		}
	})
}

func (st *scriptTester) noYouTube(t testing.TB, _, _ string, _ int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable YouTube")
		st.bc.YouTube = nil
	})
}

func (st *scriptTester) youtubeVideoTitles(t testing.TB, _, args string, lineNum int) {
	var v map[string]string

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.youtube.VideoTitleFunc = func(_ context.Context, u *url.URL) string {
			return v[u.String()]
		}
	})
}

func (st *scriptTester) noXKCD(t testing.TB, _, _ string, _ int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable XKCD")
		st.bc.XKCD = nil
	})
}

func (st *scriptTester) xkcdComics(t testing.TB, _, args string, lineNum int) {
	var v map[string]*xkcd.Comic

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.xkcd.GetComicFunc = func(_ context.Context, id int) (*xkcd.Comic, error) {
			c, ok := v[strconv.Itoa(id)]
			if !ok {
				return nil, xkcd.ErrNotFound
			}
			return c, nil
		}
	})
}

func (st *scriptTester) noExtraLife(t testing.TB, _, _ string, _ int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable Extra Life")
		st.bc.ExtraLife = nil
	})
}

func (st *scriptTester) extraLifeAmounts(t testing.TB, _, args string, lineNum int) {
	var v map[string]float64

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.extraLife.GetDonationAmountFunc = func(_ context.Context, id int) (float64, error) {
			if id == 500500 {
				return 0, &apiclient.Error{API: "extralife", StatusCode: 500}
			}

			a, ok := v[strconv.Itoa(id)]
			if !ok {
				return 0, &apiclient.Error{API: "extralife", StatusCode: 404}
			}
			return a, nil
		}
	})
}

func (st *scriptTester) noSteam(t testing.TB, _, _ string, _ int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable Steam")
		st.bc.Steam = nil
	})
}

func steamErr(t testing.TB, lineNum int, e string) error {
	switch e {
	case "":
		return nil
	case "ErrNotFound":
		return &apiclient.Error{API: "steam", StatusCode: http.StatusNotFound}
	case "ErrNotAuthorized":
		return &apiclient.Error{API: "steam", StatusCode: http.StatusUnauthorized}
	case "ErrServerError":
		return &apiclient.Error{API: "steam", StatusCode: http.StatusInternalServerError}
	case "ErrUnknown":
		return &apiclient.Error{API: "steam", StatusCode: 418}
	default:
		t.Fatalf("unknown error type %s: line %d", e, lineNum)
		return nil
	}
}

func (st *scriptTester) steamGetPlayerSummary(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID string

		Summary *steam.Summary
		Err     string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.steam.GetPlayerSummaryFunc = func(_ context.Context, id string) (*steam.Summary, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Summary, steamErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) steamGetOwnedGames(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID string

		Games []*steam.Game
		Err   string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.steam.GetOwnedGamesFunc = func(_ context.Context, id string) ([]*steam.Game, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Games, steamErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) noTinyURL(t testing.TB, _, args string, lineNum int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable TinyURL")
		st.bc.TinyURL = nil
	})
}

func (st *scriptTester) tinyURLShorten(t testing.TB, _, args string, lineNum int) {
	var call struct {
		Link string

		Short string
		Err   string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.tinyURL.ShortenFunc = func(_ context.Context, link string) (string, error) {
			assert.Equal(t, link, call.Link, "line %d", lineNum)

			var err error
			switch call.Err {
			case "ErrServerError":
				err = tinyurl.ErrServerError
			case "":
			default:
				t.Fatalf("unknown error type %s: line %d", call.Err, lineNum)
			}

			return call.Short, err
		}
	})
}

func (st *scriptTester) noUrban(t testing.TB, _, args string, lineNum int) {
	st.addAction(func(_ context.Context) {
		assert.Assert(t, st.b == nil, "bot has already been created, cannot disable Urban")
		st.bc.Urban = nil
	})
}

func (st *scriptTester) urbanDefine(t testing.TB, _, args string, lineNum int) {
	var call struct {
		Phrase string

		Def string
		Err string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.urban.DefineFunc = func(_ context.Context, s string) (string, error) {
			assert.Equal(t, s, call.Phrase, "line %d", lineNum)

			var err error
			switch call.Err {
			case "ErrNotFound":
				err = urban.ErrNotFound
			case "ErrServerError":
				err = urban.ErrServerError
			case "ErrUnknown":
				err = urban.ErrUnknown
			case "":
			default:
				t.Fatalf("unknown error type %s: line %d", call.Err, lineNum)
			}

			return call.Def, err
		}
	})
}

func (st *scriptTester) simplePlaintext(t testing.TB, _, args string, lineNum int) {
	var call struct {
		URL string

		Body       string
		StatusCode int
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.simple.PlaintextFunc = func(_ context.Context, u string) (string, error) {
			assert.Equal(t, u, call.URL, "line %d", lineNum)

			var err error

			if call.StatusCode == 777 {
				err = errors.New("testing error")
			} else if !apiclient.IsOK(call.StatusCode) {
				err = &apiclient.Error{
					API:        "simple",
					StatusCode: call.StatusCode,
				}
			}

			return call.Body, err
		}
	})
}

func (st *scriptTester) hltbSearch(t testing.TB, _, args string, lineNum int) {
	var call struct {
		Query string

		Game       *hltb.Game
		StatusCode int
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(_ context.Context) {
		st.hltb.SearchGameFunc = func(_ context.Context, query string) (*hltb.Game, error) {
			assert.Equal(t, query, call.Query, "line %d", lineNum)

			var err error

			if call.StatusCode == 777 {
				err = errors.New("testing error")
			} else if !apiclient.IsOK(call.StatusCode) {
				err = &apiclient.Error{
					API:        "hltb",
					StatusCode: call.StatusCode,
				}
			}

			return call.Game, err
		}
	})
}
