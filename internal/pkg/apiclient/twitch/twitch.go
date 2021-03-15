// Package twitch implements a Twitch API client. The client makes full use of
// OAuth tokens and requires them where needed.
package twitch

import (
	"context"
	"errors"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/twitch"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// Twitch API errors.
//
//     - 200 -> nil
//     - 400 -> ErrBadRequest
//     - 404 -> ErrNotFound
//     - 401 or 403 -> ErrNotAuthorized
//     - 5xx -> ErrServerError
//     - Otherwise -> ErrUnknown
var (
	ErrNotFound      = errors.New("twitch: not found")
	ErrNotAuthorized = errors.New("twitch: not authorized")
	ErrBadRequest    = errors.New("twitch: bad request")
	ErrServerError   = errors.New("twitch: server error")
	ErrUnknown       = errors.New("twitch: unknown error")
	ErrDeadToken     = errors.New("twitch: oauth token is dead")
)

// userScopes should be granted for all users.
var userScopes = []string{
	"user_read",                  // Kraken: ???
	"channel_editor",             // Kraken: update channel
	"moderation:read",            // Helix: get moderator list
	"user:read:broadcast",        // Helix: read channel info, markers (implied by user:edit:broadcast?)
	"user:edit:broadcast",        // Helix: modify channel information
	"channel:read:subscriptions", // Helix: get broadcaster subscriptions
}

// BotScopes are scopes which should be granted for the bot's account.
var BotScopes = []string{
	"channel:moderate",  // Chat: run moderator commands
	"chat:read",         // Chat: read messages
	"chat:edit",         // Chat: send messages
	"whispers:read",     // Chat: read whispers
	"whispers:edit",     // Chat: send whispers
	"user_follows_edit", // Kraken: followers
	"user:edit:follows", // Helix: followers
}

var twitchEndpoint = oauth2.Endpoint{
	AuthURL:   twitch.Endpoint.AuthURL,
	TokenURL:  twitch.Endpoint.TokenURL,
	AuthStyle: oauth2.AuthStyleInParams,
}

const (
	krakenRoot = "https://api.twitch.tv/kraken"
	helixRoot  = "https://api.twitch.tv/helix"
)

//counterfeiter:generate . API

// API covers the main API methods for Twitch.
type API interface {
	AuthCodeURL(state string, extraScopes ...string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)

	// Kraken
	GetChannelByID(ctx context.Context, id int64) (c *Channel, err error)
	GetCurrentStream(ctx context.Context, id int64) (s *Stream, err error)
	SetChannelStatus(ctx context.Context, id int64, userToken *oauth2.Token, status string) (newStatus string, newToken *oauth2.Token, err error)
	SetChannelGame(ctx context.Context, id int64, userToken *oauth2.Token, game string) (newGame string, newToken *oauth2.Token, err error)
	FollowChannel(ctx context.Context, id int64, userToken *oauth2.Token, toFollow int64) (newToken *oauth2.Token, err error)

	// Helix
	GetUserByToken(ctx context.Context, userToken *oauth2.Token) (user *User, newToken *oauth2.Token, err error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetChannelModerators(ctx context.Context, id int64, userToken *oauth2.Token) (mods []*ChannelModerator, newToken *oauth2.Token, err error)
	SearchCategories(ctx context.Context, query string) ([]*Category, error)
	ModifyChannel(ctx context.Context, broadcasterID int64, userToken *oauth2.Token, title string, gameID int64) (newToken *oauth2.Token, err error)
	GetGameByName(ctx context.Context, name string) (*Category, error)
	GetGameByID(ctx context.Context, id int64) (*Category, error)
	GetStreamByUserID(ctx context.Context, id int64) (*HelixStream, error)
	GetStreamByUsername(ctx context.Context, username string) (*HelixStream, error)
	GetHelixChannelByID(ctx context.Context, id int64) (*HelixChannel, error)

	// TMI
	GetChatters(ctx context.Context, channel string) (*Chatters, error)
}

// Twitch is the Twitch API client.
type Twitch struct {
	cli      httpx.Client
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

	ctx := t.cli.AsOAuth2Client(context.TODO())
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
		t.cli.Client = cli
	}
}

// AuthCodeURL returns a URL a user can visit to grant permission for the
// client, and callback to a page with the code to exchange for a token.
//
// state should be randomly generated, i.e. a random UUID which is then
// mapped back through some other lookup.
//
// extraScopes can be specified to request more scopes than the defaults.
func (t *Twitch) AuthCodeURL(state string, extraScopes ...string) string {
	c := *t.forUser
	c.Scopes = append([]string(nil), c.Scopes...)
	c.Scopes = append(c.Scopes, extraScopes...)
	return c.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("force_verify", "true"))
}

// Exchange provides the Twitch OAuth server with the code and exchanges it
// for an OAuth token for the user who provided the code.
func (t *Twitch) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	ctx = t.cli.AsOAuth2Client(ctx)
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
	ctx = t.cli.AsOAuth2Client(ctx)
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

func (t *Twitch) krakenClientForUser(ctx context.Context, tok *oauth2.Token, onNewToken func(*oauth2.Token, error)) *httpClient {
	return t.clientForUser(ctx, true, tok, onNewToken)
}

func (t *Twitch) helixClientForUser(ctx context.Context, tok *oauth2.Token, onNewToken func(*oauth2.Token, error)) *httpClient {
	return t.clientForUser(ctx, false, tok, onNewToken)
}
