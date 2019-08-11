package twitch_test

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/jarcoal/httpmock"
	"golang.org/x/oauth2"
	"gotest.tools/assert"
)

type fakeTwitch struct {
	t        testing.TB
	mt       *httpmock.MockTransport
	notFound httpmock.Responder

	clientTok    *oauth2.Token
	clientTokens []*oauth2.Token

	idToCode    map[int64]string
	codeToToken map[string]*oauth2.Token
	tokenToID   map[string]int64

	channels map[int64]*twitch.Channel
	streams  map[int64]*twitch.Stream
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
		streams:     make(map[int64]*twitch.Stream),
	}

	f.route()

	return f
}

func (f *fakeTwitch) client() *http.Client {
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

func (f *fakeTwitch) codeForUser(id int64) string {
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

	f.codeToToken[code] = tok
	f.tokenToID[tok.AccessToken] = id

	return code
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
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/channel", f.krakenChannel)

	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/401$`, httpmock.NewStringResponder(401, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/404$`, httpmock.NewStringResponder(404, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/418$`, httpmock.NewStringResponder(418, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/500$`, httpmock.NewStringResponder(500, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/channels/\d+$`, f.krakenChannelByIDGet)

	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/401$`, httpmock.NewStringResponder(401, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/404$`, httpmock.NewStringResponder(404, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/418$`, httpmock.NewStringResponder(418, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/500$`, httpmock.NewStringResponder(500, "{}"))
	f.mt.RegisterResponder("PUT", `=~https://api.twitch.tv/kraken/channels/\d+$`, f.krakenChannelByIDPut)

	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/streams/401$`, httpmock.NewStringResponder(401, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/streams/404$`, httpmock.NewStringResponder(404, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/streams/418$`, httpmock.NewStringResponder(418, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/streams/500$`, httpmock.NewStringResponder(500, "{}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/streams/777$`, httpmock.NewStringResponder(200, "}"))
	f.mt.RegisterResponder("GET", `=~https://api.twitch.tv/kraken/streams/\d+$`, f.krakenStreamByID)

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken", f.kraken)

	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/foobar/chatters", httpmock.NewStringResponder(200, `{"chatter_count": 1234, "chatters": {"broadcaster": ["foobar"], "viewers": ["foo", "bar"]}}`))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/notfound/chatters", httpmock.NewStringResponder(404, ""))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/servererror/chatters", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://tmi.twitch.tv/group/user/badbody/chatters", httpmock.NewStringResponder(200, "}"))

	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/users?login=foobar", httpmock.NewStringResponder(200, `{"users": [{"_id": 1234}]}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/users?login=notfound", httpmock.NewStringResponder(404, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/users?login=notfound2", httpmock.NewStringResponder(200, `{"users": []}`))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/users?login=servererror", httpmock.NewStringResponder(500, ""))
	f.mt.RegisterResponder("GET", "https://api.twitch.tv/kraken/users?login=decodeerror", httpmock.NewStringResponder(200, "}"))
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

	id, ok := f.tokenToID[strings.TrimPrefix(auth, "OAuth ")]
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

	assert.NilError(f.t, json.NewDecoder(req.Body).Decode(&body))

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

func (f *fakeTwitch) setStream(id int64, s *twitch.Stream) {
	f.streams[id] = s
}

func (f *fakeTwitch) krakenStreamByID(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "GET")
	f.checkHeaders(req, true)
	assert.Equal(f.t, req.Header.Get("Authorization"), "OAuth "+f.clientTok.AccessToken)

	path := req.URL.Path
	i := strings.LastIndexByte(path, '/')
	path = path[i+1:]

	id, err := strconv.ParseInt(path, 10, 64)
	assert.NilError(f.t, err)

	s, ok := f.streams[id]

	if !ok {
		return httpmock.NewStringResponse(404, "{}"), nil
	}

	v := &struct {
		Stream *twitch.Stream `json:"stream"`
	}{
		Stream: s,
	}

	return httpmock.NewJsonResponse(200, v)
}

func (f *fakeTwitch) kraken(req *http.Request) (*http.Response, error) {
	assert.Equal(f.t, req.Method, "GET")
	f.checkHeaders(req, true)

	auth := req.Header.Get("Authorization")
	assert.Assert(f.t, strings.HasPrefix(auth, "OAuth "))

	id, ok := f.tokenToID[strings.TrimPrefix(auth, "OAuth ")]
	assert.Assert(f.t, ok)

	c := f.channels[id]
	assert.Assert(f.t, c != nil)

	switch c.ID {
	case 777:
		return httpmock.NewStringResponse(200, "}"), nil
	case 503:
		return httpmock.NewStringResponse(503, ""), nil
	}

	return httpmock.NewJsonResponse(200, map[string]interface{}{
		"token": map[string]interface{}{
			"user_id": c.ID,
		},
	})
}

func (f *fakeTwitch) dumpAndFail(req *http.Request, dumped []byte) (*http.Response, error) {
	f.t.Helper()
	if len(dumped) == 0 {
		dumped, _ = httputil.DumpRequest(req, true)
	}
	f.t.Logf("%s\n", dumped)
	return httpmock.ConnectionFailure(req)
}

func (f *fakeTwitch) checkHeaders(req *http.Request, v5 bool) {
	assert.Equal(f.t, req.Header.Get("Client-ID"), clientID)
	assert.Equal(f.t, req.Header.Get("Content-Type"), "application/json")

	if v5 {
		assert.Equal(f.t, req.Header.Get("Accept"), "application/vnd.twitchtv.v5+json")
	}
}
