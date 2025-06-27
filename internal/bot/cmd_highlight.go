package bot

import (
	"context"
	"fmt"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
)

func cmdHighlight(ctx context.Context, s *session, cmd string, args string) error {
	if err := s.TryHighlightCooldown(ctx); err != nil {
		return err
	}

	stream, err := s.TwitchStream(ctx)
	if err != nil {
		if ae, ok := apiclient.AsError(err); ok && ae.IsNotFound() {
			return nil
		}
		return err
	}

	var gameName string

	if stream.GameID != 0 {
		game, err := s.Deps.Twitch.GetGameByID(ctx, int64(stream.GameID))
		if err != nil {
			return fmt.Errorf("get game by ID: %w", err)
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

	if err := highlight.Insert(ctx, s.Tx, boil.Infer()); err != nil {
		return fmt.Errorf("insert highlight: %w", err)
	}

	return nil
}
