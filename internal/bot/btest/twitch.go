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
		st.twitch.GetChannelByIDFunc = func(_ context.Context, id int64) (*twitch.Channel, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)
			return call.Channel, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchGetUserByUsername(t testing.TB, _, args string, lineNum int) {
	var v map[string]*twitch.User

	err := json.Unmarshal([]byte(args), &v)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.GetUserByUsernameFunc = func(_ context.Context, username string) (*twitch.User, error) {
			u := v[username]
			if u == nil {
				return nil, twitch.ErrNotFound
			}

			assert.Assert(t, u.Name != "")
			return u, nil
		}
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
		st.twitch.ModifyChannelFunc = func(_ context.Context, broadcasterID int64, tok *oauth2.Token, status *string, gameID *int64) (*oauth2.Token, error) {
			assert.Equal(t, broadcasterID, call.ID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(tok, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(status, call.Status), "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(gameID, call.GameID), "line %d", lineNum)

			return call.NewTok, twitchErr(t, lineNum, call.Err)
		}
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
		st.twitch.GetGameByNameFunc = func(_ context.Context, name string) (*twitch.Category, error) {
			call, ok := calls[name]
			assert.Assert(t, ok, `unknown name "%s": line %d`, name, lineNum)

			return call.Category, twitchErr(t, lineNum, call.Err)
		}
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
		st.twitch.GetGameByIDFunc = func(_ context.Context, id int64) (*twitch.Category, error) {
			call, ok := calls[twitch.IDStr(id)]
			assert.Assert(t, ok, `unknown id "%v": line %d`, id, lineNum)

			return call.Category, twitchErr(t, lineNum, call.Err)
		}
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
		st.twitch.SearchCategoriesFunc = func(_ context.Context, query string) ([]*twitch.Category, error) {
			call, ok := calls[query]
			assert.Assert(t, ok, `unknown query "%s": line %d`, query, lineNum)

			return call.Categories, twitchErr(t, lineNum, call.Err)
		}
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
		st.twitch.GetStreamByUserIDFunc = func(_ context.Context, id int64) (*twitch.Stream, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Stream, twitchErr(t, lineNum, call.Err)
		}
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
		st.twitch.GetStreamByUsernameFunc = func(_ context.Context, username string) (*twitch.Stream, error) {
			assert.Equal(t, username, call.Username, "line %d", lineNum)

			return call.Stream, twitchErr(t, lineNum, call.Err)
		}
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
		st.twitch.GetGameLinksFunc = func(_ context.Context, id int64) ([]twitch.GameLink, error) {
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.Links, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchBan(t testing.TB, _, args string, lineNum int) {
	var call struct {
		BroadcasterID int64
		ModID         int64
		Tok           *oauth2.Token
		Req           *twitch.BanRequest

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.BanFunc = func(_ context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, req *twitch.BanRequest) (newToken *oauth2.Token, err error) {
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, modID, call.ModID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(modToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(req, call.Req), "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchUnban(t testing.TB, _, args string, lineNum int) {
	var call struct {
		BroadcasterID int64
		ModID         int64
		Tok           *oauth2.Token
		UserID        int64

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.UnbanFunc = func(_ context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, userID int64) (newToken *oauth2.Token, err error) {
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, modID, call.ModID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(modToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, userID, call.UserID, "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchUpdateChatSettings(t testing.TB, _, args string, lineNum int) {
	var call struct {
		BroadcasterID int64
		ModID         int64
		Tok           *oauth2.Token
		Patch         *twitch.ChatSettingsPatch

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.UpdateChatSettingsFunc = func(_ context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, patch *twitch.ChatSettingsPatch) (newToken *oauth2.Token, err error) {
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, modID, call.ModID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(modToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(patch, call.Patch), "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchSetChatColor(t testing.TB, _, args string, lineNum int) {
	var call struct {
		UserID int64
		Tok    *oauth2.Token
		Color  string

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.SetChatColorFunc = func(_ context.Context, userID int64, userToken *oauth2.Token, color string) (newToken *oauth2.Token, err error) {
			assert.Equal(t, userID, call.UserID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(userToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, color, call.Color, "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchDeleteChatMessage(t testing.TB, _, args string, lineNum int) {
	var call struct {
		BroadcasterID int64
		ModID         int64
		Tok           *oauth2.Token
		ID            string

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.DeleteChatMessageFunc = func(_ context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, id string) (newToken *oauth2.Token, err error) {
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, modID, call.ModID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(modToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, id, call.ID, "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchClearChat(t testing.TB, _, args string, lineNum int) {
	var call struct {
		BroadcasterID int64
		ModID         int64
		Tok           *oauth2.Token

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.ClearChatFunc = func(_ context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token) (newToken *oauth2.Token, err error) {
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, modID, call.ModID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(modToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}

func (st *scriptTester) twitchAnnounce(t testing.TB, _, args string, lineNum int) {
	var call struct {
		BroadcasterID int64
		ModID         int64
		Tok           *oauth2.Token
		Message       string
		Color         string

		NewToken *oauth2.Token
		Err      string
	}

	err := json.Unmarshal([]byte(args), &call)
	assert.NilError(t, err, "line %d", lineNum)

	st.addAction(func(ctx context.Context) {
		st.twitch.AnnounceFunc = func(_ context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, message string, color string) (newToken *oauth2.Token, err error) {
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, modID, call.ModID, "line %d", lineNum)
			assert.Assert(t, cmp.DeepEqual(modToken, call.Tok, tokenCmp), "line %d", lineNum)
			assert.Equal(t, broadcasterID, call.BroadcasterID, "line %d", lineNum)
			assert.Equal(t, message, call.Message, "line %d", lineNum)
			assert.Equal(t, color, call.Color, "line %d", lineNum)

			return call.NewToken, twitchErr(t, lineNum, call.Err)
		}
	})
}
