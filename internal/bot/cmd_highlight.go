package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/hortbot/hortbot/internal/db/dbsql"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/jackc/pgx/v5/pgtype"
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

	err = s.Queries.InsertHighlight(ctx, dbsql.InsertHighlightParams{
		ChannelID:     s.Channel.ID,
		HighlightedAt: dbsql.TimestamptzFrom(time.Now()),
		StartedAt:     pgtype.Timestamptz{Time: start, Valid: !start.IsZero()},
		Status:        status,
		Game:          gameName,
	})
	if err != nil {
		return fmt.Errorf("insert highlight: %w", err)
	}

	return nil
}
