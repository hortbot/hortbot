package bot

import (
	"context"

	"github.com/hortbot/hortbot/internal/pkg/apis/steam"
)

func cmdWhatShouldIPlay(ctx context.Context, s *session, cmd string, args string) error {
	games, err := s.SteamGames(ctx)
	switch err {
	case errSteamDisabled:
		return s.Reply("Steam support is disabled.")
	case steam.ErrServerError, steam.ErrNotFound:
		return s.Reply("A Steam API error occurred.")
	case nil:
	default:
		return err
	}

	if len(games) == 0 {
		return s.Reply("Your Steam profile is private, or you own no games.")
	}

	i := s.Deps.Rand.Intn(len(games))
	game := games[i]

	return s.Replyf("You could always play: %s (http://store.steampowered.com/app/%d)", game.Name, game.ID)
}

func cmdStatusGame(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.ReplyUsage("<new status>")
	}

	summary, err := s.SteamSummary(ctx)
	switch err {
	case errSteamDisabled:
		return s.Reply("Steam support is disabled.")
	case steam.ErrServerError, steam.ErrNotFound:
		return s.Reply("A Steam API error occurred.")
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

	return s.Reply("Status and game updated.")
}

func cmdSteamGame(ctx context.Context, s *session, cmd string, args string) error {
	summary, err := s.SteamSummary(ctx)
	switch err {
	case errSteamDisabled:
		return s.Reply("Steam support is disabled.")
	case steam.ErrServerError, steam.ErrNotFound:
		return s.Reply("A Steam API error occurred.")
	case nil:
	default:
		return err
	}

	replied, err := setGame(ctx, s, summary.Game)
	if replied || err != nil {
		return err
	}

	return s.Reply("Game updated.")
}
