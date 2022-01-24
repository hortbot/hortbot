package btest

import (
	"context"
	"encoding/json"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

var tokenCmp = gocmp.Comparer(func(x, y oauth2.Token) bool {
	return oauth2x.Equals(&x, &y)
})

func twitchErr(t testing.TB, lineNum int, e string) error {
	switch e {
	case "":
		return nil
	case "ErrNotFound":
		return twitch.ErrNotFound
	case "ErrNotAuthorized":
		return twitch.ErrNotAuthorized
	case "ErrServerError":
		return twitch.ErrServerError
	case "ErrUnknown":
		return twitch.ErrUnknown
	default:
		t.Fatalf("unknown error type %s: line %d", e, lineNum)
		return nil
	}
}

func (st *scriptTester) twitchGetChannelByID(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID int64

		Channel *twitch.Channel
		Err     string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetChannelByIDCalls(func(_ context.Context, id int64) (*twitch.Channel, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)
			return call.Channel, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetChatters(t testing.TB, _, args string, lineNum int) {
	var call struct {
		Channel string

		Chatters *twitch.Chatters
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetChattersCalls(func(_ context.Context, channel string) (*twitch.Chatters, error) {
			assert.Equal(t, channel, call.Channel, "line %d", lineNum)
			return call.Chatters, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetUserByUsername(t testing.TB, _, args string, lineNum int) {
	var v map[string]*twitch.User

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetUserByUsernameCalls(func(_ context.Context, username string) (*twitch.User, error) {
			u := v[username]
			if u == nil {
				return nil, twitch.ErrNotFound
			}

			assert.Assert(t, u.Name != "")
			return u, nil
		})
	})
}

func (st *scriptTester) twitchModifyChannel(t testing.TB, directive, args string, lineNum int) {
	var call struct {
		ID     int64
		Tok    *oauth2.Token
		Status *string
		GameID *int64

		NewTok *oauth2.Token
		Err    string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.ModifyChannelCalls(func(_ context.Context, broadcasterID int64, tok *oauth2.Token, status *string, gameID *int64) (*oauth2.Token, error) {
			assert.Equal(t, broadcasterID, call.ID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(tok, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(status, call.Status), "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(gameID, call.GameID), "line %d", lineNum)

			return call.NewTok, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetGameByName(t testing.TB, directive, args string, lineNum int) {
	var calls map[string]struct {
		Category *twitch.Category
		Err      string
	}

	err := json.Unmarshal([]byte(args), &calls)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetGameByNameCalls(func(_ context.Context, name string) (*twitch.Category, error) {
			call, ok := calls[name]
			assert.Assert(t, ok, `unknown name "%s": line %d`, name, lineNum)

			return call.Category, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetGameByID(t testing.TB, directive, args string, lineNum int) {
	var calls map[twitch.IDStr]struct {
		Category *twitch.Category
		Err      string
	}

	err := json.Unmarshal([]byte(args), &calls)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetGameByIDCalls(func(_ context.Context, id int64) (*twitch.Category, error) {
			call, ok := calls[twitch.IDStr(id)]
			assert.Assert(t, ok, `unknown id "%v": line %d`, id, lineNum)

			return call.Category, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchSearchCategories(t testing.TB, directive, args string, lineNum int) {
	var calls map[string]struct {
		Categories []*twitch.Category
		Err        string
	}

	err := json.Unmarshal([]byte(args), &calls)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.SearchCategoriesCalls(func(_ context.Context, query string) ([]*twitch.Category, error) {
			call, ok := calls[query]
			assert.Assert(t, ok, `unknown query "%s": line %d`, query, lineNum)

			return call.Categories, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetStreamByUserID(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID int64

		Stream *twitch.Stream
		Err    string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetStreamByUserIDCalls(func(_ context.Context, id int64) (*twitch.Stream, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Stream, twitchErr(t, lineNum, call.Err)
		})
	})
}
func (st *scriptTester) twitchGetStreamByUsername(t testing.TB, _, args string, lineNum int) {
	var call struct {
		Username string

		Stream *twitch.Stream
		Err    string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetStreamByUsernameCalls(func(_ context.Context, username string) (*twitch.Stream, error) {
			assert.Equal(t, username, call.Username, "line %d", lineNum)

			return call.Stream, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetGameLinks(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID int64

		Links []twitch.GameLink
		Err   string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetGameLinksCalls(func(_ context.Context, id int64) ([]twitch.GameLink, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Links, twitchErr(t, lineNum, call.Err)
		})
	})
}
