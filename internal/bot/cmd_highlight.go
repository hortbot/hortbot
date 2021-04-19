package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/apiclient/twitch"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func cmdHighlight(ctx context.Context, s *session, cmd string, args string) error {
	if err := s.TryHighlightCooldown(ctx); err != nil {
		return err
	}

	stream, err := s.TwitchStream(ctx)
	if err != nil {
		if err == twitch.ErrNotFound {
			return nil
		}
		return err
	}

	var gameName string

	if stream.GameID != 0 {
		game, err := s.Deps.Twitch.GetGameByID(ctx, stream.GameID.AsInt64())
		if err != nil {
			return err
		}
		gameName = game.Name
	}

	start := stream.StartedAt
	status := stream.Title

	highlight := &models.Highlight{
		ChannelID:     s.Channel.ID,
		HighlightedAt: s.Deps.Clock.Now(),
		StartedAt:     null.NewTime(start, !start.IsZero()),
		Status:        status,
		Game:          gameName,
	}

	return highlight.Insert(ctx, s.Tx, boil.Infer())
}
