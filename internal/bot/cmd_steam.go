package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/apiclient/steam"
)

func cmdWhatShouldIPlay(ctx context.Context, s *session, cmd string, args string) error {
	games, err := s.SteamGames(ctx)
	switch err {
	case errSteamDisabled:
		return s.Reply(ctx, "Steam support is disabled.")
	case steam.ErrServerError, steam.ErrNotFound:
		return s.Reply(ctx, "A Steam API error occurred.")
	case nil:
	default:
		return err
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
	switch err {
	case errSteamDisabled:
		return s.Reply(ctx, "Steam support is disabled.")
	case steam.ErrServerError, steam.ErrNotFound:
		return s.Reply(ctx, "A Steam API error occurred.")
	case nil:
	default:
		return err
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
	switch err {
	case errSteamDisabled:
		return s.Reply(ctx, "Steam support is disabled.")
	case steam.ErrServerError, steam.ErrNotFound:
		return s.Reply(ctx, "A Steam API error occurred.")
	case nil:
	default:
		return err
	}

	replied, err := setGame(ctx, s, summary.Game)
	if replied || err != nil {
		return err
	}

	return s.Reply(ctx, "Game updated.")
}
