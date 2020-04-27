package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func cmdHighlight(ctx context.Context, s *session, cmd string, args string) error {
	if err := s.TryHighlightCooldown(ctx); err != nil {
		return err
	}

	stream, err := s.TwitchStream(ctx)
	if err != nil || stream == nil {
		return err
	}

	game := stream.Game
	start := stream.CreatedAt

	var status string
	if stream.Channel != nil {
		status = stream.Channel.Status
	}

	highlight := &models.Highlight{
		ChannelID:     s.Channel.ID,
		HighlightedAt: s.Deps.Clock.Now(),
		StartedAt:     null.NewTime(start, !start.IsZero()),
		Status:        status,
		Game:          game,
	}

	return highlight.Insert(ctx, s.Tx, boil.Infer())
}
