package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
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
	"github.com/opentracing/opentracing-go"
	"github.com/volatiletech/sqlboiler/boil"
	"golang.org/x/oauth2"
)

//go:generate gobin -run -m golang.org/x/tools/cmd/stringer -type=sessionType -trimprefix=session

type sessionType int

const (
	sessionUnknown sessionType = 0
	sessionNormal  sessionType = 1 << (iota - 1)
	sessionRepeat
	sessionAutoreply
	sessionSubNotification
)

type session struct {
	Type sessionType

	Origin string
	M      *irc.Message

	Deps *sharedDeps
	Tx   boil.ContextExecutor

	Start   time.Time
	TMISent time.Time

	N          int64
	ID         string
	RoomID     int64
	RoomIDStr  string
	IRCChannel string
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   accessLevel
	Ignored     bool

	Channel *models.Channel

	CommandParams     string
	OrigCommandParams string

	Silent bool
	Imp    bool

	usageContext   string
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

func (s *session) formatResponse(response string) string {
	response = strings.TrimSpace(response)

	if len(response) >= 4 {
		switch response[:4] {
		case "/me ", ".me ":
			return response
		}
	}

	bullet := s.Deps.DefaultBullet

	if s.Channel != nil && s.Channel.Bullet.Valid {
		bullet = s.Channel.Bullet.String
	}

	return bullet + " " + response
}

func (s *session) Reply(ctx context.Context, response string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Reply")
	defer span.Finish()

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

func (s *session) IsAdmin() bool {
	return s.UserLevel.CanAccess(levelAdmin)
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
	span, _ := opentracing.StartSpanFromContext(ctx, "Links")
	defer span.Finish()

	if s.links != nil {
		return *s.links
	}

	links := findlinks.Find(s.Message, "http", "https", "ftp")
	s.links = &links
	return links
}

var errLastFMDisabled = errors.New("bot: LastFM disabled")

func (s *session) Tracks(ctx context.Context) ([]lastfm.Track, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Tracks")
	defer span.Finish()

	if s.tracks != nil {
		return *s.tracks, nil
	}

	if s.Deps.LastFM == nil || s.Channel.LastFM == "" {
		return nil, errLastFMDisabled
	}

	tracks, err := s.Deps.LastFM.RecentTracks(s.Channel.LastFM, 2)
	if err != nil {
		return nil, err
	}
	s.tracks = &tracks
	return tracks, nil
}

func (s *session) TwitchToken(ctx context.Context) (*oauth2.Token, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "TwitchToken")
	defer span.Finish()

	if s.tok != nil {
		return *s.tok, nil
	}

	tt, err := models.TwitchTokens(models.TwitchTokenWhere.TwitchID.EQ(s.Channel.UserID)).One(ctx, s.Tx)
	switch {
	case err == sql.ErrNoRows:
		var tok *oauth2.Token
		s.tok = &tok
		return nil, nil
	case err != nil:
		return nil, err
	}

	tok := modelsx.ModelToToken(tt)
	s.tok = &tok
	return tok, nil
}

func (s *session) SetTwitchToken(ctx context.Context, newToken *oauth2.Token) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SetTwitchToken")
	defer span.Finish()

	s.tok = &newToken

	tt := modelsx.TokenToModel(s.Channel.UserID, newToken)
	return modelsx.UpsertToken(ctx, s.Tx, tt)
}

func (s *session) IsLive(ctx context.Context) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "IsLive")
	defer span.Finish()

	if s.isLive != nil {
		return *s.isLive, nil
	}

	stream, err := s.TwitchStream(ctx)
	if err != nil {
		return false, err
	}

	isLive := stream != nil
	s.isLive = &isLive
	return isLive, nil
}

func (s *session) TwitchChannel(ctx context.Context) (*twitch.Channel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "TwitchChannel")
	defer span.Finish()

	if s.twitchChannel != nil {
		return *s.twitchChannel, nil
	}

	ch, err := s.Deps.Twitch.GetChannelByID(ctx, s.Channel.UserID)
	if err != nil {
		return nil, err
	}

	s.twitchChannel = &ch
	return ch, nil
}

func (s *session) TwitchStream(ctx context.Context) (*twitch.Stream, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "TwitchStream")
	defer span.Finish()

	if s.twitchStream != nil {
		return *s.twitchStream, nil
	}

	st, err := s.Deps.Twitch.GetCurrentStream(ctx, s.Channel.UserID)
	if err != nil {
		return nil, err
	}

	s.twitchStream = &st
	return st, nil
}

func (s *session) TwitchChatters(ctx context.Context) (*twitch.Chatters, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "TwitchChatters")
	defer span.Finish()

	if s.twitchChatters != nil {
		return *s.twitchChatters, nil
	}

	chatters, err := s.Deps.Twitch.GetChatters(ctx, s.Channel.Name)
	if err != nil {
		return nil, err
	}

	s.twitchChatters = &chatters
	return chatters, nil
}

var (
	errSteamDisabled = errors.New("bot: steam disabled")
)

func (s *session) SteamSummary(ctx context.Context) (*steam.Summary, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SteamSummary")
	defer span.Finish()

	if s.steamSummary != nil {
		return *s.steamSummary, nil
	}

	if s.Deps.Steam == nil || s.Channel.SteamID == "" {
		return nil, errSteamDisabled
	}

	summary, err := s.Deps.Steam.GetPlayerSummary(ctx, s.Channel.SteamID)
	if err != nil {
		return nil, err
	}

	s.steamSummary = &summary
	return summary, nil
}

func (s *session) SteamGames(ctx context.Context) ([]*steam.Game, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "SteamGames")
	defer span.Finish()

	if s.steamGames != nil {
		return *s.steamGames, nil
	}

	if s.Deps.Steam == nil || s.Channel.SteamID == "" {
		return nil, errSteamDisabled
	}

	games, err := s.Deps.Steam.GetOwnedGames(ctx, s.Channel.SteamID)
	if err != nil {
		return nil, err
	}

	s.steamGames = &games
	return games, nil
}

func (s *session) ShortenLink(ctx context.Context, link string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ShortenLink")
	defer span.Finish()

	if s.Deps.TinyURL == nil {
		return link, nil
	}

	short, err := s.Deps.TinyURL.Shorten(ctx, link)
	if err == tinyurl.ErrServerError {
		return link, nil
	}
	return short, err
}
