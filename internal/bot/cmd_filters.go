package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/goware/urlx"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

var filterCommands handlerMap = map[string]handlerFunc{
	"on":      {fn: cmdFilterOnOff(true), minLevel: levelModerator},
	"off":     {fn: cmdFilterOnOff(false), minLevel: levelModerator},
	"links":   {fn: cmdFilterLinks, minLevel: levelModerator},
	"pd":      {fn: cmdFilterPermittedLinks, minLevel: levelModerator},
	"pl":      {fn: cmdFilterPermittedLinks, minLevel: levelModerator},
	"caps":    {fn: cmdFilterCaps, minLevel: levelModerator},
	"symbols": {fn: cmdFilterSymbols, minLevel: levelModerator},
	"me":      {fn: cmdFilterMe, minLevel: levelModerator},
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

func cmdFilterCaps(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterCaps {
			return s.Reply("Caps filter is already enabled.")
		}

		s.Channel.FilterCaps = true
		response = "Caps filter is now enabled."
		column = models.ChannelColumns.FilterCaps

	case "off":
		if !s.Channel.FilterCaps {
			return s.Reply("Caps filter is already disabled.")
		}

		s.Channel.FilterCaps = false
		response = "Caps filter is now disabled."
		column = models.ChannelColumns.FilterCaps

	case "percent":
		percent, err := strconv.Atoi(args)
		if err != nil || percent < 0 || percent > 100 {
			return s.ReplyUsage("percent <0-100>")
		}

		s.Channel.FilterCapsPercentage = percent
		response = fmt.Sprintf("Caps filter percent set to %d%%.", percent)
		column = models.ChannelColumns.FilterCapsPercentage

	case "minchars":
		minChars, err := strconv.Atoi(args)
		if err != nil || minChars < 0 {
			return s.ReplyUsage("minchars <int>")
		}

		s.Channel.FilterCapsMinChars = minChars
		response = fmt.Sprintf("Caps filter min chars set to %d.", minChars)
		column = models.ChannelColumns.FilterCapsMinChars

	case "mincaps":
		minCaps, err := strconv.Atoi(args)
		if err != nil || minCaps < 0 {
			return s.ReplyUsage("mincaps <int>")
		}

		s.Channel.FilterCapsMinCaps = minCaps
		response = fmt.Sprintf("Caps filter min caps set to %d.", minCaps)
		column = models.ChannelColumns.FilterCapsMinCaps

	case "status":
		return s.Replyf("Caps filter=%v, percent=%v, minchars=%v, mincaps=%v",
			s.Channel.FilterCaps,
			s.Channel.FilterCapsPercentage,
			s.Channel.FilterCapsMinChars,
			s.Channel.FilterCapsMinCaps,
		)

	default:
		return s.ReplyUsage("on|off|percent|minchars|mincaps|status")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return err
	}

	return s.Reply(response)
}

func cmdFilterSymbols(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterSymbols {
			return s.Reply("Symbols filter is already enabled.")
		}

		s.Channel.FilterSymbols = true
		response = "Symbols filter is now enabled."
		column = models.ChannelColumns.FilterSymbols

	case "off":
		if !s.Channel.FilterSymbols {
			return s.Reply("Symbols filter is already disabled.")
		}

		s.Channel.FilterSymbols = false
		response = "Symbols filter is now disabled."
		column = models.ChannelColumns.FilterSymbols

	case "percent":
		percent, err := strconv.Atoi(args)
		if err != nil || percent < 0 || percent > 100 {
			return s.ReplyUsage("percent <0-100>")
		}

		s.Channel.FilterSymbolsPercentage = percent
		response = fmt.Sprintf("Symbols filter percent set to %d%%.", percent)
		column = models.ChannelColumns.FilterSymbolsPercentage

	case "min":
		min, err := strconv.Atoi(args)
		if err != nil || min < 0 {
			return s.ReplyUsage("min <int>")
		}

		s.Channel.FilterSymbolsMinSymbols = min
		response = fmt.Sprintf("Symbols filter min symbols set to %d.", min)
		column = models.ChannelColumns.FilterSymbolsMinSymbols

	case "status":
		return s.Replyf("Symbols filter=%v, percent=%v, min=%v",
			s.Channel.FilterSymbols,
			s.Channel.FilterSymbolsPercentage,
			s.Channel.FilterSymbolsMinSymbols,
		)

	default:
		return s.ReplyUsage("on|off|percent|min|status")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return err
	}

	return s.Reply(response)
}

func cmdFilterMe(ctx context.Context, s *session, cmd string, args string) error {
	enable := false

	switch args {
	case "on":
		enable = true
	case "off":
		// Do nothing.
	default:
		return s.ReplyUsage("on|off")
	}

	if s.Channel.FilterMe == enable {
		if enable {
			return s.Reply("Me filter is already enabled.")
		}
		return s.Reply("Me filter is already disabled.")
	}

	s.Channel.FilterMe = enable

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterMe)); err != nil {
		return err
	}

	if enable {
		return s.Reply("Me filter is now enabled.")
	}
	return s.Reply("Me filter is now disabled.")
}
