package bot

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/jakebailey/irc"
	"github.com/volatiletech/sqlboiler/boil"
	"go.uber.org/zap"
)

func (b *Bot) Handle(ctx context.Context, m *irc.Message) {
	if m == nil {
		panic("nil message")
	}

	if m.Command != "PRIVMSG" {
		// TODO: handle other types of messages
		return
	}

	if len(m.Tags) == 0 {
		return
	}

	if len(m.Params) == 0 {
		return
	}

	id := m.Tags["id"]
	if id == "" {
		return
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
		return
	}

	if seen {
		logger.Debug("message already seen")
		return
	}

	c := &Context{
		M:       m,
		ID:      id,
		Bot:     b,
		Message: strings.TrimSpace(m.Trailing),
		Sender:  b.sender,
	}

	roomID := m.Tags["room-id"]
	if roomID == "" {
		logger.Debug("no room ID")
		return
	}

	c.RoomID, err = strconv.ParseInt(roomID, 10, 64)
	if err != nil {
		logger.Debug("error parsing room ID", zap.Error(err))
		return
	}

	// TODO: atomic locking on the channel

	channelName := m.Params[0]
	if channelName == "" || channelName[0] != '#' {
		logger.Debug("bad channel name", zap.Strings("params", m.Params))
		return
	}

	c.ChannelName = channelName[1:]

	ctx, logger = ctxlog.FromContextWith(ctx,
		zap.Int64("roomID", c.RoomID),
		zap.String("channel", c.ChannelName),
	)

	err = transact(b.db, func(tx *sql.Tx) error {
		c.Tx = tx
		return b.handle(ctx, c)
	})

	if err != nil {
		logger.Error("error during handle", zap.Error(err))
	}
}

func (b *Bot) handle(ctx context.Context, c *Context) error {
	logger := ctxlog.FromContext(ctx)

	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(c.RoomID)).One(ctx, c.Tx)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug("channel not found in database")
			return nil
		}
		return err
	}

	c.Channel = channel

	// TODO: should this be done here at all, or earlier during a rejoin?
	if channel.Name != c.ChannelName {
		channel.Name = c.ChannelName
		if err := channel.Update(ctx, c.Tx, boil.Infer()); err != nil {
			logger.Error("error updating channel name in database", zap.Error(err))
			return err
		}
	}

	if channel.BotName != b.name {
		logger.Warn("bot name mismatch",
			zap.String("expected", channel.BotName),
			zap.String("got", b.name),
		)
		return nil
	}

	// TODO: precheck for links, banned phrases, etc

	wasCommand, err := b.trySimpleCommand(ctx, c)
	if err != nil {
		return err
	}
	if wasCommand {
		return nil
	}

	// TODO: autoreplies

	return nil
}

func (b *Bot) trySimpleCommand(ctx context.Context, c *Context) (bool, error) {
	tx := c.Tx
	message := c.Message
	channel := c.Channel
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
		err := bc(ctx, c, params)
		if err != nil {
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

	// TODO: handle null bullet
	err = c.Sender.SendMessage("#"+c.ChannelName, c.formatResponse(response))
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

	rollback = false
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
