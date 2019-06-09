package bot

import (
	"context"
	"strings"

	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

var filterCommands handlerMap = map[string]handlerFunc{
	"on":    {fn: cmdFilterOnOff(true), minLevel: LevelModerator},
	"off":   {fn: cmdFilterOnOff(false), minLevel: LevelModerator},
	"links": {fn: cmdFilterLinks, minLevel: LevelModerator},
}

func cmdFilter(ctx context.Context, s *Session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return s.ReplyUsage("<option> ...")
	}

	ok, err := filterCommands.run(ctx, s, subcommand, args)
	if !ok {
		return s.Replyf("No such filter option '%s'.", subcommand)
	}

	return err
}

func cmdFilterOnOff(enable bool) func(ctx context.Context, s *Session, cmd string, args string) error {
	return func(ctx context.Context, s *Session, cmd string, args string) error {
		if s.Channel.EnableFilters == enable {
			if enable {
				return s.Reply("Filters are already enabled.")
			}
			return s.Reply("Filters are already disabled.")
		}

		s.Channel.EnableFilters = enable

		if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.EnableFilters)); err != nil {
			return err
		}

		if enable {
			return s.Reply("Filters are now enabled.")
		}
		return s.Reply("Filters are now disabled.")
	}
}

func cmdFilterLinks(ctx context.Context, s *Session, cmd string, args string) error {
	enable := false

	switch args {
	case "on":
		enable = true
	case "off":
		// Do nothing.
	default:
		return s.ReplyUsage("on|off")
	}

	if s.Channel.FilterLinks == enable {
		if enable {
			return s.Reply("Link filter is already enabled.")
		}
		return s.Reply("Link filter is already disabled.")
	}

	s.Channel.FilterLinks = enable

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterLinks)); err != nil {
		return err
	}

	if enable {
		return s.Reply("Link filter is now enabled.")
	}
	return s.Reply("Link filter is now disabled.")
}
