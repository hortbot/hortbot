package twitch_test

import (
	"context"
	"errors"
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
		f.t.Fatal("No more client tokens.")
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
		f.t.Fatalf("code %s has nil token", code)
	}
	return tok
}

func (f *fakeTwitch) route() {
	f.mt.RegisterNoResponder(f.notFound)
	f.mt.RegisterResponder("POST", "https://id.twitch.tv/oauth2/token", f.oauth2Token)

	// Kraken API

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/channel", f.krakenChannel)

	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/401$`, httpmock.NewStringResponder(401, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/404$`, httpmock.NewStringResponder(404, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/418$`, httpmock.NewStringResponder(418, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/500$`, httpmock.NewStringResponder(500, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/900$`, httpmock.NewErrorResponder(errTestBadRequest))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/901$`, httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/\d+$`, f.krakenChannelByIDGet)

	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/401$`, httpmock.NewStringResponder(401, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/404$`, httpmock.NewStringResponder(404, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/418$`, httpmock.NewStringResponder(418, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/500$`, httpmock.NewStringResponder(500, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/900$`, httpmock.NewErrorResponder(errTestBadRequest))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/901$`, httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/\d+$`, f.krakenChannelByIDPut)

	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/kraken/users/1234/follows/channels/200", httpmock.NewStringResponder(200, "{}"))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/kraken/users/1234/follows/channels/401", httpmock.NewStringResponder(401, "{}"))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/kraken/users/1234/follows/channels/404", httpmock.NewStringResponder(404, "{}"))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/kraken/users/1234/follows/channels/422", httpmock.NewStringResponder(422, "{}"))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/kraken/users/1234/follows/channels/500", httpmock.NewStringResponder(500, "{}"))
	f.mt.RegisterResponder("PUT", "https://api.twitch.tv/kraken/users/1234/follows/channels/901", httpmock.NewErrorResponder(errTestBadRequest))

	// Helix API

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users", f.helixUsers)

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=foobar", httpmock.NewStringResponder(200, `{"data": [{"id": 1234, "login": "foobar", "display_name": "Foobar"}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=notfound", httpmock.NewStringResponder(404, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=notfound2", httpmock.NewStringResponder(200, `{"data": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=servererror", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=decodeerror", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?login=requesterror", httpmock.NewErrorResponder(errTestBadRequest))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/users?id=1234", httpmock.NewStringResponder(200, `{"data": [{"id": 1234, "login": "foobar", "display_name": "Foobar"}]}`))

	f.mt.RegisterResponder("POST", `=~https://api.twitch.tv/helix/users/follows$`, f.helixUserFollows)

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/helix/moderation/moderators", f.helixModerationModerators)

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

	// TMI API

	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/foobar/chatters", httpmock.NewStringResponder(200, `{"chatter_count": 1234, "chatters": {"broadcaster": ["foobar"], "viewers": ["foo", "bar"]}}`))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/notfound/chatters", httpmock.NewStringResponder(404, ""))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/servererror/chatters", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/badbody/chatters", httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/geterr/chatters", httpmock.NewErrorResponder(errTestBadRequest))
}

func (f *fakeTwitch) oauth2Token(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "POST")
	dumped, _ := httputil.DumpRequest(req, true)

	id := req.FormValue("client_id")
	assert.Equal(f.t, id, clientID)

	secret := req.FormValue("client_secret")
	assert.Equal(f.t, secret, clientSecret)

	grantType := req.FormValue("grant_type")

	if grantType == "client_credentials" {
		tok := f.nextToken()

		return httpmock.NewJsonResponse(200, map[string]interface{}{
			"access_token": tok.AccessToken,
			"expires_in":   int(time.Until(tok.Expiry).Seconds()),
			"token_type":   "bearer",
		})
	}

	if grantType == "authorization_code" {
		code := req.FormValue("code")
		tok := f.tokenForCode(code)

		return httpmock.NewJsonResponse(200, map[string]interface{}{
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
			return httpmock.NewJsonResponse(400, map[string]interface{}{
				"error":   "Bad Request",
				"status":  400,
				"message": "Invalid refresh token",
			})
		case "unknown":
			return httpmock.NewJsonResponse(400, map[string]interface{}{
				"error":   "Bad Request",
				"status":  400,
				"message": "huh",
			})
		case "decodeerror":
			return httpmock.NewStringResponse(400, "}"), nil
		default:
			f.t.Errorf("unknown refresh token: %s", refreshToken)
		}
	}

	f.t.Logf("unknown grant type: %s", grantType)
	return f.dumpAndFail(req, dumped)
}

func (f *fakeTwitch) setChannel(c *twitch.Channel) {
	f.channels[c.ID.AsInt64()] = c
}

func (f *fakeTwitch) krakenChannel(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "GET")
	f.checkHeaders(req, true)

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, "OAuth "))

	tok := strings.TrimPrefix(auth, "OAuth ")
	if tok == "requesterror" {
		return nil, errTestBadRequest
	}

	id, ok := f.tokenToID[tok]
	assert.Assert(f.t, ok)

	c := f.channels[id]
	assert.Assert(f.t, c != nil)

	return httpmock.NewJsonResponse(200, c)
}

func (f *fakeTwitch) krakenChannelByIDGet(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "GET")
	f.checkHeaders(req, true)
	assert.Equal(f.t, req.Header.Get("Authorization"), "OAuth "+f.clientTok.AccessToken)

	path := req.URL.Path
	i := strings.LastIndexByte(path, '/')
	path = path[i+1:]

	id, err := strconv.ParseInt(path, 10, 64)
	assert.NilError(f.t, err)

	c := f.channels[id]

	if c == nil {
		return httpmock.NewStringResponse(404, "{}"), nil
	}

	return httpmock.NewJsonResponse(200, c)
}

func (f *fakeTwitch) krakenChannelByIDPut(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "PUT")
	f.checkHeaders(req, true)

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, "OAuth "))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, "OAuth ")]
	assert.Assert(f.t, ok)

	c := f.channels[id]
	assert.Assert(f.t, c != nil)

	body := &struct {
		Channel struct {
			Status *string
			Game   *string
		}
	}{}

	assert.NilError(f.t, jsonx.DecodeSingle(req.Body, &body))

	switch {
	case body.Channel.Status != nil:
		c.Status = *body.Channel.Status
	case body.Channel.Game != nil:
		c.Game = *body.Channel.Game
	default:
		f.t.Fatal("Nothing changed.")
	}

	return httpmock.NewJsonResponse(200, c)
}

func (f *fakeTwitch) helixUsers(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "GET")
	f.checkHeaders(req, false)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, authPrefix))

	tok := strings.TrimPrefix(auth, authPrefix)
	if tok == "requesterror" {
		return nil, errTestBadRequest
	}

	id, ok := f.tokenToID[tok]
	assert.Assert(f.t, ok)

	c := f.channels[id]
	if c == nil {
		f.t.Fatalf("nil channel for %d", id)
		panic("unreachable")
	}

	switch c.ID {
	case 777:
		return httpmock.NewStringResponse(200, "}"), nil
	case 503:
		return httpmock.NewStringResponse(503, ""), nil
	}

	return httpmock.NewJsonResponse(200, map[string]interface{}{
		"data": []twitch.User{
			{
				ID:          c.ID,
				Name:        c.Name,
				DisplayName: c.DisplayName,
			},
		},
	})
}

func (f *fakeTwitch) setMods(id int64, mods []*twitch.ChannelModerator) {
	f.moderators[id] = mods
}

func (f *fakeTwitch) helixModerationModerators(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "GET")
	f.checkHeaders(req, false)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, authPrefix))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, authPrefix)]
	assert.Assert(f.t, ok)

	q := req.URL.Query()
	gotID, err := strconv.ParseInt(q.Get("broadcaster_id"), 10, 64)
	assert.NilError(f.t, err)

	assert.Equal(f.t, gotID, id)

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
	assert.Assert(f.t, mods != nil)

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
			assert.NilError(f.t, err)
			assert.Assert(f.t, x >= 0)
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

func (f *fakeTwitch) helixUserFollows(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "POST")
	f.checkHeaders(req, false)

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, "Bearer "))

	body := &struct {
		FromID string `json:"from_id"`
		ToID   string `json:"to_id"`
	}{}

	assert.NilError(f.t, jsonx.DecodeSingle(req.Body, &body))

	switch body.ToID {
	case "777":
		return httpmock.NewStringResponse(204, "}"), nil
	case "901":
		return nil, errTestBadRequest
	}

	assert.Equal(f.t, body.FromID, "1234")

	status, err := strconv.Atoi(body.ToID)
	assert.NilError(f.t, err)

	return httpmock.NewStringResponse(status, ""), nil
}

func (f *fakeTwitch) helixChannelsPatch(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "PATCH")
	f.checkHeaders(req, false)

	const authPrefix = "Bearer "

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, authPrefix))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, authPrefix)]
	assert.Assert(f.t, ok)

	body := &struct {
		BroadcasterID twitch.IDStr  `json:"broadcaster_id"`
		Title         *string       `json:"title,omitempty"`
		GameID        *twitch.IDStr `json:"game_id,omitempty"`
	}{}

	assert.NilError(f.t, jsonx.DecodeSingle(req.Body, &body))

	assert.Equal(f.t, int64(body.BroadcasterID), id)

	switch id {
	case 1234:
		assert.Equal(f.t, *body.Title, "some new title")
		assert.Equal(f.t, body.GameID, (*twitch.IDStr)(nil))
		return httpmock.NewStringResponse(204, ""), nil
	case 5678:
		assert.Equal(f.t, body.Title, (*string)(nil))
		assert.Equal(f.t, int64(*body.GameID), int64(9876))
		return httpmock.NewStringResponse(204, ""), nil
	case 900:
		return nil, errTestBadRequest
	}

	return httpmock.NewStringResponse(int(id), ""), nil
}

func (f *fakeTwitch) dumpAndFail(req *http.Request, dumped []byte) (*http.Response, error) {
	f.t.Helper()
	if len(dumped) == 0 {
		dumped, _ = httputil.DumpRequest(req, true)
	}
	f.t.Logf("%s\n", dumped)
	return httpmock.ConnectionFailure(req)
}

func (f *fakeTwitch) checkHeaders(req *http.Request, kraken bool) {
	f.t.Helper()

	assert.Equal(f.t, req.Header.Get("Client-ID"), clientID)
	assert.Equal(f.t, req.Header.Get("Content-Type"), "application/json")

	if kraken {
		assert.Equal(f.t, req.Header.Get("Accept"), "application/vnd.twitchtv.v5+json")
	}
}

func createTester(t *testing.T) (*fakeTwitch, *twitch.Twitch) {
	t.Helper()
	ft := newFakeTwitch(t)
	cli := ft.client()
	tw := twitch.New(clientID, clientSecret, redirectURL, twitch.HTTPClient(cli))
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
