package bot

import (
	"context"
	"strconv"
	"strings"

	"github.com/goware/urlx"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

var filterCommands handlerMap = map[string]handlerFunc{
	"on":    {fn: cmdFilterOnOff(true), minLevel: LevelModerator},
	"off":   {fn: cmdFilterOnOff(false), minLevel: LevelModerator},
	"links": {fn: cmdFilterLinks, minLevel: LevelModerator},
	"pd":    {fn: cmdFilterPermittedLinks, minLevel: LevelModerator},
	"pl":    {fn: cmdFilterPermittedLinks, minLevel: LevelModerator},
}

func cmdFilter(ctx context.Context, s *session, cmd string, args string) error {
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

func cmdFilterOnOff(enable bool) func(ctx context.Context, s *session, cmd string, args string) error {
	return func(ctx context.Context, s *session, cmd string, args string) error {
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

func cmdFilterLinks(ctx context.Context, s *session, cmd string, args string) error {
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

func cmdFilterPermittedLinks(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage("add|delete|list ...")
	}

	subcommand, args := splitSpace(args)
	if subcommand == "" {
		return usage()
	}

	subcommand = strings.ToLower(subcommand)
	args = strings.ToLower(args)

	switch subcommand {
	case "list":
		permitted := s.Channel.PermittedLinks

		if len(permitted) == 0 {
			return s.Reply("There are no permitted link patterns.")
		}

		var builder strings.Builder
		builder.WriteString("Permitted link patterns: ")

		for i, p := range permitted {
			if i != 0 {
				builder.WriteString(", ")
			}

			builder.WriteString(strconv.Itoa(i + 1))
			builder.WriteString(" = ")
			builder.WriteString(p)
		}

		return s.Reply(builder.String())

	case "add":
		pd, args := splitSpace(args)
		if args != "" {
			return s.ReplyUsage(subcommand + " <link pattern>")
		}

		_, err := urlx.ParseWithDefaultScheme(pd, "https")
		if err != nil {
			return s.Reply("Could not parse link pattern.")
		}

		s.Channel.PermittedLinks = append(s.Channel.PermittedLinks, pd)

		if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.PermittedLinks)); err != nil {
			return err
		}

		n := len(s.Channel.PermittedLinks)
		return s.Replyf("Permitted link pattern #%d added.", n)

	case "delete", "remove":
		n, err := strconv.Atoi(args)
		if err != nil || n <= 0 {
			return s.ReplyUsage(subcommand + " <num>")
		}

		i := n - 1

		old := s.Channel.PermittedLinks

		if i >= len(old) {
			return s.Replyf("Permitted link pattern #%d does not exist.", n)
		}

		oldP := old[i]
		s.Channel.PermittedLinks = append(old[:i], old[i+1:]...) //nolint:gocritic

		if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.PermittedLinks)); err != nil {
			return err
		}

		return s.Replyf("Permitted link pattern #%d deleted; was '%s'.", n, oldP)

	default:
		return usage()
	}
}
