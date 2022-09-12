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
	"github.com/volatiletech/null/v8"
	"github.com/zikaeroh/ctxlog"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
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
		botTok         onced[tokenAndUserID]
		isLive         onced[bool]
		twitchChannel  onced[*twitch.Channel]
		twitchStream   onced[*twitch.Stream]
		twitchChatters onced[*twitch.Chatters]
		steamSummary   onced[*steam.Summary]
		steamGames     onced[[]*steam.Game]
		gameLinks      onced[[]twitch.GameLink]
	}
}

type tokenAndUserID struct {
	tok *oauth2.Token
	id  int64
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
	case "me": // TODO: remove; unused
	case "host": // TODO: remove
	case "unhost": // TODO: remove
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
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.DeleteChatMessage(ctx, s.Channel.TwitchID, botID, tok, s.ID)
	if err != nil {
		ctxlog.Error(ctx, "unable to delete message", zap.Error(err))
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
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

func (s *session) ChannelTwitchToken(ctx context.Context) (*oauth2.Token, error) {
	ctx, span := trace.StartSpan(ctx, "ChannelTwitchToken")
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

func (s *session) SetChannelTwitchToken(ctx context.Context, newToken *oauth2.Token) error {
	ctx, span := trace.StartSpan(ctx, "SetChannelTwitchToken")
	defer span.End()

	s.cache.tok.set(newToken, nil)

	tt := modelsx.TokenToModel(s.Channel.TwitchID, newToken)
	return modelsx.UpsertToken(ctx, s.Tx, tt)
}

func (s *session) BotTwitchToken(ctx context.Context) (int64, *oauth2.Token, error) {
	ctx, span := trace.StartSpan(ctx, "BotTwitchToken")
	defer span.End()

	pair, err := s.cache.botTok.get(func() (tokenAndUserID, error) {
		tt, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(null.StringFrom(s.Channel.BotName))).One(ctx, s.Tx)
		switch {
		case err == sql.ErrNoRows:
			return tokenAndUserID{}, nil //nolint:nilnil
		case err != nil:
			return tokenAndUserID{}, err
		}

		return tokenAndUserID{
			tok: modelsx.ModelToToken(tt),
			id:  tt.TwitchID,
		}, nil
	})

	return pair.id, pair.tok, err
}

func (s *session) SetBotTwitchToken(ctx context.Context, botID int64, newToken *oauth2.Token) error {
	ctx, span := trace.StartSpan(ctx, "SetBotTwitchToken")
	defer span.End()

	s.cache.botTok.set(tokenAndUserID{tok: newToken, id: botID}, nil)

	tt := modelsx.TokenToModel(botID, newToken)
	return modelsx.UpsertToken(ctx, s.Tx, tt)
}

func (s *session) GetUserID(ctx context.Context, username string) (int64, error) {
	switch username {
	case s.Channel.Name:
		return s.Channel.TwitchID, nil
	case s.User:
		return s.UserID, nil
	}

	user, err := s.Deps.Twitch.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, err
	}
	return user.ID.AsInt64(), nil
}

func (s *session) BanByUsername(ctx context.Context, username string, duration int64, reason string) error {
	userID, err := s.GetUserID(ctx, username)
	if err != nil {
		return err
	}
	return s.BanByID(ctx, userID, duration, reason)
}

func (s *session) BanByID(ctx context.Context, userID int64, duration int64, reason string) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	req := &twitch.BanRequest{
		UserID:   twitch.IDStr(userID),
		Duration: duration,
		Reason:   reason,
	}

	newToken, err := s.Deps.Twitch.Ban(ctx, s.Channel.TwitchID, botID, tok, req)
	if err != nil {
		ctxlog.Error(ctx, "unable to ban user", zap.Error(err))
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
}

func (s *session) UnbanByUsername(ctx context.Context, username string) error {
	userID, err := s.GetUserID(ctx, username)
	if err != nil {
		return err
	}
	return s.UnbanByID(ctx, userID)
}

func (s *session) UnbanByID(ctx context.Context, userID int64) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.Unban(ctx, s.Channel.TwitchID, botID, tok, userID)
	if err != nil {
		ctxlog.Error(ctx, "unable to unban user", zap.Error(err))
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
}

func (s *session) SetBotColor(ctx context.Context, color string) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.SetChatColor(ctx, botID, tok, color)
	if err != nil {
		ctxlog.Error(ctx, "unable to set chat color", zap.Error(err))
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
}

func (s *session) ClearChat(ctx context.Context) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.ClearChat(ctx, s.Channel.TwitchID, botID, tok)
	if err != nil {
		ctxlog.Error(ctx, "unable to clear chat", zap.Error(err))
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
}

func (s *session) UpdateChatSettings(ctx context.Context, patch *twitch.ChatSettingsPatch) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.UpdateChatSettings(ctx, s.Channel.TwitchID, botID, tok, patch)
	if err != nil {
		ctxlog.Error(ctx, "unable to change chat settings", zap.Error(err))
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
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
