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
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/lastfm"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/hortbot/hortbot/internal/pkg/findlinks"
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
	M      Message

	Deps *sharedDeps
	Tx   *sql.Tx

	Start   time.Time
	TMISent time.Time

	ID         string
	RoomID     int64
	RoomIDOrig int64
	IRCChannel string // No '#' prefix.
	Message    string
	Me         bool

	User        string
	UserDisplay string
	UserID      int64
	UserLevel   AccessLevel

	Channel *models.Channel

	CommandParams  string
	parameters     *[]string
	parameterIndex int

	Silent bool
	Imp    bool

	usageContext string

	sendRoundtrip bool

	cache struct {
		links         onced[[]*url.URL]
		tracks        onced[[]lastfm.Track]
		tok           onced[*oauth2.Token]
		botTok        onced[tokenAndUserID]
		isLive        onced[bool]
		twitchChannel onced[*twitch.Channel]
		twitchStream  onced[*twitch.Stream]
		steamSummary  onced[*steam.Summary]
		steamGames    onced[[]*steam.Game]
		gameLinks     onced[[]twitch.GameLink]
	}
}

type tokenAndUserID struct {
	tok *oauth2.Token
	id  int64
}

func (s *session) formatResponse(response string) (message string, announce bool) {
	response = strings.TrimSpace(response)

	var builder strings.Builder

	addBullet := true

	if strings.HasPrefix(response, "/me ") || strings.HasPrefix(response, ".me ") {
		addBullet = false
	} else if strings.HasPrefix(response, "/announce ") {
		response = strings.TrimPrefix(response, "/announce ")
		if s.Type != sessionAutoreply && (s.UserLevel.CanAccess(AccessLevelModerator) || s.Type == sessionRepeat) {
			announce = true
		}
	}

	if addBullet {
		builder.WriteString(s.bullet())
		builder.WriteByte(' ')
	}

	builder.WriteString(response)

	response = builder.String()

	if len(response) > maxResponseLen {
		return response[:maxResponseLen], announce
	}

	return response, announce
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

	response, announce := s.formatResponse(response)

	if announce {
		return s.Announce(ctx, response)
	}

	return s.SendTwitchChatMessage(ctx, s.IRCChannel, response)
}

func (s *session) Replyf(ctx context.Context, format string, args ...any) error {
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

func (s *session) parseUserLevel() AccessLevel {
	if s.Deps.SuperAdmins[s.User] {
		return AccessLevelSuperAdmin
	}

	if s.Deps.Admins[s.User] {
		return AccessLevelAdmin
	}

	accessLevel := s.M.UserAccessLevel()
	if accessLevel != AccessLevelUnknown {
		return accessLevel
	}

	if s.Channel != nil {
		if _, isOwner := stringSliceIndex(s.Channel.CustomOwners, s.User); isOwner {
			return AccessLevelBroadcaster
		}

		if _, isMod := stringSliceIndex(s.Channel.CustomMods, s.User); isMod {
			return AccessLevelModerator
		}

		if _, isReg := stringSliceIndex(s.Channel.CustomRegulars, s.User); isReg {
			return AccessLevelSubscriber
		}
	}

	return AccessLevelEveryone
}

func (s *session) DeleteMessage(ctx context.Context) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.DeleteChatMessage(ctx, s.Channel.TwitchID, botID, tok, s.ID)
	if err != nil {
		logTwitchModerationError(ctx, err, "delete")
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
		case errors.Is(err, sql.ErrNoRows):
			return nil, nil
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

	tt := modelsx.TokenToModelWithoutPreservedColumns(newToken, s.Channel.TwitchID)
	return modelsx.UpsertTokenWithoutPreservedColumns(ctx, s.Tx, tt)
}

func (s *session) BotTwitchToken(ctx context.Context) (int64, *oauth2.Token, error) {
	ctx, span := trace.StartSpan(ctx, "BotTwitchToken")
	defer span.End()

	botName := s.Origin
	if s.Channel != nil {
		botName = s.Channel.BotName
	}

	pair, err := s.cache.botTok.get(func() (tokenAndUserID, error) {
		tt, err := models.TwitchTokens(models.TwitchTokenWhere.BotName.EQ(null.StringFrom(botName))).One(ctx, s.Tx)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return tokenAndUserID{}, nil
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

	tt := modelsx.TokenToModelWithoutPreservedColumns(newToken, botID)
	return modelsx.UpsertTokenWithoutPreservedColumns(ctx, s.Tx, tt)
}

func (s *session) GetUserID(ctx context.Context, username string) (int64, error) {
	switch username {
	case s.IRCChannel:
		return s.RoomIDOrig, nil
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
		logTwitchModerationError(ctx, err, "ban")
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
		logTwitchModerationError(ctx, err, "unban")
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
		logTwitchModerationError(ctx, err, "update chat")
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
		logTwitchModerationError(ctx, err, "update settings")
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
}

func (s *session) Announce(ctx context.Context, message string) error {
	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.Announce(ctx, s.Channel.TwitchID, botID, tok, message, "")
	if err != nil {
		logTwitchModerationError(ctx, err, "announce")
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	return err
}

func (s *session) SendTwitchChatMessage(ctx context.Context, target string, message string) error {
	if s.Deps.NoSend {
		ctxlog.Info(ctx, "not sending", zap.String("origin", s.Origin), zap.String("target", target), zap.String("message", message))
		return nil
	}

	botID, tok, err := s.BotTwitchToken(ctx)
	if err != nil {
		return err
	}

	if tok == nil {
		return errors.New("bot: bot not authorized")
	}

	targetID, err := s.GetUserID(ctx, target)
	if err != nil {
		return err
	}

	newToken, err := s.Deps.Twitch.SendChatMessage(ctx, targetID, botID, tok, message)
	if err != nil {
		logTwitchModerationError(ctx, err, "send message")
	}

	if newToken != nil {
		if err := s.SetBotTwitchToken(ctx, botID, newToken); err != nil {
			return err
		}
	}

	if err != nil {
		metricSentErrors.Inc()
		return err
	}

	metricSent.Inc()
	return nil
}

func (s *session) IsLive(ctx context.Context) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "IsLive")
	defer span.End()

	return s.cache.isLive.get(func() (bool, error) {
		stream, err := s.TwitchStream(ctx)
		if err != nil {
			if errors.Is(err, twitch.ErrNotFound) {
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
	if apiErr, ok := apiclient.AsError(err); ok && apiErr.IsServerError() {
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

func (s *session) FilterExemptLevel() AccessLevel {
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

func logTwitchModerationError(ctx context.Context, err error, operation string) {
	if err == nil {
		return
	}

	// Usually indicates that the bot isn't modded.
	if errors.Is(err, twitch.ErrNotAuthorized) {
		return
	}

	ctxlog.Error(ctx, "error in twitch API", zap.Error(err), zap.String("operation", operation))
}
