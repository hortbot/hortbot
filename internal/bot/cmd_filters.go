package bot

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/hortbot/hortbot/internal/db/models"
	"github.com/hortbot/hortbot/internal/pkg/linkmatch"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var filterCommands = newHandlerMap(map[string]handlerFunc{
	"status":        {fn: cmdFilterStatus, minLevel: AccessLevelModerator},
	"on":            {fn: cmdFilterOnOff(true), minLevel: AccessLevelModerator},
	"off":           {fn: cmdFilterOnOff(false), minLevel: AccessLevelModerator},
	"links":         {fn: cmdFilterLinks, minLevel: AccessLevelModerator},
	"pd":            {fn: cmdFilterPermittedLinks, minLevel: AccessLevelModerator},
	"pl":            {fn: cmdFilterPermittedLinks, minLevel: AccessLevelModerator},
	"caps":          {fn: cmdFilterCaps, minLevel: AccessLevelModerator},
	"symbols":       {fn: cmdFilterSymbols, minLevel: AccessLevelModerator},
	"me":            {fn: cmdFilterMe, minLevel: AccessLevelModerator},
	"messagelength": {fn: cmdFilterMessageLength, minLevel: AccessLevelModerator},
	"emotes":        {fn: cmdFilterEmotes, minLevel: AccessLevelModerator},
	"banphrase":     {fn: cmdFilterBanPhrase, minLevel: AccessLevelModerator},
	"exempt":        {fn: cmdFilterExemptLevel, minLevel: AccessLevelModerator},
})

func cmdFilter(ctx context.Context, s *session, cmd string, args string) error {
	subcommand, args := splitSpace(args)
	subcommand = strings.ToLower(subcommand)

	if subcommand == "" {
		return s.ReplyUsage(ctx, "<option> ...")
	}

	ok, err := filterCommands.Run(ctx, s, subcommand, args)
	if !ok {
		return s.Replyf(ctx, "No such filter option '%s'.", subcommand)
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

	return s.Reply(ctx, builder.String())
}

func cmdFilterOnOff(enable bool) func(ctx context.Context, s *session, cmd string, args string) error {
	return func(ctx context.Context, s *session, cmd string, args string) error {
		if s.Channel.EnableFilters == enable {
			if enable {
				return s.Reply(ctx, "Filters are already enabled.")
			}
			return s.Reply(ctx, "Filters are already disabled.")
		}

		s.Channel.EnableFilters = enable

		if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.EnableFilters)); err != nil {
			return fmt.Errorf("updating channel: %w", err)
		}

		if enable {
			return s.Reply(ctx, "Filters are now enabled.")
		}
		return s.Reply(ctx, "Filters are now disabled.")
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
		return s.ReplyUsage(ctx, "on|off")
	}

	if s.Channel.FilterLinks == enable {
		if enable {
			return s.Reply(ctx, "Link filter is already enabled.")
		}
		return s.Reply(ctx, "Link filter is already disabled.")
	}

	s.Channel.FilterLinks = enable

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterLinks)); err != nil {
		return fmt.Errorf("updating channel: %w", err)
	}

	if enable {
		return s.Reply(ctx, "Link filter is now enabled.")
	}
	return s.Reply(ctx, "Link filter is now disabled.")
}

func cmdFilterPermittedLinks(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.ReplyUsage(ctx, "add|delete|list ...")
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
			return s.Reply(ctx, "There are no permitted link patterns.")
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

		return s.Reply(ctx, builder.String())

	case "add":
		pd, args := splitSpace(args)
		if args != "" {
			return s.ReplyUsage(ctx, subcommand+" <link pattern>")
		}

		if linkmatch.IsBadPattern(pd) {
			return s.Replyf(ctx, "Pattern '%s' is too permissive.", pd)
		}

		s.Channel.PermittedLinks = append(s.Channel.PermittedLinks, pd)

		if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.PermittedLinks)); err != nil {
			return fmt.Errorf("updating channel: %w", err)
		}

		n := len(s.Channel.PermittedLinks)
		return s.Replyf(ctx, "Permitted link pattern #%d added.", n)

	case "delete", "remove":
		n, err := strconv.Atoi(args)
		if err != nil || n <= 0 {
			return s.ReplyUsage(ctx, subcommand+" <num>")
		}

		i := n - 1

		old := s.Channel.PermittedLinks

		if i >= len(old) {
			return s.Replyf(ctx, "Permitted link pattern #%d does not exist.", n)
		}

		oldP := old[i]
		s.Channel.PermittedLinks = append(old[:i], old[i+1:]...) //nolint:gocritic

		if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.PermittedLinks)); err != nil {
			return fmt.Errorf("updating channel: %w:", err)
		}

		return s.Replyf(ctx, "Permitted link pattern #%d deleted; was '%s'.", n, oldP)

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
			return s.Reply(ctx, "Caps filter is already enabled.")
		}

		s.Channel.FilterCaps = true
		response = "Caps filter is now enabled."
		column = models.ChannelColumns.FilterCaps

	case "off":
		if !s.Channel.FilterCaps {
			return s.Reply(ctx, "Caps filter is already disabled.")
		}

		s.Channel.FilterCaps = false
		response = "Caps filter is now disabled."
		column = models.ChannelColumns.FilterCaps

	case "percent":
		percent, err := strconv.Atoi(args)
		if err != nil || percent < 0 || percent > 100 {
			return s.ReplyUsage(ctx, "percent <0-100>")
		}

		s.Channel.FilterCapsPercentage = percent
		response = fmt.Sprintf("Caps filter percent set to %d%%.", percent)
		column = models.ChannelColumns.FilterCapsPercentage

	case "minchars":
		minChars, err := strconv.Atoi(args)
		if err != nil || minChars < 0 {
			return s.ReplyUsage(ctx, "minchars <int>")
		}

		s.Channel.FilterCapsMinChars = minChars
		response = fmt.Sprintf("Caps filter min chars set to %d.", minChars)
		column = models.ChannelColumns.FilterCapsMinChars

	case "mincaps":
		minCaps, err := strconv.Atoi(args)
		if err != nil || minCaps < 0 {
			return s.ReplyUsage(ctx, "mincaps <int>")
		}

		s.Channel.FilterCapsMinCaps = minCaps
		response = fmt.Sprintf("Caps filter min caps set to %d.", minCaps)
		column = models.ChannelColumns.FilterCapsMinCaps

	case "status":
		return s.Replyf(ctx, "Caps filter=%v, percent=%v, minchars=%v, mincaps=%v",
			s.Channel.FilterCaps,
			s.Channel.FilterCapsPercentage,
			s.Channel.FilterCapsMinChars,
			s.Channel.FilterCapsMinCaps,
		)

	default:
		return s.ReplyUsage(ctx, "on|off|percent|minchars|mincaps|status")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	return s.Reply(ctx, response)
}

func cmdFilterSymbols(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterSymbols {
			return s.Reply(ctx, "Symbols filter is already enabled.")
		}

		s.Channel.FilterSymbols = true
		response = "Symbols filter is now enabled."
		column = models.ChannelColumns.FilterSymbols

	case "off":
		if !s.Channel.FilterSymbols {
			return s.Reply(ctx, "Symbols filter is already disabled.")
		}

		s.Channel.FilterSymbols = false
		response = "Symbols filter is now disabled."
		column = models.ChannelColumns.FilterSymbols

	case "percent":
		percent, err := strconv.Atoi(args)
		if err != nil || percent < 0 || percent > 100 {
			return s.ReplyUsage(ctx, "percent <0-100>")
		}

		s.Channel.FilterSymbolsPercentage = percent
		response = fmt.Sprintf("Symbols filter percent set to %d%%.", percent)
		column = models.ChannelColumns.FilterSymbolsPercentage

	case "min":
		minValue, err := strconv.Atoi(args)
		if err != nil || minValue < 0 {
			return s.ReplyUsage(ctx, "min <int>")
		}

		s.Channel.FilterSymbolsMinSymbols = minValue
		response = fmt.Sprintf("Symbols filter min symbols set to %d.", minValue)
		column = models.ChannelColumns.FilterSymbolsMinSymbols

	case "status":
		return s.Replyf(ctx, "Symbols filter=%v, percent=%v, min=%v",
			s.Channel.FilterSymbols,
			s.Channel.FilterSymbolsPercentage,
			s.Channel.FilterSymbolsMinSymbols,
		)

	default:
		return s.ReplyUsage(ctx, "on|off|percent|min|status")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	return s.Reply(ctx, response)
}

func cmdFilterMe(ctx context.Context, s *session, cmd string, args string) error {
	enable := false

	switch args {
	case "on":
		enable = true
	case "off":
		// Do nothing.
	default:
		return s.ReplyUsage(ctx, "on|off")
	}

	if s.Channel.FilterMe == enable {
		if enable {
			return s.Reply(ctx, "Me filter is already enabled.")
		}
		return s.Reply(ctx, "Me filter is already disabled.")
	}

	s.Channel.FilterMe = enable

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterMe)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	if enable {
		return s.Reply(ctx, "Me filter is now enabled.")
	}
	return s.Reply(ctx, "Me filter is now disabled.")
}

func cmdFilterMessageLength(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.Replyf(ctx, "Max message length set to %d.", s.Channel.FilterMaxLength)
	}

	n, err := strconv.Atoi(args)
	if err != nil || n < 0 {
		return s.ReplyUsage(ctx, "<length>")
	}

	s.Channel.FilterMaxLength = n

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterMaxLength)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	return s.Replyf(ctx, "Max message length set to %d.", n)
}

func cmdFilterEmotes(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterEmotes {
			return s.Reply(ctx, "Emote filter is already enabled.")
		}

		s.Channel.FilterEmotes = true
		response = "Emote filter is now enabled."
		column = models.ChannelColumns.FilterEmotes

	case "off":
		if !s.Channel.FilterEmotes {
			return s.Reply(ctx, "Emote filter is already disabled.")
		}

		s.Channel.FilterEmotes = false
		response = "Emote filter is now disabled."
		column = models.ChannelColumns.FilterEmotes

	case "max":
		maxValue, err := strconv.Atoi(args)
		if err != nil || maxValue < 0 {
			return s.ReplyUsage(ctx, "max <num>")
		}

		s.Channel.FilterEmotesMax = maxValue
		response = fmt.Sprintf("Emote filter max emotes set to %d.", maxValue)
		column = models.ChannelColumns.FilterEmotesMax

	case "single":
		enabled := false

		switch args {
		case "on":
			enabled = true
		case "off":
		default:
			return s.ReplyUsage(ctx, "single on|off")
		}

		if s.Channel.FilterEmotesSingle == enabled {
			if enabled {
				return s.Reply(ctx, "Single emote filter is already enabled.")
			}
			return s.Reply(ctx, "Single emote filter is already disabled.")
		}

		s.Channel.FilterEmotesSingle = enabled

		if enabled {
			response = "Single emote filter is now enabled."
		} else {
			response = "Single emote filter is now disabled."
		}

		column = models.ChannelColumns.FilterEmotesSingle

	case "status":
		return s.Replyf(ctx, "Emote filter=%v, max=%v, single=%v",
			s.Channel.FilterEmotes,
			s.Channel.FilterEmotesMax,
			s.Channel.FilterEmotesSingle,
		)

	default:
		return s.ReplyUsage(ctx, "on|off|max|single|status")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	return s.Reply(ctx, response)
}

func cmdFilterBanPhrase(ctx context.Context, s *session, cmd string, args string) error {
	var response string
	var column string

	subcommand, args := splitSpace(args)

	switch subcommand {
	case "on":
		if s.Channel.FilterBannedPhrases {
			return s.Reply(ctx, "Banned phrase filter is already enabled.")
		}

		s.Channel.FilterBannedPhrases = true
		column = models.ChannelColumns.FilterBannedPhrases
		response = "Banned phrase filter is now enabled."

	case "off":
		if !s.Channel.FilterBannedPhrases {
			return s.Reply(ctx, "Banned phrase filter is already disabled.")
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
			if !s.UserLevel.CanAccess(AccessLevelAdmin) {
				return s.Reply(ctx, "Only admins may add regex banned words.")
			}

			pattern = strings.TrimPrefix(args, "REGEX:")
			if pattern == "" {
				return s.replyBadPattern(ctx, errEmptyPattern)
			}

			_, err := s.Deps.ReCache.Compile(pattern)
			if err != nil {
				return s.replyBadPattern(ctx, err)
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
			return s.Reply(ctx, "Banned phrase not found.")
		}

		l := s.Channel.FilterBannedPhrasesPatterns
		l = append(l[:found], l[found+1:]...)
		s.Channel.FilterBannedPhrasesPatterns = l
		column = models.ChannelColumns.FilterBannedPhrasesPatterns
		response = "Banned phrase removed."

	case "list":
		count := len(s.Channel.FilterBannedPhrasesPatterns)
		if count == 1 {
			return s.Reply(ctx, "There is 1 banned phrase.")
		}

		return s.Replyf(ctx, "There are %d banned phrases.", count)

	default:
		return s.ReplyUsage(ctx, "on|off|add|delete|clear|list")
	}

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, column)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	return s.Reply(ctx, response)
}

func cmdFilterExemptLevel(ctx context.Context, s *session, cmd string, args string) error {
	if args == "" {
		return s.Replyf(ctx, "Filter exempt level is set to %s.", flect.Pluralize(s.Channel.FilterExemptLevel))
	}

	newLevel := parseLevel(args)
	if newLevel == AccessLevelUnknown || !AccessLevelModerator.CanAccess(newLevel) {
		return s.Reply(ctx, "Invalid level.")
	}

	newLevelPG := newLevel.PGEnum()

	if s.Channel.FilterExemptLevel == newLevelPG {
		return s.Replyf(ctx, "Filter exempt level is already set to %s.", flect.Pluralize(newLevel.PGEnum()))
	}

	s.Channel.FilterExemptLevel = newLevelPG

	if err := s.Channel.Update(ctx, s.Tx, boil.Whitelist(models.ChannelColumns.UpdatedAt, models.ChannelColumns.FilterExemptLevel)); err != nil {
		return fmt.Errorf("updating channel: %w:", err)
	}

	return s.Replyf(ctx, "Filter exempt level set to %s.", flect.Pluralize(newLevelPG))
}
