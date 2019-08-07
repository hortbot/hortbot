package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

func runCustomCommand(ctx context.Context, s *session, msg string) error {
	logger := ctxlog.FromContext(ctx)

	if strings.Contains(msg, "(_ONLINE_CHECK_)") {
		isLive, err := s.IsLive(ctx)
		if err != nil || !isLive {
			return err
		}
	}

	if strings.Contains(msg, "(_SILENT_)") {
		s.Silent = true
	}

	nodes, err := cbp.Parse(msg)
	if err != nil {
		logger.Error("command did not parse, which should not happen", zap.Error(err))
		return err
	}

	response, err := walk(ctx, nodes, s.doAction)
	if err != nil {
		logger.Debug("error while walking command tree", zap.Error(err))
		return err
	}

	return s.Reply(response)
}
