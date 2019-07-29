package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/cbp"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"github.com/volatiletech/sqlboiler/boil"
	"go.uber.org/zap"
)

func runCustomCommand(ctx context.Context, s *session, command *models.CustomCommand) error {
	logger := ctxlog.FromContext(ctx)

	nodes, err := cbp.Parse(command.Message)
	if err != nil {
		logger.Error("command did not parse, which should not happen", zap.Error(err))
		return err
	}

	command.Count++

	// Do not modify UpdatedAt, which should be only used for "real" modifications.
	if err := command.Update(ctx, s.Tx, boil.Whitelist(models.CustomCommandColumns.Count)); err != nil {
		return err
	}

	response, err := walk(ctx, nodes, s.doAction)
	if err != nil {
		logger.Debug("error while walking command tree", zap.Error(err))
		return err
	}

	return s.Reply(response)
}
