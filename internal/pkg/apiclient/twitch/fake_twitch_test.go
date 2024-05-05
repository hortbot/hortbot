package twitch_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"github.com/hortbot/hortbot/internal/pkg/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/zikaeroh/ctxlog"
	"golang.org/x/oauth2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

var errTestBadRequest = errors.New("twitch_test: bad request")

type fakeTwitch struct {
	t        testing.TB
	mt       *httpmock.MockTransport
	notFound httpmock.Responder

	clientTok    *oauth2.Token
	clientTokens []*oauth2.Token

	idToCode    map[int64]string
	codeToToken map[string]*oauth2.Token
	tokenToID   map[string]int64

	channels   map[int64]*twitch.Channel
	moderators map[int64][]*twitch.ChannelModerator
	moderated  map[int64][]*twitch.ModeratedChannel
}

func newFakeTwitch(t testing.TB) *fakeTwitch {
	f := &fakeTwitch{
		t:           t,
		mt:          httpmock.NewMockTransport(),
		notFound:    httpmock.NewNotFoundResponder(t.Fatal),
		idToCode:    make(map[int64]string),
		codeToToken: make(map[string]*oauth2.Token),
		tokenToID:   make(map[string]int64),
		channels:    make(map[int64]*twitch.Channel),
		moderators:  make(map[int64][]*twitch.ChannelModerator),
		moderated:   make(map[int64][]*twitch.ModeratedChannel),
	}

	f.route()

	return f
}

func (f *fakeTwitch) client() *http.Client {
	if f.mt == nil {
		panic("MockTransport unset")
	}
	return &http.Client{Transport: f.mt}
}

func (f *fakeTwitch) nextToken() *oauth2.Token {
	f.t.Helper()
	if len(f.clientTokens) == 0 {
		f.t.Error("No more client tokens.")
		panic("failed assertion")
	}

	f.clientTok, f.clientTokens = f.clientTokens[0], f.clientTokens[1:]
	return f.clientTok
}

func (f *fakeTwitch) setClientTokens(tokens ...*oauth2.Token) {
	f.clientTokens = tokens
}

func (f *fakeTwitch) codeForUserAux(id int64, forceRefresh string) string {
	f.t.Helper()
	code, ok := f.idToCode[id]
	if ok {
		return code
	}

	code = uuid.Must(uuid.NewV4()).String()
	f.idToCode[id] = code

	tok := &oauth2.Token{
		AccessToken:  uuid.Must(uuid.NewV4()).String(),
		TokenType:    "bearer",
		RefreshToken: uuid.Must(uuid.NewV4()).String(),
		Expiry:       time.Now().Add(time.Hour).Round(time.Second),
	}

	if forceRefresh != "" {
		tok.RefreshToken = forceRefresh
		tok.Expiry = time.Now().Add(15 * time.Second)
	}

	f.codeToToken[code] = tok
	f.tokenToID[tok.AccessToken] = id

	return code
}

func (f *fakeTwitch) codeForUser(id int64) string {
	f.t.Helper()
	return f.codeForUserAux(id, "")
}

func (f *fakeTwitch) codeForUserInvalidRefresh(id int64, forceRefresh string) string {
	f.t.Helper()
	return f.codeForUserAux(id, forceRefresh)
}

func (f *fakeTwitch) tokenForCode(code string) *oauth2.Token {
	f.t.Helper()
	tok := f.codeToToken[code]
	if tok == nil {
		f.fatalf("code %s has nil token", code)
	}
	return tok
}

func (f *fakeTwitch) route() {
	f.mt.RegisterNoResponder(f.notFound)
	f.mt.RegisterResponder("POST", "https://id.twitch.tv/oauth2/token", f.oauth2Token)

	// Helix API

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users", f.helixUsers)

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=foobar", httpmock.NewStringResponder(200, `{"data": [{"id": 1234, "login": "foobar", "display_name": "Foobar"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=notfound", httpmock.NewStringResponder(404, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=notfound2", httpmock.NewStringResponder(200, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=servererror", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=decodeerror", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=requesterror", httpmock.NewErrorResponder(errTestBadRequest))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?id=1234", httpmock.NewStringResponder(200, `{"data": [{"id": 1234, "login": "foobar", "display_name": "Foobar"}]}`))

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/moderation/moderators", f.helixModerationModerators)
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/moderation/channels", f.helixModerationChannels)

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/search/categories?query=pubg", httpmock.NewStringResponder(200, `{"data": [{"id": "287491", "name": "PLAYERUNKNOWN's BATTLEGROUNDS"}, {"id": "58730284", "name": "PUBG MOBILE"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/search/categories?query=notfound", httpmock.NewStringResponder(200, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/search/categories?query=notfound2", httpmock.NewStringResponder(404, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/search/categories?query=servererror", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/search/categories?query=decodeerror", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/search/categories?query=requesterror", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/games?name=PLAYERUNKNOWN%27s+BATTLEGROUNDS", httpmock.NewStringResponder(200, `{"data": [{"id": "287491", "name": "PLAYERUNKNOWN's BATTLEGROUNDS"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/games?id=287491", httpmock.NewStringResponder(200, `{"data": [{"id": "287491", "name": "PLAYERUNKNOWN's BATTLEGROUNDS"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/games?name=notfound", httpmock.NewStringResponder(200, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/games?name=servererror", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/games?name=decodeerror", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/games?name=requesterror", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("PATCH", "https://api.twitch.tv/helix/channels", f.helixChannelsPatch)

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_id=1234", httpmock.NewStringResponder(200, `{"data": [{"id": "512301723123", "user_id": "1234", "user_name": "FooBar", "game_id": "847362", "title": "This is the title.", "viewer_count": 4321, "started_at": "2017-08-14T16:08:32Z"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_login=foobar", httpmock.NewStringResponder(200, `{"data": [{"id": "512301723123", "user_id": "1234", "user_name": "FooBar", "game_id": "847362", "title": "This is the title.", "viewer_count": 4321, "started_at": "2017-08-14T16:08:32Z"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_login=notfound", httpmock.NewStringResponder(404, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_login=notfound2", httpmock.NewStringResponder(200, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_login=servererror", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_login=decodeerror", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/streams?user_login=requesterror", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/channels?broadcaster_id=1234", httpmock.NewStringResponder(200, `{"data": [{"broadcaster_id": "1234", "broadcaster_name": "foobar", "game_id": "58730284", "game_name": "PUBG MOBILE", "title": "This is the title."}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/channels?broadcaster_id=404", httpmock.NewStringResponder(404, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/channels?broadcaster_id=444", httpmock.NewStringResponder(200, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/channels?broadcaster_id=500", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/channels?broadcaster_id=900", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/channels?broadcaster_id=901", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("POST", `=~https://api.twitch.tv/helix/moderation/bans$`, f.helixModerationBans)

	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/bans?broadcaster_id=1234&moderator_id=3141&user_id=666", httpmock.NewStringResponder(200, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/bans?broadcaster_id=401&moderator_id=3141&user_id=666", httpmock.NewStringResponder(401, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/bans?broadcaster_id=404&moderator_id=3141&user_id=666", httpmock.NewStringResponder(404, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/bans?broadcaster_id=418&moderator_id=3141&user_id=666", httpmock.NewStringResponder(418, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/bans?broadcaster_id=500&moderator_id=3141&user_id=666", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/bans?broadcaster_id=777&moderator_id=3141&user_id=666", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/helix/chat/color?user_id=1234&color=%239146FF", httpmock.NewStringResponder(200, ``))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/helix/chat/color?user_id=401&color=%239146FF", httpmock.NewStringResponder(401, ``))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/helix/chat/color?user_id=404&color=%239146FF", httpmock.NewStringResponder(404, ``))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/helix/chat/color?user_id=418&color=%239146FF", httpmock.NewStringResponder(418, ``))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/helix/chat/color?user_id=500&color=%239146FF", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/helix/chat/color?user_id=777&color=%239146FF", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=1234&moderator_id=3141&message_id=somemessage", httpmock.NewStringResponder(200, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=401&moderator_id=3141&message_id=somemessage", httpmock.NewStringResponder(401, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=404&moderator_id=3141&message_id=somemessage", httpmock.NewStringResponder(404, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=418&moderator_id=3141&message_id=somemessage", httpmock.NewStringResponder(418, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=500&moderator_id=3141&message_id=somemessage", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=777&moderator_id=3141&message_id=somemessage", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=1234&moderator_id=3141", httpmock.NewStringResponder(200, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=401&moderator_id=3141", httpmock.NewStringResponder(401, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=404&moderator_id=3141", httpmock.NewStringResponder(404, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=418&moderator_id=3141", httpmock.NewStringResponder(418, ``))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=500&moderator_id=3141", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("DELETE", "https://api.twitch.tv/helix/moderation/chat?broadcaster_id=777&moderator_id=3141", httpmock.NewErrorResponder(errTestBadRequest))

	f.mt.RegisterResponder("PATCH", `=~https://api.twitch.tv/helix/chat/settings$`, f.helixChatSettings)
	f.mt.RegisterResponder("POST", `=~https://api.twitch.tv/helix/chat/announcements$`, f.helixChatAnnouncements)

	f.mt.RegisterResponder("POST", "https://api.igdb.com/v4/games", f.igdbGames)

	// TMI API

	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/foobar/chatters", httpmock.NewStringResponder(200, `{"chatter_count": 1234, "chatters": {"broadcaster": ["foobar"], "viewers": ["foo", "bar"]}}`))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/notfound/chatters", httpmock.NewStringResponder(404, ""))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/servererror/chatters", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/badbody/chatters", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/geterr/chatters", httpmock.NewErrorResponder(errTestBadRequest))
}

// Helpers for use in responders, which are run in a different goroutine so cannot
// call runtime.Goexit().

func (f *fakeTwitch) assert(comparison assert.BoolOrComparison, msgAndArgs ...any) {
	// Workaround for httpmock running responders in a separate goroutine.
	// If t.Fail is called from a responder, the test hangs.
	f.t.Helper()
	if !assert.Check(f.t, comparison, msgAndArgs...) {
		panic("failed assertion")
	}
}

func (f *fakeTwitch) assertEqual(x, y any, msgAndArgs ...any) {
	f.t.Helper()
	f.assert(cmp.Equal(x, y), msgAndArgs...)
}

func (f *fakeTwitch) assertNilError(err error, msgAndArgs ...any) {
	f.t.Helper()
	f.assert(err, msgAndArgs...)
}

func (f *fakeTwitch) fatalf(format string, args ...any) {
	f.t.Helper()
	f.t.Errorf(format, args...)
	panic("test failed")
}

func (f *fakeTwitch) oauth2Token(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "POST")
	dumped, _ := httputil.DumpRequest(req, true)

	id := req.FormValue("client_id")
	f.assertEqual(id, clientID)

	secret := req.FormValue("client_secret")
	f.assertEqual(secret, clientSecret)

	grantType := req.FormValue("grant_type")

	if grantType == "client_credentials" {
		tok := f.nextToken()

		return httpmock.NewJsonResponse(200, map[string]any{
			"access_token": tok.AccessToken,
			"expires_in":   int(time.Until(tok.Expiry).Seconds()),
			"token_type":   "bearer",
		})
	}

	if grantType == "authorization_code" {
		code := req.FormValue("code")
		tok := f.tokenForCode(code)

		return httpmock.NewJsonResponse(200, map[string]any{
			"access_token":  tok.AccessToken,
			"refresh_token": tok.RefreshToken,
			"expires_in":    int(time.Until(tok.Expiry).Seconds()),
			"token_type":    tok.TokenType,
		})
	}

	if grantType == "refresh_token" {
		// Currently this only tests the "error" case where this refresh is happening.
		refreshToken := req.FormValue("refresh_token")
		switch refreshToken {
		case "invalid":
			return httpmock.NewJsonResponse(400, map[string]any{
				"error":   "Bad Request",
				"status":  400,
				"message": "Invalid refresh token",
			})
		case "unknown":
			return httpmock.NewJsonResponse(400, map[string]any{
				"error":   "Bad Request",
				"status":  400,
				"message": "huh",
			})
		case "decodeerror":
			return httpmock.NewStringResponse(400, "}"), nil
		default:
			f.fatalf("unknown refresh token: %s", refreshToken)
			return f.dumpAndFail(req, dumped)
		}
	}

	f.t.Logf("unknown grant type: %s", grantType)
	return f.dumpAndFail(req, dumped)
}

func (f *fakeTwitch) setChannel(c *twitch.Channel) {
	f.channels[c.ID.AsInt64()] = c
}

func (f *fakeTwitch) helixUsers(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "GET")
	f.checkHeaders(req)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, authPrefix))

	tok := strings.TrimPrefix(auth, authPrefix)
	if tok == "requesterror" {
		return nil, errTestBadRequest
	}

	id, ok := f.tokenToID[tok]
	f.assert(ok)

	c := f.channels[id]
	if c == nil {
		f.t.Errorf("nil channel for %d", id)
		panic("failed assertion")
	}

	switch c.ID {
	case 777:
		return httpmock.NewStringResponse(200, "}"), nil
	case 503:
		return httpmock.NewStringResponse(503, ""), nil
	}

	return httpmock.NewJsonResponse(200, map[string]any{
		"data": []twitch.User{
			{
				ID:          c.ID,
				Name:        strings.ToLower(c.Name), // TODO: actually test name/display name in Helix API.
				DisplayName: c.Name,
			},
		},
	})
}

func (f *fakeTwitch) setMods(id int64, mods []*twitch.ChannelModerator) {
	f.moderators[id] = mods
}

func (f *fakeTwitch) helixModerationModerators(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "GET")
	f.checkHeaders(req)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, authPrefix))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, authPrefix)]
	f.assert(ok)

	q := req.URL.Query()
	gotID, err := strconv.ParseInt(q.Get("broadcaster_id"), 10, 64)
	f.assertNilError(err)

	f.assertEqual(gotID, id)

	first, err := strconv.ParseInt(q.Get("first"), 10, 64)
	f.assertNilError(err)
	f.assertEqual(first, int64(100))

	if _, ok := expectedErrors[int(id)]; ok {
		return httpmock.NewStringResponse(int(id), ""), nil
	}

	switch id {
	case 777:
		return nil, errTestBadRequest
	case 888:
		return httpmock.NewStringResponse(200, "{"), nil
	}

	mods := f.moderators[id]
	f.assert(mods != nil)

	var v struct {
		Mods       []*twitch.ChannelModerator `json:"data"`
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}

	if len(mods) != 0 {
		i := 0
		if after := q.Get("after"); after != "" {
			x, err := strconv.Atoi(q.Get("after"))
			f.assertNilError(err)
			f.assert(x >= 0)
			i = x + 1
		}

		cursor := ""
		if i != len(mods)-1 || id != 999 {
			cursor = strconv.Itoa(i)
		}

		v.Pagination.Cursor = cursor

		// Allow this to go past to handle the 999 case, for testing no-change pagination.
		if i < len(mods) {
			m := mods[i]
			v.Mods = []*twitch.ChannelModerator{m}
		}
	}

	return httpmock.NewJsonResponse(200, &v)
}

func (f *fakeTwitch) setModerated(id int64, mods []*twitch.ModeratedChannel) {
	f.moderated[id] = mods
}

func (f *fakeTwitch) helixModerationChannels(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "GET")
	f.checkHeaders(req)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, authPrefix))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, authPrefix)]
	f.assert(ok)

	q := req.URL.Query()
	gotID, err := strconv.ParseInt(q.Get("user_id"), 10, 64)
	f.assertNilError(err)

	f.assertEqual(gotID, id)

	first, err := strconv.ParseInt(q.Get("first"), 10, 64)
	f.assertNilError(err)
	f.assertEqual(first, int64(100))

	if _, ok := expectedErrors[int(id)]; ok {
		return httpmock.NewStringResponse(int(id), ""), nil
	}

	switch id {
	case 777:
		return nil, errTestBadRequest
	case 888:
		return httpmock.NewStringResponse(200, "{"), nil
	}

	mods := f.moderated[id]
	f.assert(mods != nil)

	var v struct {
		Mods       []*twitch.ModeratedChannel `json:"data"`
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}

	if len(mods) != 0 {
		i := 0
		if after := q.Get("after"); after != "" {
			x, err := strconv.Atoi(q.Get("after"))
			f.assertNilError(err)
			f.assert(x >= 0)
			i = x + 1
		}

		cursor := ""
		if i != len(mods)-1 || id != 999 {
			cursor = strconv.Itoa(i)
		}

		v.Pagination.Cursor = cursor

		// Allow this to go past to handle the 999 case, for testing no-change pagination.
		if i < len(mods) {
			m := mods[i]
			v.Mods = []*twitch.ModeratedChannel{m}
		}
	}

	return httpmock.NewJsonResponse(200, &v)
}

func (f *fakeTwitch) helixChannelsPatch(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "PATCH")
	f.checkHeaders(req)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, authPrefix))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, authPrefix)]
	f.assert(ok)

	body := &struct {
		BroadcasterID twitch.IDStr  `json:"broadcaster_id"`
		Title         *string       `json:"title,omitempty"`
		GameID        *twitch.IDStr `json:"game_id,omitempty"`
	}{}

	f.assertNilError(jsonx.DecodeSingle(req.Body, &body))

	f.assertEqual(int64(body.BroadcasterID), id)

	switch id {
	case 1234:
		f.assertEqual(*body.Title, "some new title")
		f.assertEqual(body.GameID, (*twitch.IDStr)(nil))
		return httpmock.NewStringResponse(204, ""), nil
	case 5678:
		f.assertEqual(body.Title, (*string)(nil))
		f.assertEqual(int64(*body.GameID), int64(9876))
		return httpmock.NewStringResponse(204, ""), nil
	case 900:
		return nil, errTestBadRequest
	}

	return httpmock.NewStringResponse(int(id), ""), nil
}

const igdbSuccessResponse = `[
    {
        "id": 121084,
        "external_games": [
            {
                "id": 1728356,
                "category": 1,
                "url": "https://store.steampowered.com/app/1119980"
            },
            {
                "id": 1883089,
                "category": 14,
                "url": "https://www.twitch.tv/directory/game/In%20Sound%20Mind"
            },
            {
                "id": 2070779,
                "category": 5,
                "url": "https://www.gog.com/game/in_sound_mind"
            },
            {
                "id": 2071943,
                "category": 20,
                "url": "https://amazon.com/dp/B0987SBLFL"
            },
            {
                "id": 2071976,
                "category": 20,
                "url": "https://amazon.de/dp/B098BLJDMY"
            },
            {
                "id": 2071978,
                "category": 20,
                "url": "https://amazon.co.uk/dp/B098BJY9YK"
            },
            {
                "id": 2072070,
                "category": 20,
                "url": "https://amazon.fr/dp/B098BR9T58"
            },
            {
                "id": 2124403,
                "category": 26,
                "url": "https://www.epicgames.com/store/p/in-sound-mind"
            },
            {
                "id": 2125622,
                "category": 11,
                "url": "https://www.microsoft.com/en-us/p/-1-/9PKDNVFFZCWX"
            }
        ],
        "websites": [
            {
                "id": 114698,
                "category": 13,
                "url": "https://store.steampowered.com/app/1119980"
            },
            {
                "id": 148717,
                "category": 5,
                "url": "https://twitter.com/insoundmindgame"
            },
            {
                "id": 182350,
                "category": 17,
                "url": "https://www.gog.com/game/in_sound_mind"
            },
            {
                "id": 215595,
                "category": 1,
                "url": "https://modusgames.com/in-sound-mind"
            },
            {
                "id": 228911,
                "category": 16,
                "url": "https://www.epicgames.com/store/p/in-sound-mind"
            },
            {
                "id": 228912,
                "category": 6,
                "url": "https://www.twitch.tv/modus_games"
            },
            {
                "id": 228913,
                "category": 18,
                "url": "https://discord.gg/modus"
            }
        ]
    }
]`

const igdbNoGamesResponse = `[
    {
        "id": 121084,
        "external_games": [
            {
                "id": 2071943,
                "category": 20,
                "url": "https://amazon.com/dp/B0987SBLFL"
            },
            {
                "id": 2071976,
                "category": 20,
                "url": "https://amazon.de/dp/B098BLJDMY"
            },
            {
                "id": 2071978,
                "category": 20,
                "url": "https://amazon.co.uk/dp/B098BJY9YK"
            },
            {
                "id": 2072070,
                "category": 20,
                "url": "https://amazon.fr/dp/B098BR9T58"
            },
            {
                "id": 2125622,
                "category": 11,
                "url": "https://www.microsoft.com/en-us/p/-1-/9PKDNVFFZCWX"
            }
        ],
        "websites": [
            {
                "id": 228912,
                "category": 6,
                "url": "https://www.twitch.tv/modus_games"
            },
            {
                "id": 228913,
                "category": 18,
                "url": "https://discord.gg/modus"
            }
        ]
    }
]`

func (f *fakeTwitch) igdbGames(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "POST")
	f.checkHeaders(req)

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, "Bearer "))

	body, err := io.ReadAll(req.Body)
	f.assertNilError(err)
	bodyString := string(body)

	switch bodyString {
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "518088"; limit 1;`:
		return httpmock.NewStringResponse(200, igdbSuccessResponse), nil
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "4040"; limit 1;`:
		return httpmock.NewStringResponse(200, `[{"id": 1234, "external_games": []}]`), nil
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "4041"; limit 1;`:
		return httpmock.NewStringResponse(200, `[]`), nil
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "404"; limit 1;`:
		return httpmock.NewStringResponse(404, ""), nil
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "777"; limit 1;`:
		return httpmock.NewStringResponse(200, igdbNoGamesResponse), nil
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "500"; limit 1;`:
		return httpmock.NewStringResponse(500, ""), nil
	case `fields websites.category, websites.url, external_games.category, external_games.url; where external_games.category = 14 & external_games.uid = "700"; limit 1;`:
		return httpmock.NewStringResponse(200, "{"), nil
	default:
		return nil, errTestBadRequest
	}
}

func (f *fakeTwitch) helixModerationBans(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "POST")
	f.checkHeaders(req)

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, "Bearer "))

	q := req.URL.Query()
	broadcasterID, err := strconv.ParseInt(q.Get("broadcaster_id"), 10, 64)
	f.assertNilError(err)
	moderatorID, err := strconv.ParseInt(q.Get("moderator_id"), 10, 64)
	f.assertNilError(err)

	bodyBytes, err := io.ReadAll(req.Body)
	f.assertNilError(err)

	var body struct {
		Data *twitch.BanRequest `json:"data"`
	}

	f.assertNilError(jsonx.DecodeSingle(bytes.NewReader(bodyBytes), &body))

	if body.Data.Duration == 0 {
		f.assert(!bytes.Contains(bodyBytes, []byte("duration")))
	}

	switch {
	case broadcasterID == 1 && moderatorID == 123:
		f.assertEqual(body.Data.UserID.AsInt64(), int64(666))
		f.assertEqual(body.Data.Duration, int64(30))
		f.assertEqual(body.Data.Reason, "Broke a rule.")
	case broadcasterID == 404:
		return httpmock.NewStringResponse(404, ""), nil
	case broadcasterID == 401:
		return httpmock.NewStringResponse(401, ""), nil
	case broadcasterID == 418:
		return httpmock.NewStringResponse(418, ""), nil
	case broadcasterID == 500:
		return httpmock.NewStringResponse(500, ""), nil
	default:
		return nil, errTestBadRequest
	}

	return httpmock.NewStringResponse(200, "{}"), nil
}

func (f *fakeTwitch) helixChatSettings(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "PATCH")
	f.checkHeaders(req)

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, "Bearer "))

	q := req.URL.Query()
	broadcasterID, err := strconv.ParseInt(q.Get("broadcaster_id"), 10, 64)
	f.assertNilError(err)
	moderatorID, err := strconv.ParseInt(q.Get("moderator_id"), 10, 64)
	f.assertNilError(err)

	bodyBytes, err := io.ReadAll(req.Body)
	f.assertNilError(err)

	var body twitch.ChatSettingsPatch

	f.assertNilError(jsonx.DecodeSingle(bytes.NewReader(bodyBytes), &body))

	switch {
	case broadcasterID == 1 && moderatorID == 123:
		f.assertEqual(*body.EmoteMode, true)
	case broadcasterID == 404:
		return httpmock.NewStringResponse(404, ""), nil
	case broadcasterID == 401:
		return httpmock.NewStringResponse(401, ""), nil
	case broadcasterID == 418:
		return httpmock.NewStringResponse(418, ""), nil
	case broadcasterID == 500:
		return httpmock.NewStringResponse(500, ""), nil
	default:
		return nil, errTestBadRequest
	}

	return httpmock.NewStringResponse(200, "{}"), nil
}

func (f *fakeTwitch) helixChatAnnouncements(req *http.Request) (*http.Response, error) {
	f.assertEqual(req.Method, "POST")
	f.checkHeaders(req)

	auth := req.Header.Get("Authorization")
	f.assert(strings.HasPrefix(auth, "Bearer "))

	q := req.URL.Query()
	broadcasterID, err := strconv.ParseInt(q.Get("broadcaster_id"), 10, 64)
	f.assertNilError(err)
	moderatorID, err := strconv.ParseInt(q.Get("moderator_id"), 10, 64)
	f.assertNilError(err)

	bodyBytes, err := io.ReadAll(req.Body)
	f.assertNilError(err)

	var body struct {
		Message string `json:"message"`
		Color   string `json:"color,omitempty"`
	}

	f.assertNilError(jsonx.DecodeSingle(bytes.NewReader(bodyBytes), &body))

	switch {
	case broadcasterID == 1 && moderatorID == 123:
		f.assertEqual(body.Message, "Some announcement!")
		f.assertEqual(body.Color, "purple")
	case broadcasterID == 404:
		return httpmock.NewStringResponse(404, ""), nil
	case broadcasterID == 401:
		return httpmock.NewStringResponse(401, ""), nil
	case broadcasterID == 418:
		return httpmock.NewStringResponse(418, ""), nil
	case broadcasterID == 500:
		return httpmock.NewStringResponse(500, ""), nil
	default:
		return nil, errTestBadRequest
	}

	return httpmock.NewStringResponse(200, "{}"), nil
}

func (f *fakeTwitch) dumpAndFail(req *http.Request, dumped []byte) (*http.Response, error) {
	f.t.Helper()
	if len(dumped) == 0 {
		dumped, _ = httputil.DumpRequest(req, true)
	}
	f.t.Logf("%s\n", dumped)
	f.t.Fail()
	return httpmock.ConnectionFailure(req)
}

func (f *fakeTwitch) checkHeaders(req *http.Request) {
	f.t.Helper()

	f.assertEqual(req.Header.Get("Client-Id"), clientID)
	f.assertEqual(req.Header.Get("Content-Type"), "application/json")
}

func createTester(t *testing.T) (*fakeTwitch, *twitch.Twitch) {
	t.Helper()
	ft := newFakeTwitch(t)
	cli := ft.client()
	tw := twitch.New(clientID, clientSecret, redirectURL, cli)
	return ft, tw
}

func testContext(t testing.TB) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx := ctxlog.WithLogger(context.Background(), testutil.Logger(t))
	return context.WithTimeout(ctx, 10*time.Minute)
}

func strPtr(s string) *string {
	return &s
}

func int64Ptr(x int64) *int64 {
	return &x
}
