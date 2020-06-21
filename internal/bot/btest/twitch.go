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

func (st *scriptTester) twitchSetChannel(t testing.TB, directive, args string, lineNum int) {
	var call struct {
		ID  int64
		Tok *oauth2.Token
		New string

		Set    string
		NewTok *oauth2.Token
		Err    string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		fn := st.twitch.SetChannelGameCalls
		if directive == "twitch_set_channel_status" {
			fn = st.twitch.SetChannelStatusCalls
		}

		fn(func(_ context.Context, id int64, tok *oauth2.Token, n string) (string, *oauth2.Token, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(tok, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, n, call.New, "line %d", lineNum)

			return call.Set, call.NewTok, twitchErr(t, lineNum, call.Err)
		})
	})
}

func (st *scriptTester) twitchGetCurrentStream(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID int64

		Stream *twitch.Stream
		Err    string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetCurrentStreamCalls(func(_ context.Context, id int64) (*twitch.Stream, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)
			return call.Stream, twitchErr(t, lineNum, call.Err)
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

func (st *scriptTester) twitchFollowChannel(t testing.TB, _, args string, lineNum int) {
	var call struct {
		ID       int64
		Tok      *oauth2.Token
		ToFollow int64

		NewTok *oauth2.Token
		Err    string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.FollowChannelCalls(func(_ context.Context, id int64, tok *oauth2.Token, toFollow int64) (*oauth2.Token, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(tok, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, toFollow, call.ToFollow, "line %d", lineNum)

			return call.NewTok, twitchErr(t, lineNum, call.Err)
		})
	})
}
