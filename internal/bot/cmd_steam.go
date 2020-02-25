package bot

import (
	"context"
	"errors"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
)

func cmdWhatShouldIPlay(ctx context.Context, s *session, cmd string, args string) error {
	games, err := s.SteamGames(ctx)
	if err != nil {
		return steamError(ctx, s, err)
	}

	if len(games) == 0 {
		return s.Reply(ctx, "Your Steam profile is private, or you own no games.")
	}

	i := s.Deps.Rand.Intn(len(games))
	game := games[i]

	return s.Replyf(ctx, "You could always play: %s (http://store.steampowered.com/app/%d)", game.Name, game.ID)
}

func cmdStatusGame(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage(ctx, "<new status>")
	}

	summary, err := s.SteamSummary(ctx)
	if err != nil {
		return steamError(ctx, s, err)
	}

	replied, err := setGame(ctx, s, summary.Game)
	if replied || err != nil {
		return err
	}

	replied, err = setStatus(ctx, s, args)
	if replied || err != nil {
		return err
	}

	return s.Reply(ctx, "Status and game updated.")
}

func cmdSteamGame(ctx context.Context, s *session, cmd string, args string) error {
	summary, err := s.SteamSummary(ctx)
	if err != nil {
		return steamError(ctx, s, err)
	}

	replied, err := setGame(ctx, s, summary.Game)
	if replied || err != nil {
		return err
	}

	return s.Reply(ctx, "Game updated.")
}

func steamError(ctx context.Context, s *session, err error) error {
	if err == errSteamDisabled {
		return s.Reply(ctx, "Steam support is disabled.")
	}

	var apiErr *apiclient.Error
	if errors.As(err, &apiErr) {
		return s.Reply(ctx, "A Steam API error occurred.")
	}

	return err
}
