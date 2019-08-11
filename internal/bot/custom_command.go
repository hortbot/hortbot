package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"go.uber.org/zap"
)

func handleCustomCommand(ctx context.Context, s *session, info *models.CommandInfo, message string) (bool, error) {
	if err := s.TryCooldown(); err != nil {
		return false, err
	}
	return true, runCommandAndCount(ctx, s, info, message)
}

func runCommandAndCount(ctx context.Context, s *session, info *models.CommandInfo, message string) error {
	ctx = withCommandGuard(ctx, info.Name)

	reply, err := processCommand(ctx, s, message)
	if err != nil {
		return err
	}

	if err := s.Reply(reply); err != nil {
		return err
	}

	info.Count++
	info.LastUsed = null.TimeFrom(s.Deps.Clock.Now())

	return info.Update(ctx, s.Tx, boil.Whitelist(models.CommandInfoColumns.Count, models.CommandInfoColumns.LastUsed))
}

func processCommand(ctx context.Context, s *session, msg string) (string, error) {
	logger := ctxlog.FromContext(ctx)

	if strings.Contains(msg, "(_ONLINE_CHECK_)") {
		isLive, err := s.IsLive(ctx)
		if err != nil || !isLive {
			return "", err
		}
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
