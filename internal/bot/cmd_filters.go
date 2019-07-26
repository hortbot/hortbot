package bot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/goware/urlx"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/volatiletech/sqlboiler/boil"
)

var filterCommands handlerMap = map[string]handlerFunc{
	"status":        {fn: cmdFilterStatus, minLevel: levelModerator},
	"on":            {fn: cmdFilterOnOff(true), minLevel: levelModerator},
	"off":           {fn: cmdFilterOnOff(false), minLevel: levelModerator},
	"links":         {fn: cmdFilterLinks, minLevel: levelModerator},
	"pd":            {fn: cmdFilterPermittedLinks, minLevel: levelModerator},
	"pl":            {fn: cmdFilterPermittedLinks, minLevel: levelModerator},
	"caps":          {fn: cmdFilterCaps, minLevel: levelModerator},
	"symbols":       {fn: cmdFilterSymbols, minLevel: levelModerator},
	"me":            {fn: cmdFilterMe, minLevel: levelModerator},
	"messagelength": {fn: cmdFilterMessageLength, minLevel: levelModerator},
	"emotes":        {fn: cmdFilterEmotes, minLevel: levelModerator},
	"banphrase":     {fn: cmdFilterBanPhrase, minLevel: levelModerator},
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

func cmdFilterStatus(ctx context.Context, s *session, cmd string, args string) error {
	builder := &strings.Builder{}

	builder.WriteString("Global: ")
	writeBool(builder, s.Channel.EnableFilters)

	builder.WriteString(", enable warnings: ")
	writeBool(builder, s.Channel.EnableWarnings)

	builder.WriteString(", timeout duration: ")
	builder.WriteString(strconv.Itoa(s.Channel.TimeoutDuration))

	builder.WriteString(", display warnings: ")
	writeBool(builder, s.Channel.DisplayWarnings)

	builder.WriteString(", max message length: ")
	builder.WriteString(strconv.Itoa(s.Channel.FilterMaxLength))

	builder.WriteString(", me: ")
	writeBool(builder, s.Channel.FilterMe)

	builder.WriteString(", links: ")
	writeBool(builder, s.Channel.FilterLinks)

	// builder.WriteString(", banned phrases: ")

	builder.WriteString(", caps: ")
	writeBool(builder, s.Channel.FilterCaps)
	fmt.Fprintf(builder, " {%d%%, %d, %d}", s.Channel.FilterCapsPercentage, s.Channel.FilterCapsMinChars, s.Channel.FilterCapsMinCaps)

	builder.WriteString(", emotes: ")
	writeBool(builder, s.Channel.FilterEmotes)
	fmt.Fprintf(builder, " {%d, %v}", s.Channel.FilterEmotesMax, s.Channel.FilterEmotesSingle)

	builder.WriteString(", symbols: ")
	writeBool(builder, s.Channel.FilterSymbols)
	fmt.Fprintf(builder, " {%d%%, %d}", s.Channel.FilterSymbolsPercentage, s.Channel.FilterSymbolsMinSymbols)

	return s.Reply(builder.String())
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

func cmdFilterMessageLength(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.Replyf("Max message length set to %d.", s.Channel.FilterMaxLength)
	}

	n, err := strconv.Atoi(args)
	if err != nil || n < 0 {
		return s.ReplyUsage("<length>")
	}

	s.Channel.FilterMaxLength = n

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterMaxLength)); err != nil {
		return err
	}

	return s.Replyf("Max message length set to %d.", n)
}

func cmdFilterEmotes(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterEmotes {
			return s.Reply("Emote filter is already enabled.")
		}

		s.Channel.FilterEmotes = true
		response = "Emote filter is now enabled."
		column = models.ChannelColumns.FilterEmotes

	case "off":
		if !s.Channel.FilterEmotes {
			return s.Reply("Emote filter is already disabled.")
		}

		s.Channel.FilterEmotes = false
		response = "Emote filter is now disabled."
		column = models.ChannelColumns.FilterEmotes

	case "max":
		max, err := strconv.Atoi(args)
		if err != nil || max < 0 {
			return s.ReplyUsage("max <num>")
		}

		s.Channel.FilterEmotesMax = max
		response = fmt.Sprintf("Emote filter max emotes set to %d.", max)
		column = models.ChannelColumns.FilterEmotesMax

	case "single":
		enabled := false

		switch args {
		case "on":
			enabled = true
		case "off":
		default:
			return s.ReplyUsage("single on|off")
		}

		if s.Channel.FilterEmotesSingle == enabled {
			if enabled {
				return s.Reply("Single emote filter is already enabled.")
			}
			return s.Reply("Single emote filter is already disabled.")
		}

		s.Channel.FilterEmotesSingle = enabled

		if enabled {
			response = "Single emote filter is now enabled."
		} else {
			response = "Single emote filter is now disabled."
		}

		column = models.ChannelColumns.FilterEmotesSingle

	case "status":
		return s.Replyf("Emote filter=%v, max=%v, single=%v",
			s.Channel.FilterEmotes,
			s.Channel.FilterEmotesMax,
			s.Channel.FilterEmotesSingle,
		)

	default:
		return s.ReplyUsage("on|off|max|single|status")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return err
	}

	return s.Reply(response)
}

func cmdFilterBanPhrase(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterBannedPhrases {
			return s.Reply("Banned phrase filter is already enabled.")
		}

		s.Channel.FilterBannedPhrases = true
		column = models.ChannelColumns.FilterBannedPhrases
		response = "Banned phrase filter is now enabled."

	case "off":
		if !s.Channel.FilterBannedPhrases {
			return s.Reply("Banned phrase filter is already disabled.")
		}

		s.Channel.FilterBannedPhrases = false
		column = models.ChannelColumns.FilterBannedPhrases
		response = "Banned phrase filter is now disabled."

	case "clear":
		s.Channel.FilterBannedPhrasesPatterns = []string{}
		column = models.ChannelColumns.FilterBannedPhrasesPatterns
		response = "Banned phrases have been cleared."

	case "add":
		var pattern string
		if strings.HasPrefix(args, "REGEX:") {
			if !s.UserLevel.CanAccess(levelAdmin) {
				return s.Reply("Only admins may add regex banned words.")
			}

			pattern = strings.TrimPrefix(args, "REGEX:")
			if pattern == "" {
				return s.replyBadPattern(errEmptyPattern)
			}

			_, err := s.Deps.ReCache.Compile(pattern)
			if err != nil {
				return s.replyBadPattern(err)
			}
		} else {
			pattern = regexp.QuoteMeta(args)
		}

		s.Channel.FilterBannedPhrasesPatterns = append(s.Channel.FilterBannedPhrasesPatterns, pattern)
		column = models.ChannelColumns.FilterBannedPhrasesPatterns
		response = "Banned phrase added."

	case "delete", "remove":
		quoted := regexp.QuoteMeta(args)

		tries := []string{
			args,
			".*" + quoted + ".*",
			quoted,
			".*" + args + ".*",
			strings.TrimPrefix(args, "REGEX:"),
		}

		found := -1

	Outer:
		for i, pattern := range s.Channel.FilterBannedPhrasesPatterns {
			for _, t := range tries {
				if strings.EqualFold(pattern, t) {
					found = i
					break Outer
				}
			}
		}

		if found == -1 {
			return s.Reply("Banned phrase not found.")
		}

		l := s.Channel.FilterBannedPhrasesPatterns
		l = append(l[:found], l[found+1:]...)
		s.Channel.FilterBannedPhrasesPatterns = l
		column = models.ChannelColumns.FilterBannedPhrasesPatterns
		response = "Banned phrase removed."

	case "list":
		// TODO: Replace with link to site (or other).

		count := len(s.Channel.FilterBannedPhrasesPatterns)
		if count == 1 {
			return s.Reply("There is 1 banned phrase.")
		}

		return s.Replyf("There are %d banned phrases.", count)

	default:
		return s.ReplyUsage("on|off|add|delete|clear|list")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return err
	}

	return s.Reply(response)
}
