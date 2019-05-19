package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
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
		panic("nil message")
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

	s := &Session{
		Origin:  origin,
		M:       m,
		ID:      id,
		User:    m.Prefix.Name,
		Message: strings.TrimSpace(m.Trailing), // TODO: handle ACTION
		Bot:     b,
		Sender:  b.sender,
	}

	if displayName, ok := m.Tags["display-name"]; ok {
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
	if channelName == "" || channelName[0] != '#' {
		logger.Debug("bad channel name", zap.Strings("params", m.Params))
		return errInvalidMessage
	}

	s.IRCChannel = channelName[1:]

	s.UserLevel = s.parseUserLevel()

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

	// TODO: precheck for links, banned phrases, etc

	wasCommand, err := b.tryCommand(ctx, s)
	if err != nil {
		return err
	}
	if wasCommand {
		return nil
	}

	// TODO: autoreplies

	return nil
}

func (b *Bot) tryCommand(ctx context.Context, s *Session) (bool, error) {
	tx := s.Tx
	message := s.Message
	channel := s.Channel
	prefix := channel.Prefix

	if !strings.HasPrefix(message, prefix) {
		return false, nil
	}

	commandName := message[len(prefix):]
	params := ""

	if i := strings.IndexFunc(commandName, unicode.IsSpace); i != -1 {
		params = strings.TrimSpace(commandName[i+1:])
		commandName = commandName[:i]
	}

	if commandName == "" {
		return false, nil
	}

	ctx, logger := ctxlog.FromContextWith(ctx, zap.String("command", commandName), zap.String("params", params))

	if bc, ok := builtins[commandName]; ok {
		err := bc.run(ctx, s, params)
		if err != nil && err != errNotAuthorized {
			logger.Debug("error in builtin command", zap.Error(err))
		}
		return true, err
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(channel.ID),
		models.SimpleCommandWhere.Name.EQ(commandName),
	).One(ctx, tx)

	switch err {
	case sql.ErrNoRows:
		logger.Debug("unknown command", zap.String("name", commandName))
		return false, nil
	case nil:
	default:
		logger.Error("error getting simple command from database", zap.Error(err))
		return true, err
	}

	nodes, err := cbp.Parse(command.Message)
	if err != nil {
		logger.Error("command did not parse, which should not happen", zap.Error(err))
		return true, err
	}

	walker := func(ctx context.Context, action string) (string, error) {
		switch action {
		case "PARAMETER":
			return params, nil
		case "PARAMETER_CAPS":
			return strings.ToUpper(params), nil
		}

		return "", fmt.Errorf("unknown action: %s", action)
	}

	response, err := walk(ctx, nodes, walker)
	if err != nil {
		logger.Debug("error while walking command tree", zap.Error(err))
		return true, err
	}

	err = s.Reply(response)
	return true, err
}

func transact(db *sql.DB, fn func(*sql.Tx) error) (err error) {
	var tx *sql.Tx
	tx, err = db.Begin()
	if err != nil {
		return err
	}

	rollback := true

	defer func() {
		if rollback {
			if rerr := tx.Rollback(); err == nil && rerr != nil {
				err = rerr
			}
		}
	}()

	err = fn(tx)
	rollback = false

	if err != nil {
		return tx.Rollback()
	}

	return tx.Commit()
}

func walk(ctx context.Context, nodes []cbp.Node, fn func(ctx context.Context, action string) (string, error)) (string, error) {
	// Process all commands, converting them to text nodes.
	for i, node := range nodes {
		if node.Text != "" {
			continue
		}

		action, err := walk(ctx, node.Children, fn)
		if err != nil {
			return "", err
		}

		s, err := fn(ctx, action)
		if err != nil {
			return "", err
		}

		nodes[i] = cbp.Node{
			Text: s,
		}
	}

	var sb strings.Builder

	// Merge all strings.
	for _, node := range nodes {
		sb.WriteString(node.Text)
	}

	return sb.String(), nil
}
