package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var raffleCommands = newHandlerMap(map[string]handlerFunc{
	"enable":  {fn: cmdRaffleEnableDisable, minLevel: levelModerator},
	"disable": {fn: cmdRaffleEnableDisable, minLevel: levelModerator},
	"count":   {fn: cmdRaffleCount, minLevel: levelModerator},
	"winner":  {fn: cmdRaffleWinner, minLevel: levelModerator},
	"reset":   {fn: cmdRaffleReset, minLevel: levelModerator},
})

func cmdRaffle(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		if s.Channel.RaffleEnabled {
			if err := s.RaffleAdd(ctx, s.User); err != nil {
				return err
			}
		}
		return nil
	}

	ok, err := raffleCommands.RunWithCooldown(ctx, s, subcommand, args)
	if ok || err != nil {
		return err
	}

	return s.ReplyUsage(ctx, "enable|disable|count|winner|reset")
}

func cmdRaffleEnableDisable(ctx context.Context, s *session, cmd string, args string) error {
	enable := false

	switch cmd {
	case "enable":
		enable = true
	case "disable":
	default:
		panic("unreachable")
	}

	if s.Channel.RaffleEnabled == enable {
		if enable {
			return s.Replyf(ctx, "Raffle is already enabled. Use %sraffle to enter!", s.Channel.Prefix)
		}
		return s.Reply(ctx, "Raffle is already disabled.")
	}

	if enable {
		if err := s.RaffleReset(ctx); err != nil {
			return err
		}
	}

	s.Channel.RaffleEnabled = enable

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.RaffleEnabled)); err != nil {
		return err
	}

	if enable {
		return s.Replyf(ctx, "Raffle enabled. Use %sraffle to enter!", s.Channel.Prefix)
	}

	return s.Reply(ctx, "Raffle disabled.")
}

func cmdRaffleCount(ctx context.Context, s *session, cmd string, args string) error {
	count, err := s.RaffleCount(ctx)
	if err != nil {
		return err
	}

	if count == 0 {
		return s.Reply(ctx, "Raffle has no entries.")
	}

	entries := "entries"
	if count == 1 {
		entries = "entry"
	}

	return s.Replyf(ctx, "Raffle has %d %s.", count, entries)
}

func cmdRaffleWinner(ctx context.Context, s *session, cmd string, args string) error {
	winner, ok, err := s.RaffleWinner(ctx)
	if err != nil {
		return err
	}

	if !ok {
		return s.Reply(ctx, "Raffle has no entries.")
	}

	return s.Replyf(ctx, "Winner is %s!", winner)
}

func cmdRaffleReset(ctx context.Context, s *session, cmd string, args string) error {
	if err := s.RaffleReset(ctx); err != nil {
		return err
	}

	return s.Reply(ctx, "Raffle reset.")
}
