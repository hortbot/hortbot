package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/findlinks"
	"github.com/jakebailey/irc"
	"go.opencensus.io/trace"
	"golang.org/x/oauth2"
)

const maxResponseLen = 500

type sessionType int

const (
	sessionUnknown sessionType = iota //nolint:varcheck,deadcode
	sessionNormal
	sessionRepeat
	sessionAutoreply
)

type session struct {
	Type sessionType

	Origin string
	M      *irc.Message

	Deps *sharedDeps
	Tx   *sql.Tx

	Start   time.Time
	TMISent time.Time

	ID         string
	RoomID     int64
	IRCChannel string // No '#' prefix.
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   accessLevel

	Channel *models.Channel

	CommandParams  string
	parameters     *[]string
	parameterIndex int

	Silent bool
	Imp    bool

	usageContext string

	sendRoundtrip bool

	cache struct {
		links          onced[[]*url.URL]
		tracks         onced[[]lastfm.Track]
		tok            onced[*oauth2.Token]
		isLive         onced[bool]
		twitchChannel  onced[*twitch.Channel]
		twitchStream   onced[*twitch.Stream]
		twitchChatters onced[*twitch.Chatters]
		steamSummary   onced[*steam.Summary]
		steamGames     onced[[]*steam.Game]
		gameLinks      onced[[]twitch.GameLink]
	}
}

func (s *session) formatResponse(response string) string {
	response = strings.TrimSpace(response)

	var builder strings.Builder

	if !strings.HasPrefix(response, "/me ") && !strings.HasPrefix(response, ".me ") {
		builder.WriteString(s.bullet())
		builder.WriteByte(' ')
	}

	builder.WriteString(response)

	response = builder.String()

	if len(response) > maxResponseLen {
		return response[:maxResponseLen]
	}

	return response
}

func (s *session) bullet() string {
	if s.Channel != nil && s.Channel.Bullet.Valid {
		return s.Channel.Bullet.String
	}

	return s.defaultBullet()
}

func (s *session) defaultBullet() string {
	if s.Deps.BulletMap != nil {
		if b := s.Deps.BulletMap[s.Origin]; b != "" {
			return b
		}
	}

	return DefaultBullet
}

func (s *session) Reply(ctx context.Context, response string) error {
	ctx, span := trace.StartSpan(ctx, "Reply")
	defer span.End()

	if s.Silent {
		return nil
	}

	response = strings.TrimSpace(response)

	if response == "" {
		return nil
	}

	return s.Deps.Sender.SendMessage(ctx, s.Origin, "#"+s.IRCChannel, s.formatResponse(response))
}

func (s *session) Replyf(ctx context.Context, format string, args ...interface{}) error {
	response := fmt.Sprintf(format, args...)
	return s.Reply(ctx, response)
}

func (s *session) ReplyUsage(ctx context.Context, usage string) error {
	var builder strings.Builder
	builder.WriteString("Usage: ")

	if s.Channel != nil {
		builder.WriteString(s.Channel.Prefix)
	}

	builder.WriteString(s.usageContext)
	builder.WriteString(usage)

	return s.Reply(ctx, builder.String())
}

func (s *session) UsageContext(command string) func() {
	if command == "" {
		return func() {}
	}

	do := true

	old := s.usageContext
	s.usageContext += command + " "
	return func() {
		if do {
			s.usageContext = old
			do = false
		}
	}
}

func (s *session) SetUserLevel() {
	s.UserLevel = s.parseUserLevel()
}

func (s *session) parseUserLevel() accessLevel {
	if s.Deps.SuperAdmins[s.User] {
		return levelSuperAdmin
	}

	if s.Deps.Admins[s.User] {
		return levelAdmin
	}

	// Tags are present, safe to not check for nil

	tags := s.M.Tags

	if isTesting {
		switch {
		case tags["testing-super-admin"] != "":
			return levelSuperAdmin
		case tags["testing-admin"] != "":
			return levelAdmin
		}
	}

	if s.User == s.IRCChannel {
		return levelBroadcaster
	}

	if tags["mod"] == "1" {
		return levelModerator
	}

	badges := parseBadges(tags["badges"])

	switch {
	case badges["broadcaster"] != "":
		return levelBroadcaster
	case badges["moderator"] != "":
		return levelModerator
	case badges["vip"] != "":
		return levelVIP
	case badges["subscriber"] != "", tags["subscriber"] == "1", badges["founder"] != "":
		return levelSubscriber
	}

	if tags["user-type"] == "mod" {
		return levelModerator
	}

	if s.Channel != nil {
		if _, isOwner := stringSliceIndex(s.Channel.CustomOwners, s.User); isOwner {
			return levelBroadcaster
		}

		if _, isMod := stringSliceIndex(s.Channel.CustomMods, s.User); isMod {
			return levelModerator
		}

		if _, isReg := stringSliceIndex(s.Channel.CustomRegulars, s.User); isReg {
			return levelSubscriber
		}
	}

	return levelEveryone
}

func (s *session) SendCommand(ctx context.Context, command string, args ...string) error {
	switch command {
	case "slow": // TODO: s.Deps.Twitch.UpdateChatSettings
	case "slowoff": // TODO: s.Deps.Twitch.UpdateChatSettings
	case "subscribers": // TODO: s.Deps.Twitch.UpdateChatSettings
	case "subscribersoff": // TODO: s.Deps.Twitch.UpdateChatSettings
	case "r9kbeta": // TODO: s.Deps.Twitch.UpdateChatSettings
	case "r9kbetaoff": // TODO: s.Deps.Twitch.UpdateChatSettings
	case "ban": // TODO: s.Deps.Twitch.Ban
	case "unban": // TODO: s.Deps.Twitch.Unban
	case "timeout": // TODO: s.Deps.Twitch.Ban(1)
	case "untimeout": // TODO: s.Deps.Twitch.Unban
	case "me": // OK
	case "delete": // TODO: s.Deps.Twitch.DeleteChatMessage
	case "clear": // TODO: s.Deps.Twitch.ClearChat
	case "host": // TODO: remove
	case "unhost": // TODO: remove
	case "color": // TODO: s.Deps.Twitch.SetChatColor
	default:
		panic("attempt to use IRC command " + command)
	}

	var builder strings.Builder
	builder.WriteByte('/')
	builder.WriteString(command)

	for _, arg := range args {
		builder.WriteByte(' ')
		builder.WriteString(arg)
	}

	message := strings.TrimSpace(builder.String())

	return s.Deps.Sender.SendMessage(ctx, s.Origin, "#"+s.IRCChannel, message)
}

func (s *session) DeleteMessage(ctx context.Context) error {
	return s.SendCommand(ctx, "delete", s.ID)
}

func (s *session) Links(ctx context.Context) []*url.URL {
	_, span := trace.StartSpan(ctx, "Links")
	defer span.End()

	links, _ := s.cache.links.get(func() ([]*url.URL, error) {
		return findlinks.Find(s.Message, "http", "https", "ftp"), nil
	})
	return links
}

var errLastFMDisabled = errors.New("bot: LastFM disabled")

func (s *session) Tracks(ctx context.Context) ([]lastfm.Track, error) {
	_, span := trace.StartSpan(ctx, "Tracks")
	defer span.End()

	return s.cache.tracks.get(func() ([]lastfm.Track, error) {
		if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
			return nil, errLastFMDisabled
		}
		return s.Deps.LastFM.RecentTracks(ctx, s.Channel.LastFM, 2)
	})
}

// TwitchToken returns the twitch token for the user. If not found, the token is nil,
// which is a valid token to use in the Twitch API client.
func (s *session) TwitchToken(ctx context.Context) (*oauth2.Token, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchToken")
	defer span.End()

	return s.cache.tok.get(func() (*oauth2.Token, error) {
		tt, err := models.TwitchTokens(models.TwitchTokenWhere.TwitchID.EQ(s.Channel.TwitchID)).One(ctx, s.Tx)
		switch {
		case err == sql.ErrNoRows:
			return nil, nil //nolint:nilnil
		case err != nil:
			return nil, err
		}

		return modelsx.ModelToToken(tt), nil
	})
}

func (s *session) SetTwitchToken(ctx context.Context, newToken *oauth2.Token) error {
	ctx, span := trace.StartSpan(ctx, "SetTwitchToken")
	defer span.End()

	s.cache.tok.set(newToken, nil)

	tt := modelsx.TokenToModel(s.Channel.TwitchID, newToken)
	return modelsx.UpsertToken(ctx, s.Tx, tt)
}

func (s *session) IsLive(ctx context.Context) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "IsLive")
	defer span.End()

	return s.cache.isLive.get(func() (bool, error) {
		stream, err := s.TwitchStream(ctx)
		if err != nil {
			if err == twitch.ErrNotFound {
				stream = nil
			} else {
				return false, err
			}
		}
		return stream != nil, nil
	})
}

func (s *session) TwitchChannel(ctx context.Context) (*twitch.Channel, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchChannel")
	defer span.End()

	return s.cache.twitchChannel.get(func() (*twitch.Channel, error) {
		return s.Deps.Twitch.GetChannelByID(ctx, s.Channel.TwitchID)
	})
}

func (s *session) TwitchStream(ctx context.Context) (*twitch.Stream, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchStream")
	defer span.End()

	return s.cache.twitchStream.get(func() (*twitch.Stream, error) {
		return s.Deps.Twitch.GetStreamByUserID(ctx, s.Channel.TwitchID)
	})
}

func (s *session) TwitchChatters(ctx context.Context) (*twitch.Chatters, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchChatters")
	defer span.End()

	return s.cache.twitchChatters.get(func() (*twitch.Chatters, error) {
		return s.Deps.Twitch.GetChatters(ctx, s.Channel.Name)
	})
}

var errSteamDisabled = errors.New("bot: steam disabled")

func (s *session) SteamSummary(ctx context.Context) (*steam.Summary, error) {
	ctx, span := trace.StartSpan(ctx, "SteamSummary")
	defer span.End()

	return s.cache.steamSummary.get(func() (*steam.Summary, error) {
		if s.Deps.Steam == nil || s.Channel.SteamID == "" {
			return nil, errSteamDisabled
		}
		return s.Deps.Steam.GetPlayerSummary(ctx, s.Channel.SteamID)
	})
}

func (s *session) SteamGames(ctx context.Context) ([]*steam.Game, error) {
	ctx, span := trace.StartSpan(ctx, "SteamGames")
	defer span.End()

	return s.cache.steamGames.get(func() ([]*steam.Game, error) {
		if s.Deps.Steam == nil || s.Channel.SteamID == "" {
			return nil, errSteamDisabled
		}
		return s.Deps.Steam.GetOwnedGames(ctx, s.Channel.SteamID)
	})
}

func (s *session) GameLinks(ctx context.Context) ([]twitch.GameLink, error) {
	ctx, span := trace.StartSpan(ctx, "GameLinks")
	defer span.End()

	return s.cache.gameLinks.get(func() ([]twitch.GameLink, error) {
		ch, err := s.TwitchChannel(ctx)
		if err != nil {
			return nil, err
		}

		links, err := s.Deps.Twitch.GetGameLinks(ctx, ch.GameID.AsInt64())
		if err != nil {
			return nil, err
		}

		sort.Slice(links, func(i, j int) bool { return links[i].Type < links[j].Type })
		return links, nil
	})
}

func (s *session) ShortenLink(ctx context.Context, link string) (string, error) {
	ctx, span := trace.StartSpan(ctx, "ShortenLink")
	defer span.End()

	if s.Deps.TinyURL == nil {
		return link, nil
	}

	short, err := s.Deps.TinyURL.Shorten(ctx, link)
	if err == tinyurl.ErrServerError {
		return link, nil
	}
	return short, err
}

func (s *session) WebAddr() string {
	return s.WebAddrFor(s.Channel.BotName)
}

func (s *session) WebAddrFor(botName string) string {
	if s.Deps.WebAddrMap != nil {
		if adr := s.Deps.WebAddrMap[botName]; adr != "" {
			return adr
		}
	}
	return s.Deps.WebAddr
}

func (s *session) HelpMessage() string {
	return "You can find help at: " + s.WebAddr()
}

func (s *session) TwitchNotAuthMessage() string {
	return "The bot wasn't authorized to perform this action. Log in on the website to give permission: " + s.WebAddr() + "/login"
}

func (s *session) SetCommandParams(params string) {
	s.CommandParams = params
	s.parameters = nil
	s.parameterIndex = 0
}

func (s *session) RoomIDStr() string {
	return strconv.FormatInt(s.RoomID, 10)
}

func (s *session) BetaFeatures() bool {
	_, ok := stringSliceIndex(s.Deps.BetaFeatures, s.IRCChannel)
	return ok
}

func (s *session) FilterExemptLevel() accessLevel {
	return newAccessLevel(s.Channel.FilterExemptLevel)
}

// onced is not safe for concurrent use.
type onced[T any] struct {
	v     T
	err   error
	valid bool
}

func (o *onced[T]) get(fn func() (T, error)) (T, error) {
	if !o.valid {
		o.set(fn())
	}
	return o.v, o.err
}

func (o *onced[T]) set(v T, err error) {
	o.v = v
	o.err = err
	o.valid = true
}
