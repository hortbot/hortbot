package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/opentracing/opentracing-go"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"go.uber.org/zap"
)

func handleCustomCommand(ctx context.Context, s *session, info *models.CommandInfo, message string, update bool) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "handleCustomCommand")
	defer span.Finish()

	if err := s.TryCooldown(); err != nil {
		return false, err
	}
	return true, runCommandAndCount(ctx, s, info, message, update)
}

func runCommandAndCount(ctx context.Context, s *session, info *models.CommandInfo, message string, update bool) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "runCommandAndCount")
	defer span.Finish()

	ctx = withCommandGuard(ctx, info.Name)

	reply, err := processCommand(ctx, s, message)
	if err != nil {
		return err
	}

	if err := s.Reply(ctx, reply); err != nil {
		return err
	}

	if !update {
		return nil
	}

	info.Count++
	info.LastUsed = null.TimeFrom(s.Deps.Clock.Now())

	return info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.Count, models.CommandInfoColumns.LastUsed))
}

func processCommand(ctx context.Context, s *session, msg string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "processCommand")
	defer span.Finish()

	logger := ctxlog.FromContext(ctx)

	if strings.Contains(msg, "(_ONLINE_CHECK_)") {
		isLive, err := s.IsLive(ctx)
		if err != nil || !isLive {
			return "", err
		}
	}

	msg = checkGame(ctx, s, msg, "(_GAME_IS_NOT_", false)
	msg = checkGame(ctx, s, msg, "(_GAME_IS_", true)

	if msg == "" {
		return "", nil
	}

	if strings.Contains(msg, "(_SILENT_)") {
		s.Silent = true
	}

	nodes, err := cbp.Parse(msg)
	if err != nil {
		logger.Error("command did not parse, which should not happen", zap.Error(err))
		return "", err
	}

	return walk(ctx, nodes, s.doAction)
}

func checkGame(ctx context.Context, s *session, msg string, prefix string, want bool) string {
	const suffix = "_)"

	if msg == "" {
		return ""
	}

	i := strings.Index(msg, prefix)
	if i < 0 {
		return msg
	}

	front := msg[:i]
	game := msg[i+len(prefix):]

	i = strings.Index(game, suffix)
	if i < 0 {
		return msg
	}

	end := game[i+len(suffix):]
	game = game[:i]

	game = strings.Map(func(r rune) rune {
		switch r {
		case '-':
			return ' '
		default:
			return r
		}
	}, game)

	if game == "(Not set)" {
		game = ""
	}

	actual := "(error)"
	ch, err := s.TwitchChannel(ctx)
	if err == nil {
		actual = ch.Game
	}

	if want == strings.EqualFold(game, actual) {
		return front + end
	}

	return ""
}
