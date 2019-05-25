package bot

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"go.uber.org/zap"
)

var (
	errNilMessage     = errors.New("bot: nil message")
	errInvalidMessage = errors.New("bot: invalid message")
	errNotImplemented = errors.New("bot: not implemented")
	errNotAuthorized  = errors.New("bot: user is not authorized to use this command")
)

func (b *Bot) Handle(ctx context.Context, origin string, m *irc.Message) {
	logger := ctxlog.FromContext(ctx)

	err := b.handle(ctx, origin, m)

	switch err {
	case nil:
		// Do nothing
	case errNilMessage:
		logger.Error("nil message")
	case errInvalidMessage:
		logger.Warn("invalid message", zap.Any("message", m))
	case errNotImplemented:
		logger.Debug("not implemented", zap.Any("message", m))
	default:
		logger.Error("unhandled error during handle", zap.Error(err), zap.Any("message", m))
	}
}

func (b *Bot) handle(ctx context.Context, origin string, m *irc.Message) error {
	if m == nil {
		return errNilMessage
	}

	if m.Command != "PRIVMSG" {
		// TODO: handle other types of messages
		return nil
	}

	if len(m.Tags) == 0 {
		return errInvalidMessage
	}

	if len(m.Params) == 0 {
		return errInvalidMessage
	}

	id := m.Tags["id"]
	if id == "" {
		return errInvalidMessage
	}

	ctx, logger := ctxlog.FromContextWith(ctx, zap.String("id", id))

	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic during handle", zap.Any("value", r), zap.Stack("stack"))
		}
	}()

	seen, err := b.dedupe.CheckAndMark(id)
	if err != nil {
		logger.Error("error checking for duplicate", zap.Error(err))
		return err
	}

	if seen {
		logger.Debug("message already seen")
		return nil
	}

	message := m.Trailing

	if message == "" {
		return nil
	}

	me := false
	if c, a, ok := irc.ParseCTCP(message); ok {
		if c != "ACTION" {
			logger.Warn("unknown CTCP", zap.String("ctcpCommand", c), zap.String("ctcpArgs", a))
			return nil
		}

		message = a
		me = true
	}

	message = strings.TrimSpace(message)

	if message == "" {
		return nil
	}

	s := &Session{
		Origin:  origin,
		M:       m,
		ID:      id,
		User:    m.Prefix.Name,
		Message: message,
		Me:      me,
		Bot:     b,
		Sender:  b.sender,
	}

	if displayName := m.Tags["display-name"]; displayName != "" {
		s.UserDisplay = displayName
	} else {
		s.UserDisplay = s.User
	}

	roomID := m.Tags["room-id"]
	if roomID == "" {
		logger.Debug("no room ID")
		return errInvalidMessage
	}

	s.RoomID, err = strconv.ParseInt(roomID, 10, 64)
	if err != nil {
		logger.Debug("error parsing room ID", zap.Error(err))
		return err
	}

	// TODO: atomic locking on the channel

	channelName := m.Params[0]
	if channelName == "" || channelName[0] != '#' || len(channelName) == 1 {
		logger.Debug("bad channel name", zap.Strings("params", m.Params))
		return errInvalidMessage
	}

	s.IRCChannel = channelName[1:]

	// TODO: read out user name, ID, and access level

	ctx, logger = ctxlog.FromContextWith(ctx,
		zap.Int64("roomID", s.RoomID),
		zap.String("channel", s.IRCChannel),
	)

	err = transact(b.db, func(tx *sql.Tx) error {
		s.Tx = tx
		return b.handleSession(ctx, s)
	})

	if err != nil {
		logger.Error("error during handle", zap.Error(err))
	}

	return err
}

func (b *Bot) handleSession(ctx context.Context, s *Session) error {
	logger := ctxlog.FromContext(ctx)

	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(s.RoomID)).One(ctx, s.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug("channel not found in database")
			return nil
		}
		return err
	}

	if !channel.Active {
		logger.Warn("channel is not active")
		return nil
	}

	if channel.Name != s.IRCChannel {
		logger.Error("channel name mismatch", zap.String("fromMessage", s.IRCChannel), zap.String("fromDB", channel.Name))
		return errors.New("channel name mismatch") // TODO
	}

	if channel.BotName != s.Origin {
		logger.Warn("bot name mismatch",
			zap.String("expected", channel.BotName),
			zap.String("origin", s.Origin),
		)
		return nil
	}

	s.Channel = channel
	s.UserLevel = s.parseUserLevel()

	// TODO: precheck for links, banned phrases, etc

	wasCommand, err := b.tryCommand(ctx, s, s.Message)
	if wasCommand {
		return err
	}

	// TODO: autoreplies

	return nil
}

func (b *Bot) tryCommand(ctx context.Context, s *Session, message string) (bool, error) {
	if s.Me {
		return false, nil
	}

	tx := s.Tx
	channel := s.Channel
	prefix := channel.Prefix

	if !strings.HasPrefix(message, prefix) {
		return false, nil
	}

	commandName, params := splitSpace(message[len(prefix):])
	commandName = strings.ToLower(commandName)

	if commandName == "" {
		return false, nil
	}

	s.CommandParams = params
	ctx, logger := ctxlog.FromContextWith(ctx, zap.String("command", commandName), zap.String("params", params))

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(channel.ID),
		models.SimpleCommandWhere.Name.EQ(commandName),
		qm.For("UPDATE"),
	).One(ctx, tx)

	switch err {
	case sql.ErrNoRows:
		if ok, err := b.tryBuiltinCommand(ctx, s, commandName, params); ok {
			return true, err
		}

		logger.Debug("unknown command", zap.String("name", commandName))
		return false, nil
	case nil:
	default:
		logger.Error("error getting simple command from database", zap.Error(err))
		return true, err
	}

	commandLevel := NewAccessLevel(command.AccessLevel)
	if !s.UserLevel.CanAccess(commandLevel) {
		return true, errNotAuthorized
	}

	nodes, err := cbp.Parse(command.Message)
	if err != nil {
		logger.Error("command did not parse, which should not happen", zap.Error(err))
		return true, err
	}

	command.Count++

	// Do not modify UpdatedAt, which should be only used for "real" modifications.
	if err := command.Update(ctx, tx, boil.Whitelist(models.SimpleCommandColumns.Count)); err != nil {
		return true, err
	}

	response, err := walk(ctx, nodes, s.doAction)
	if err != nil {
		logger.Debug("error while walking command tree", zap.Error(err))
		return true, err
	}

	err = s.Reply(response)
	return true, err
}

func (b *Bot) tryBuiltinCommand(ctx context.Context, s *Session, name string, params string) (bool, error) {
	builtin := name == "builtin"
	if builtin {
		name, params = splitSpace(params)
		name = strings.ToLower(name)
	}

	if bc, ok := builtinCommands[name]; ok {
		err := bc.run(ctx, s, name, params)
		if err != nil && err != errNotAuthorized {
			logger := ctxlog.FromContext(ctx)
			logger.Debug("error in builtin command", zap.Error(err))
		}
		return true, err
	}

	return builtin, nil
}
