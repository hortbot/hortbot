package bot

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/ctxlog"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/dedupe"
	"github.com/jakebailey/irc"
	"go.uber.org/zap"
)

type Config struct {
	DB     *sql.DB
	Dedupe dedupe.Deduplicator
}

type Bot struct {
	config *Config

	Nick string
}

func NewBot(config *Config) *Bot {
	return &Bot{
		config: config,
	}
}

type MessageSender interface {
	SendMessage(target, message string) error
}

func (b *Bot) Handle(ctx context.Context, m *irc.Message) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := ctxlog.FromContext(ctx)

	// This defer is okay to be early; the code below updates the logger but this function
	// will refer to the active version (and capture any added IDs and such).
	defer func() {
		if r := recover(); r != nil {
			logger.Error("panic during handle", zap.Any("value", r), zap.Stack("stack"))
		}
	}()

	if len(m.Tags) == 0 {
		// Tags are required to process messages (id, room-id, etc).
		return
	}

	m2 := NewMessage(m)

	id := m2.ID()
	if id == "" {
		logger.Debug("missing ID")

		// TODO: add a correlation ID and handle?
		return
	}

	ctx, logger = ctxlog.FromContextWith(ctx, zap.String("id", id))

	before := time.Now()
	b.handle(ctx, m2)
	took := time.Since(before)

	// TODO: only print when the command actually did something?
	logger.Debug("handled message", zap.Duration("took", took))
}

func (b *Bot) handle(ctx context.Context, m *Message) {
	logger := ctxlog.FromContext(ctx)

	if m.Command() != "PRIVMSG" {
		// TODO
		return
	}

	channelName := m.ChannelName()
	if channelName == "" {
		logger.Debug("bad channel name")
		return
	}

	if channelName == b.Nick {
		// TODO: special case for commands in the bot's own channel
		return
	}

	roomID := m.RoomID()

	if roomID == 0 {
		logger.Debug("no room ID")
		return
	}

	ctx, logger = ctxlog.FromContextWith(ctx, zap.Int64("roomID", roomID))

	channel, err := models.Channels(models.ChannelWhere.UserID.EQ(roomID)).One(ctx, b.config.DB)
	switch err {
	case sql.ErrNoRows:
		// TODO: handle other rooms for a channel
		logger.Debug("channel not found in database")
		return
	case nil:
	default:
		logger.Error("error getting channel from database", zap.Error(err))
		return
	}

	message := m.Message()

	if !strings.HasPrefix(message, channel.Prefix) {
		// Not a command, return for now.
		return
	}

	commandName := message[len(channel.Prefix):]
	rest := ""

	if i := strings.IndexFunc(commandName, unicode.IsSpace); i != -1 {
		rest = strings.TrimSpace(commandName[i+1:])
		commandName = commandName[:i]
	}

	command, err := models.SimpleCommands(
		models.SimpleCommandWhere.ChannelID.EQ(channel.ID),
		models.SimpleCommandWhere.Name.EQ(commandName),
	).One(ctx, b.config.DB)

	switch err {
	case sql.ErrNoRows:
		logger.Debug("unknown command", zap.String("name", commandName))
		return
	case nil:
	default:
		logger.Error("error getting simple command from database", zap.Error(err))
		return
	}

	_ = command

	ctx, logger = ctxlog.FromContextWith(ctx, zap.String("command", commandName), zap.String("rest", rest))

	nodes, err := cbp.Parse(command.Message)
	if err != nil {
		logger.Error("command did not parse, which should not happen", zap.Error(err))
		return
	}

	walker := func(ctx context.Context, action string) (string, error) {
		switch action {
		case "PARAMETER":
			return rest, nil
		case "PARAMETER_CAPS":
			return strings.ToUpper(rest), nil
		}

		return "", fmt.Errorf("unknown action: %s", action)
	}

	response, err := walk(ctx, nodes, walker)
	if err != nil {
		logger.Debug("error while walking command tree", zap.Error(err))
		return
	}

	logger.Info("responsing to command", zap.String("response", response))
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
