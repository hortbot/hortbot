// Package twitch implements a Twitch API client. The client makes full use of
// OAuth tokens and requires them where needed.
package twitch

import (
	"context"
	"errors"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Twitch API errors.
//
// - 200 -> nil
// - 404 -> ErrNotFound
// - 401 or 403 -> ErrNotAuthorized
// - 5xx -> ErrServerError
// - Otherwise -> ErrUnknown
var (
	ErrNotFound      = errors.New("twitch: not found")
	ErrNotAuthorized = errors.New("twitch: not authorized")
	ErrServerError   = errors.New("twitch: server error")
	ErrUnknown       = errors.New("twitch: unknown error")
)

var userScopes = []string{
	"user_read",
	"channel_editor",
}

var twitchEndpoint = oauth2.Endpoint{
	AuthURL:   twitch.Endpoint.AuthURL,
	TokenURL:  twitch.Endpoint.TokenURL,
	AuthStyle: oauth2.AuthStyleInParams,
}

const (
	krakenRoot = "https://api.twitch.tv/kraken"
	// helixRoot  = "https://api.twitch.tv/helix"
)

// API covers the main API methods for Twitch. It does not include OAuth-only
// methods which would not be called from the bot (but instead by a website).
//
//counterfeiter:generate . API
type API interface {
	GetIDForUsername(ctx context.Context, username string) (int64, error)
	GetIDForToken(ctx context.Context, userToken *oauth2.Token) (id int64, newToken *oauth2.Token, err error)
	GetChannelByID(ctx context.Context, id int64) (c *Channel, err error)
	SetChannelStatus(ctx context.Context, id int64, userToken *oauth2.Token, status string) (newStatus string, newToken *oauth2.Token, err error)
	SetChannelGame(ctx context.Context, id int64, userToken *oauth2.Token, game string) (newGame string, newToken *oauth2.Token, err error)
	GetCurrentStream(ctx context.Context, id int64) (s *Stream, err error)
	GetChatters(ctx context.Context, channel string) (int64, error)
}

// Twitch is the Twitch API client.
type Twitch struct {
	cli      *http.Client
	clientID string
	forUser  *oauth2.Config

	krakenCli *httpClient
	helixCli  *httpClient
}

var _ API = (*Twitch)(nil)

// Option sets an option on the Twitch client.
type Option func(*Twitch)

// New creates a new Twitch client. A client ID, client secret, and redirect
// URL are required; if not provided, New will panic.
func New(clientID, clientSecret, redirectURL string, opts ...Option) *Twitch {
	switch {
	case clientID == "":
		panic("empty clientID")
	case clientSecret == "":
		panic("empty clientSecret")
	case redirectURL == "":
		panic("empty redirectURL")
	}

	t := &Twitch{
		cli:      http.DefaultClient,
		clientID: clientID,
		forUser: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     twitchEndpoint,
			RedirectURL:  redirectURL,
			Scopes:       userScopes,
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	cconf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     twitchEndpoint.TokenURL,
		AuthStyle:    twitchEndpoint.AuthStyle,
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, t.cli)
	clientTS := cconf.TokenSource(ctx)

	t.krakenCli = &httpClient{
		cli:     t.cli,
		ts:      oauth2x.NewTypeOverride(clientTS, "OAuth"),
		headers: t.headers(true),
	}

	t.helixCli = &httpClient{
		cli:     t.cli,
		ts:      oauth2x.NewTypeOverride(clientTS, "Bearer"),
		headers: t.headers(false),
	}

	return t
}

// HTTPClient sets the Twitch client's underlying http.Client.
// If nil (or if this option wasn't used), http.DefaultClient will be used.
func HTTPClient(cli *http.Client) Option {
	return func(t *Twitch) {
		if cli == nil {
			t.cli = http.DefaultClient
		} else {
			t.cli = cli
		}
	}
}

// AuthCodeURL returns a URL a user can visit to grant permission for the
// client, and callback to a page with the code to exchange for a token.
//
// state should be randomly generated, i.e. a random UUID which is then
// mapped back through some other lookup.
func (t *Twitch) AuthCodeURL(state string) string {
	return t.forUser.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange provides the Twitch OAuth server with the code and exchanges it
// for an OAuth token for the user who provided the code.
func (t *Twitch) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, t.cli)
	return t.forUser.Exchange(ctx, code)
}

func (t *Twitch) headers(v5 bool) http.Header {
	headers := make(http.Header)
	headers.Set("Client-ID", t.clientID)
	headers.Set("Content-Type", "application/json")

	if v5 {
		headers.Set("Accept", "application/vnd.twitchtv.v5+json")
	}

	return headers
}

func (t *Twitch) clientForUser(ctx context.Context, v5 bool, tok *oauth2.Token, onNewToken func(*oauth2.Token, error)) *httpClient {
	ts := t.forUser.TokenSource(ctx, tok)
	ts = oauth2x.NewOnNewWithToken(ts, onNewToken, tok)

	if v5 {
		ts = oauth2x.NewTypeOverride(ts, "OAuth")
	} else {
		ts = oauth2x.NewTypeOverride(ts, "Bearer")
	}

	return &httpClient{
		cli:     t.cli,
		ts:      ts,
		headers: t.headers(v5),
	}
}
