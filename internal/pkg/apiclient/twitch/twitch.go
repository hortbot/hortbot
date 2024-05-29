// Package twitch implements a Twitch API client. The client makes full use of
// OAuth tokens and requires them where needed.
package twitch

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/eventsub"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch/idstr"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/oauth2x"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/oauth2/endpoints"
)

var ErrDeadToken = errors.New("twitch: dead token")

// UserScopes should be granted for end users.
var UserScopes = []string{
	"moderation:read",            // Helix: get moderator list
	"user:read:broadcast",        // Helix: read channel info, markers
	"channel:read:subscriptions", // Helix: get broadcaster subscriptions
	"channel:read:editors",       // Helix: get channel editors
	"channel:manage:broadcast",   // Helix: modify channel information
	"channel:bot",                // Chat: This token is a bot in the user's channel.
}

// BotScopes are scopes which should be granted for the bot's account.
var BotScopes = slices.Concat(UserScopes, []string{
	"channel:moderate",               // Chat: run moderator commands (TODO: defunct in Feb 2023)
	"chat:read",                      // Chat: read messages
	"chat:edit",                      // Chat: send messages
	"whispers:read",                  // Chat: read whispers
	"whispers:edit",                  // Chat: send whispers
	"moderator:manage:announcements", // Helix: Make announcements
	"moderator:manage:banned_users",  // Helix: Ban/timeout users
	"moderator:manage:chat_messages", // Helix: Delete messages
	"moderator:read:chat_settings",   // Helix: Read chat settings, like emote only, slow mode
	"moderator:manage:chat_settings", // Helix: Change chat settings, like emote only, slow mode
	"user:manage:chat_color",         // Helix: Change bot user color
	"user:bot",                       // Chat: This is a bot
	"user:read:chat",                 // Chat: Read chat via EventSub
	"user:write:chat",                // Helix: Send chat messages
	"user:read:moderated_channels",   // Helix: Get list of channels the user moderates
	"user:manage:whispers",           // Helix: Manage whispers
})

var twitchEndpoint = oauth2.Endpoint{
	AuthURL:   endpoints.Twitch.AuthURL,
	TokenURL:  endpoints.Twitch.TokenURL,
	AuthStyle: oauth2.AuthStyleInParams,
}

const helixRoot = "https://api.twitch.tv/helix"

//go:generate go run github.com/matryer/moq -fmt goimports -out twitchmocks/mocks.go -pkg twitchmocks . API

// API covers the main API methods for Twitch.
type API interface {
	// Auth
	AuthCodeURL(state string, scopes []string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	Validate(ctx context.Context, tok *oauth2.Token) (*Validation, *oauth2.Token, error)

	// Helix
	GetUserByToken(ctx context.Context, userToken *oauth2.Token) (user *User, newToken *oauth2.Token, err error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetChannelModerators(ctx context.Context, id int64, userToken *oauth2.Token) (mods []*ChannelModerator, newToken *oauth2.Token, err error)
	SearchCategories(ctx context.Context, query string) ([]*Category, error)
	ModifyChannel(ctx context.Context, broadcasterID int64, userToken *oauth2.Token, title *string, gameID *int64) (newToken *oauth2.Token, err error)
	GetGameByName(ctx context.Context, name string) (*Category, error)
	GetGameByID(ctx context.Context, id int64) (*Category, error)
	GetStreamByUserID(ctx context.Context, id int64) (*Stream, error)
	GetStreamByUsername(ctx context.Context, username string) (*Stream, error)
	GetChannelByID(ctx context.Context, id int64) (*Channel, error)
	Ban(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, req *BanRequest) (newToken *oauth2.Token, err error)
	Unban(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, userID int64) (newToken *oauth2.Token, err error)
	UpdateChatSettings(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, patch *ChatSettingsPatch) (newToken *oauth2.Token, err error)
	SetChatColor(ctx context.Context, userID int64, userToken *oauth2.Token, color string) (newToken *oauth2.Token, err error)
	DeleteChatMessage(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, id string) (newToken *oauth2.Token, err error)
	ClearChat(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token) (newToken *oauth2.Token, err error)
	Announce(ctx context.Context, broadcasterID int64, modID int64, modToken *oauth2.Token, message string, color string) (newToken *oauth2.Token, err error)
	GetModeratedChannels(ctx context.Context, modID int64, modToken *oauth2.Token) (channels []*ModeratedChannel, newToken *oauth2.Token, err error)
	SendChatMessage(ctx context.Context, broadcasterID int64, senderID int64, senderToken *oauth2.Token, message string) (newToken *oauth2.Token, err error)
	GetConduits(ctx context.Context) ([]*Conduit, error)
	CreateConduit(ctx context.Context, shardCount int) (*Conduit, error)
	UpdateConduit(ctx context.Context, id string, shardCount int) (*Conduit, error)
	DeleteConduit(ctx context.Context, id string) error
	UpdateShards(ctx context.Context, conduitID string, shards []*Shard) error
	GetSubscriptions(ctx context.Context) ([]*eventsub.Subscription, error)
	DeleteSubscription(ctx context.Context, id string) error
	CreateChatSubscription(ctx context.Context, conduitID string, broadcasterID int64, botID int64) error

	// IGDB
	GetGameLinks(ctx context.Context, twitchCategory int64) ([]GameLink, error)
}

// Twitch is the Twitch API client.
type Twitch struct {
	cli      httpx.Client
	clientID string
	forUser  *oauth2.Config
	helixCli *httpClient
}

var _ API = (*Twitch)(nil)

// Option sets an option on the Twitch client.
type Option func(*Twitch)

// New creates a new Twitch client. A client ID, client secret, and redirect
// URL are required; if not provided, New will panic.
func New(clientID, clientSecret, redirectURL string, cli *http.Client) *Twitch {
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
			Scopes:       UserScopes,
		},
		cli: httpx.NewClient(cli, "twitch", false),
	}

	cconf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     twitchEndpoint.TokenURL,
		AuthStyle:    twitchEndpoint.AuthStyle,
	}

	ctx := t.cli.AsOAuth2Client(context.TODO())
	clientTS := cconf.TokenSource(ctx)

	t.helixCli = &httpClient{
		cli:     t.cli,
		ts:      clientTS,
		headers: t.headers(),
	}

	return t
}

// AuthCodeURL returns a URL a user can visit to grant permission for the
// client, and callback to a page with the code to exchange for a token.
//
// state should be randomly generated, i.e. a random UUID which is then
// mapped back through some other lookup.
//
// extraScopes can be specified to request more scopes than the defaults.
func (t *Twitch) AuthCodeURL(state string, scopes []string) string {
	c := *t.forUser
	c.Scopes = scopes
	return c.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("force_verify", "true"))
}

// Exchange provides the Twitch OAuth server with the code and exchanges it
// for an OAuth token for the user who provided the code.
func (t *Twitch) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	ctx = t.cli.AsOAuth2Client(ctx)
	return t.forUser.Exchange(ctx, code) //nolint:wrapcheck
}

func (t *Twitch) headers() http.Header {
	headers := make(http.Header)
	headers.Set("Client-Id", t.clientID)
	headers.Set("Content-Type", "application/json")
	return headers
}

func (t *Twitch) clientForUser(ctx context.Context, tok *oauth2.Token, onNewToken func(*oauth2.Token, error)) *httpClient {
	ctx = t.cli.AsOAuth2Client(ctx)
	ts := t.forUser.TokenSource(ctx, tok)
	ts = oauth2x.NewOnNewWithToken(ts, onNewToken, tok)

	return &httpClient{
		cli:     t.cli,
		ts:      ts,
		headers: t.headers(),
	}
}

type Validation struct {
	UserID idstr.IDStr `json:"user_id"`
	Name   string      `json:"name"`
	Scopes []string    `json:"scopes"`
}

func (t *Twitch) Validate(ctx context.Context, tok *oauth2.Token) (*Validation, *oauth2.Token, error) {
	var newToken *oauth2.Token

	cli := t.clientForUser(ctx, tok, setToken(&newToken))

	var validation Validation
	req, err := cli.NewRequestToJSON(ctx, "https://id.twitch.tv/oauth2/validate", &validation)
	if err != nil {
		return nil, newToken, err
	}

	if err := req.Fetch(ctx); err != nil {
		return nil, newToken, apiclient.WrapRequestErr("twitch", err, nil)
	}

	return &validation, newToken, nil
}
