package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/apis/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
	"github.com/hortbot/hortbot/internal/pkg/apis/tinyurl"
	"github.com/hortbot/hortbot/internal/pkg/apis/twitch"
	"github.com/hortbot/hortbot/internal/pkg/findlinks"
	"github.com/jakebailey/irc"
	"go.opencensus.io/trace"
	"golang.org/x/oauth2"
)

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
	IRCChannel string
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   accessLevel

	Channel *models.Channel

	CommandParams  string
	Parameters     *[]string
	ParameterIndex int

	Silent bool
	Imp    bool

	usageContext string

	onFinish func(ctx context.Context) error

	cache struct {
		links          *[]*url.URL
		tracks         *[]lastfm.Track
		tok            **oauth2.Token
		isLive         *bool
		twitchChannel  **twitch.Channel
		twitchStream   **twitch.Stream
		twitchChatters **twitch.Chatters
		steamSummary   **steam.Summary
		steamGames     *[]*steam.Game
	}
}

func (s *session) formatResponse(response string) string {
	response = strings.TrimSpace(response)

	if len(response) >= 4 {
		switch response[:4] {
		case "/me ", ".me ":
			return response
		}
	}

	return s.bullet() + " " + response
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
	if s.Deps.Admins[s.User] {
		return levelAdmin
	}

	// Tags are present, safe to not check for nil

	tags := s.M.Tags

	if isTesting && tags["testing-admin"] != "" {
		return levelAdmin
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
	case badges["subscriber"] != "", badges["vip"] != "", tags["subscriber"] == "1":
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
	case "slow":
	case "slowoff":
	case "subscribers":
	case "subscribersoff":
	case "r9kbeta":
	case "r9kbetaoff":
	case "ban":
	case "unban":
	case "timeout":
	case "untimeout":
	case "me":
	case "delete":
	case "clear":
	case "host":
	case "unhost":
	case "color":
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

	return s.Deps.Sender.SendMessage(ctx, s.Origin, "#"+s.IRCChannel, builder.String())
}

func (s *session) DeleteMessage(ctx context.Context) error {
	return s.SendCommand(ctx, "delete", s.ID)
}

func (s *session) Links(ctx context.Context) []*url.URL {
	_, span := trace.StartSpan(ctx, "Links")
	defer span.End()

	if s.cache.links != nil {
		return *s.cache.links
	}

	links := findlinks.Find(s.Message, "http", "https", "ftp")
	s.cache.links = &links
	return links
}

var errLastFMDisabled = errors.New("bot: LastFM disabled")

func (s *session) Tracks(ctx context.Context) ([]lastfm.Track, error) {
	_, span := trace.StartSpan(ctx, "Tracks")
	defer span.End()

	if s.cache.tracks != nil {
		return *s.cache.tracks, nil
	}

	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return nil, errLastFMDisabled
	}

	tracks, err := s.Deps.LastFM.RecentTracks(ctx, s.Channel.LastFM, 2)
	if err != nil {
		return nil, err
	}
	s.cache.tracks = &tracks
	return tracks, nil
}

func (s *session) TwitchToken(ctx context.Context) (*oauth2.Token, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchToken")
	defer span.End()

	if s.cache.tok != nil {
		return *s.cache.tok, nil
	}

	tt, err := models.TwitchTokens(models.TwitchTokenWhere.TwitchID.EQ(s.Channel.UserID)).One(ctx, s.Tx)
	switch {
	case err == sql.ErrNoRows:
		var tok *oauth2.Token
		s.cache.tok = &tok
		return nil, nil
	case err != nil:
		return nil, err
	}

	tok := modelsx.ModelToToken(tt)
	s.cache.tok = &tok
	return tok, nil
}

func (s *session) SetTwitchToken(ctx context.Context, newToken *oauth2.Token) error {
	ctx, span := trace.StartSpan(ctx, "SetTwitchToken")
	defer span.End()

	s.cache.tok = &newToken

	tt := modelsx.TokenToModel(s.Channel.UserID, newToken)
	return modelsx.UpsertToken(ctx, s.Tx, tt)
}

func (s *session) IsLive(ctx context.Context) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "IsLive")
	defer span.End()

	if s.cache.isLive != nil {
		return *s.cache.isLive, nil
	}

	stream, err := s.TwitchStream(ctx)
	if err != nil {
		return false, err
	}

	isLive := stream != nil
	s.cache.isLive = &isLive
	return isLive, nil
}

func (s *session) TwitchChannel(ctx context.Context) (*twitch.Channel, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchChannel")
	defer span.End()

	if s.cache.twitchChannel != nil {
		return *s.cache.twitchChannel, nil
	}

	ch, err := s.Deps.Twitch.GetChannelByID(ctx, s.Channel.UserID)
	if err != nil {
		return nil, err
	}

	s.cache.twitchChannel = &ch
	return ch, nil
}

func (s *session) TwitchStream(ctx context.Context) (*twitch.Stream, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchStream")
	defer span.End()

	if s.cache.twitchStream != nil {
		return *s.cache.twitchStream, nil
	}

	st, err := s.Deps.Twitch.GetCurrentStream(ctx, s.Channel.UserID)
	if err != nil {
		return nil, err
	}

	s.cache.twitchStream = &st
	return st, nil
}

func (s *session) TwitchChatters(ctx context.Context) (*twitch.Chatters, error) {
	ctx, span := trace.StartSpan(ctx, "TwitchChatters")
	defer span.End()

	if s.cache.twitchChatters != nil {
		return *s.cache.twitchChatters, nil
	}

	chatters, err := s.Deps.Twitch.GetChatters(ctx, s.Channel.Name)
	if err != nil {
		return nil, err
	}

	s.cache.twitchChatters = &chatters
	return chatters, nil
}

var errSteamDisabled = errors.New("bot: steam disabled")

func (s *session) SteamSummary(ctx context.Context) (*steam.Summary, error) {
	ctx, span := trace.StartSpan(ctx, "SteamSummary")
	defer span.End()

	if s.cache.steamSummary != nil {
		return *s.cache.steamSummary, nil
	}

	if s.Deps.Steam == nil || s.Channel.SteamID == "" {
		return nil, errSteamDisabled
	}

	summary, err := s.Deps.Steam.GetPlayerSummary(ctx, s.Channel.SteamID)
	if err != nil {
		return nil, err
	}

	s.cache.steamSummary = &summary
	return summary, nil
}

func (s *session) SteamGames(ctx context.Context) ([]*steam.Game, error) {
	ctx, span := trace.StartSpan(ctx, "SteamGames")
	defer span.End()

	if s.cache.steamGames != nil {
		return *s.cache.steamGames, nil
	}

	if s.Deps.Steam == nil || s.Channel.SteamID == "" {
		return nil, errSteamDisabled
	}

	games, err := s.Deps.Steam.GetOwnedGames(ctx, s.Channel.SteamID)
	if err != nil {
		return nil, err
	}

	s.cache.steamGames = &games
	return games, nil
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
	if s.Deps.WebAddrMap != nil {
		if adr := s.Deps.WebAddrMap[s.Channel.BotName]; adr != "" {
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
	s.Parameters = nil
	s.ParameterIndex = 0
}

func (s *session) RoomIDStr() string {
	return strconv.FormatInt(s.RoomID, 10)
}

func (s *session) OnFinish(fn func(context.Context) error) {
	s.onFinish = fn
}
