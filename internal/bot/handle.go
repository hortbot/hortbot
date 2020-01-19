package bot

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/db/modelsx"
	"github.com/hortbot/hortbot/internal/pkg/correlation"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/hortbot/hortbot/internal/pkg/stringsx"
	"github.com/jakebailey/irc"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

var (
	errInvalidMessage  = errors.New("bot: invalid message")
	errNotAuthorized   = errors.New("bot: user is not authorized to use this command")
	errBuiltinDisabled = errors.New("bot: builtin disabled")
	errNotAllowed      = errors.New("bot: user not allowed")
	errPanicked        = errors.New("bot: handler panicked")
)

func (b *Bot) Handle(ctx context.Context, origin string, m *irc.Message) {
	ctx = correlation.With(ctx)

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	ctx, span := trace.StartSpan(ctx, "Handle")
	defer span.End()

	if !b.initialized {
		panic("bot is not initialized")
	}

	if m == nil {
		ctxlog.Error(ctx, "nil message")
		return
	}

	defer metricHandled.Inc()

	if m.Command == "PING" {
		return
	}

	span.AddAttributes(trace.StringAttribute("irc_command", m.Command))
	ctx = ctxlog.With(ctx, zap.String("irc_command", m.Command))

	start := b.deps.Clock.Now()
	defer func() {
		secs := b.deps.Clock.Since(start).Seconds()
		metricHandleDuration.WithLabelValues(m.Command).Observe(secs)
	}()

	err := b.handle(ctx, origin, m)

	if !isTesting {
		ctxlog.Debug(ctx, "handled message", zap.Duration("took", time.Since(start)))
	}

	switch err {
	case nil, errNotAllowed:
	case errPanicked: // Logged below with more info.
	default:
		metricHandleError.Inc()
		ctxlog.Error(ctx, "error during handle", zap.Error(err), zap.Any("message", m))
	}
}

func (b *Bot) handle(ctx context.Context, origin string, m *irc.Message) (retErr error) {
	ctx, span := trace.StartSpan(ctx, "handle")
	defer span.End()

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(testingPanic); ok {
				panic(r)
			}
			ctxlog.Error(ctx, "panic during handle", zap.Any("value", r), zap.Stack("stack"))
			retErr = errPanicked
		}
	}()

	switch m.Command {
	case "PRIVMSG":
		return b.handlePrivMsg(ctx, origin, m)
	case "USERSTATE":
		return b.handleUserState(ctx, origin, m)
	default:
		ctxlog.Debug(ctx, "unhandled command", zap.Any("message", m))
		return nil
	}
}

func (b *Bot) handlePrivMsg(ctx context.Context, origin string, m *irc.Message) error {
	ctx, span := trace.StartSpan(ctx, "handlePrivMsg")
	defer span.End()

	start := b.deps.Clock.Now()

	s, err := b.buildSession(ctx, origin, m)
	if err != nil {
		return err
	}

	if s.User == s.Origin {
		return nil
	}

	s.Start = start

	span.AddAttributes(
		trace.Int64Attribute("roomID", s.RoomID),
		trace.StringAttribute("channel", s.IRCChannel),
	)
	ctx = ctxlog.With(ctx, zap.Int64("roomID", s.RoomID), zap.String("channel", s.IRCChannel))

	err = transact(ctx, b.db, func(ctx context.Context, tx *sql.Tx) error {
		s.Tx = tx
		err := handleSession(ctx, s)
		s.Tx = nil
		return err
	})
	if err != nil {
		return err
	}

	if s.onFinish != nil {
		return s.onFinish(ctx)
	}

	return nil
}

func (b *Bot) buildSession(ctx context.Context, origin string, m *irc.Message) (*session, error) {
	ctx, span := trace.StartSpan(ctx, "buildSession")
	defer span.End()

	if len(m.Tags) == 0 || len(m.Params) == 0 {
		return nil, errInvalidMessage
	}

	id := m.Tags["id"]
	if id == "" {
		return nil, errInvalidMessage
	}

	if seen, err := b.maybeDedupe(ctx, id); seen || err != nil {
		return nil, err
	}

	user := strings.ToLower(m.Prefix.Name)

	if !b.deps.IsAllowed(user) {
		return nil, errNotAllowed
	}

	message, me := readMessage(m)
	if message == "" {
		return nil, nil
	}

	s := &session{
		Type:    sessionNormal,
		Origin:  origin,
		M:       m,
		Deps:    b.deps,
		ID:      id,
		User:    user,
		Message: message,
		Me:      me,
	}

	if displayName := m.Tags["display-name"]; displayName != "" {
		s.UserDisplay = displayName
	} else {
		s.UserDisplay = s.User
	}

	roomID := m.Tags["room-id"]
	if roomID == "" {
		ctxlog.Debug(ctx, "no room ID")
		return nil, errInvalidMessage
	}

	var err error
	s.RoomID, err = strconv.ParseInt(roomID, 10, 64)
	if err != nil {
		ctxlog.Debug(ctx, "error parsing room ID", zap.String("parsed", roomID), zap.Error(err))
		return nil, err
	}

	if s.RoomID == 0 {
		ctxlog.Debug(ctx, "room ID cannot be zero")
		return nil, errInvalidMessage
	}

	userID := m.Tags["user-id"]
	if userID == "" {
		ctxlog.Debug(ctx, "no user ID")
		return nil, errInvalidMessage
	}

	s.UserID, err = strconv.ParseInt(userID, 10, 64)
	if err != nil {
		ctxlog.Debug(ctx, "error parsing user ID", zap.String("parsed", userID), zap.Error(err))
		return nil, err
	}

	if s.UserID == 0 {
		ctxlog.Debug(ctx, "user ID cannot be zero")
		return nil, errInvalidMessage
	}

	tmiSentTs, _ := strconv.ParseInt(m.Tags["tmi-sent-ts"], 10, 64)
	s.TMISent = time.Unix(tmiSentTs/1000, 0)

	channelName := m.Params[0]
	if channelName == "" || channelName[0] != '#' || len(channelName) == 1 {
		ctxlog.Debug(ctx, "bad channel name", zap.Strings("params", m.Params))
		return nil, errInvalidMessage
	}

	s.IRCChannel = channelName[1:]

	b.testingHelper.checkUserNameID(s.User, s.UserID)
	b.testingHelper.checkUserNameID(s.IRCChannel, s.RoomID)

	return s, nil
}

func (b *Bot) maybeDedupe(ctx context.Context, id string) (seen bool, err error) {
	ctx, span := trace.StartSpan(ctx, "maybeDedupe")
	defer span.End()

	if b.noDedupe {
		return false, nil
	}

	seen, err = b.deps.Redis.DedupeCheckAndMark(ctx, id, 5*time.Minute)
	if err != nil {
		ctxlog.Error(ctx, "error checking for duplicate", zap.Error(err), zap.String("id", id))
		return false, err
	}

	if seen {
		ctxlog.Debug(ctx, "message already seen", zap.String("id", id))
		metricDuplicateMessage.Inc()
	}

	return seen, nil
}

//nolint:gocyclo
func handleSession(ctx context.Context, s *session) error {
	ctx, span := trace.StartSpan(ctx, "handleSession")
	defer span.End()

	if err := pgLock(ctx, s.Tx, s.RoomID); err != nil {
		return err
	}

	s.SetUserLevel()

	if s.Origin == s.IRCChannel {
		return handleManagement(ctx, s)
	}

	// This is the most frequent query; speed it up by executing a hand written query.
	channel := &models.Channel{}
	err := queries.Raw(`SELECT * FROM channels WHERE user_id = $1 FOR UPDATE`, s.RoomID).Bind(ctx, s.Tx, channel)
	if err != nil {
		if err == sql.ErrNoRows {
			ctxlog.Debug(ctx, "channel not found in database")
			return nil
		}
		return err
	}

	if s.IRCChannel == s.User && channel.DisplayName != s.UserDisplay {
		channel.DisplayName = s.UserDisplay
		if err := channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.DisplayName)); err != nil {
			return err
		}
	}

	if !s.Imp {
		if !channel.Active {
			ctxlog.Warn(ctx, "channel is not active")
			return nil
		}

		if channel.Name != s.IRCChannel {
			ctxlog.Warn(ctx, "channel name mismatch",
				zap.String("fromMessage", s.IRCChannel),
				zap.String("fromDB", channel.Name),
			)
			return nil
		}

		if channel.BotName != s.Origin {
			ctxlog.Warn(ctx, "bot name mismatch",
				zap.String("expected", channel.BotName),
				zap.String("origin", s.Origin),
			)
			return nil
		}
	}

	s.Channel = channel
	s.SetUserLevel()

	_, ignored := stringSliceIndex(channel.Ignored, s.User)

	if ignored {
		if s.UserLevel.CanAccess(levelBroadcaster) {
			ignored = false
		} else {
			s.UserLevel = levelEveryone
		}
	}

	if filtered, err := tryFilter(ctx, s); filtered || err != nil {
		return err
	}

	// Ignoring does not exempt messages from filters.

	if ignored || !s.UserLevel.CanAccessPG(s.Channel.Mode) {
		return nil
	}

	s.Channel.MessageCount++
	s.Channel.LastSeen = s.Deps.Clock.Now()

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.MessageCount, models.ChannelColumns.LastSeen)); err != nil {
		return err
	}

	if ok, err := tryCommand(ctx, s); ok || err != nil {
		switch err {
		case errNotAuthorized, errBuiltinDisabled, errInCooldown:
		default:
			if err == nil {
				metricCommands.Inc()
			}
			return err
		}
	}

	if s.Channel.ParseYoutube && s.Deps.YouTube != nil {
		for _, u := range s.Links(ctx) {
			title := s.Deps.YouTube.VideoTitle(ctx, u)
			if title != "" {
				return s.Replyf(ctx, "Linked YouTube video: \"%s\"", title)
			}
		}
	}

	if ok, err := tryAutoreplies(ctx, s); ok || err != nil {
		return err
	}

	return nil
}

func tryCommand(ctx context.Context, s *session) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "tryCommand")
	defer span.End()

	if s.Me {
		return false, nil
	}

	prefix := s.Channel.Prefix
	message := s.Message

	name, params := splitSpace(message)

	if s.Channel.ShouldModerate {
		ok, err := moderationCommands.Run(ctx, s, strings.ToLower(name), params)
		switch {
		case err == errNotAuthorized:
			// Continue.
		case err != nil:
			return true, err
		case ok:
			return true, nil
		}
	}

	if name == "+whatprefix" && s.UserLevel.CanAccess(levelAdmin) {
		return true, s.Reply(ctx, "The prefix for this channel is: "+prefix)
	}

	if !strings.HasPrefix(name, prefix) {
		return false, nil
	}

	name = name[len(prefix):]

	var foreignChannel string
	if strings.HasPrefix(name, "#") && s.UserLevel.CanAccess(levelBroadcaster) {
		foreignChannel, name = stringsx.SplitByte(name[1:], '/')
	}

	name = cleanCommandName(name)

	if name == "" {
		return false, nil
	}

	s.SetCommandParams(params)
	thisChannel := foreignChannel == ""

	ctx = ctxlog.With(ctx, zap.String("name", name), zap.String("params", params), zap.Bool("foreign", !thisChannel))

	channelID := s.Channel.ID

	if !thisChannel {
		foreignChannel = strings.ToLower(foreignChannel)
		otherChannel, err := models.Channels(models.ChannelWhere.Name.EQ(foreignChannel)).One(ctx, s.Tx)
		if err != nil {
			if err == sql.ErrNoRows {
				return true, s.Replyf(ctx, "Channel %s does not exist.", foreignChannel)
			}
			return true, err
		}
		channelID = otherChannel.ID
	}

	info, commandMsg, found, err := modelsx.FindCommand(ctx, s.Tx, channelID, name, thisChannel)
	if err != nil {
		ctxlog.Error(ctx, "error looking up command name in database", zap.Error(err))
		return true, err
	}

	if !found {
		return tryBuiltinCommand(ctx, s, name, params)
	}

	if !s.UserLevel.CanAccessPG(info.AccessLevel) {
		return false, errNotAuthorized
	}

	if commandMsg.Valid {
		return handleCustomCommand(ctx, s, info, commandMsg.String, thisChannel)
	}

	return handleList(ctx, s, info, thisChannel)
}

func tryBuiltinCommand(ctx context.Context, s *session, cmd string, args string) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "tryBuiltinCommand")
	defer span.End()

	if cmd == "builtin" {
		cmd, args = splitSpace(args)
		cmd = cleanCommandName(cmd)
	}

	return builtinCommands.RunWithCooldown(ctx, s, cmd, args)
}
