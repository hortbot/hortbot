package bot

import (
	"context"
	"strconv"
	"time"

	"github.com/hako/durafmt"
)

var moderationCommands = newHandlerMap(map[string]handlerFunc{
	"+b":   {fn: cmdModBan, minLevel: levelModerator},
	"-b":   {fn: cmdModUnban, minLevel: levelModerator},
	"+t":   {fn: cmdModTimeout, minLevel: levelModerator},
	"-t":   {fn: cmdModUntimeout, minLevel: levelModerator},
	"+p":   {fn: cmdModPurge, minLevel: levelModerator},
	"+m":   {fn: cmdChangeMode("slow", "Chat is now in slow mode."), minLevel: levelModerator},
	"-m":   {fn: cmdChangeMode("slowoff", "Chat is no longer in slow mode."), minLevel: levelModerator},
	"+s":   {fn: cmdChangeMode("subscribers", "Chat is now in subscribers only mode."), minLevel: levelModerator},
	"-s":   {fn: cmdChangeMode("subscribersoff", "Chat is no longer in subscribers only mode."), minLevel: levelModerator},
	"+r9k": {fn: cmdChangeMode("r9kbeta", "Chat is now in r9k mode."), minLevel: levelModerator},
	"-r9k": {fn: cmdChangeMode("r9kbetaoff", "Chat is no longer in r9k mode."), minLevel: levelModerator},
})

func isModerationCommand(prefix, name string) (prefixAndName string, ok bool) {
	if prefix != "+" && prefix != "-" {
		return "", false
	}

	prefixAndName = prefix + name
	_, ok = moderationCommands[prefixAndName]
	return prefixAndName, ok
}

func cmdModBan(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply(ctx, "Usage: +b <user>")
	}

	user = cleanUsername(user)

	if err := s.SendCommand(ctx, "ban", user); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s has been banned.", user)
}

func cmdModUnban(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply(ctx, "Usage: -b <user>")
	}

	user = cleanUsername(user)

	if err := s.SendCommand(ctx, "unban", user); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s has been unbanned.", user)
}

func cmdModTimeout(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.Reply(ctx, "Usage: +t <user> [seconds]")
	}

	user, args := splitSpace(args)
	seconds, _ := splitSpace(args)

	if user == "" {
		return usage()
	}

	user = cleanUsername(user)

	if seconds == "" {
		if err := s.SendCommand(ctx, "timeout", user); err != nil {
			return err
		}

		return s.Replyf(ctx, "%s has been timed out.", user)
	}

	if _, err := strconv.Atoi(seconds); err != nil {
		return usage()
	}

	if err := s.SendCommand(ctx, "timeout", user, seconds); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s has been timed out for %s seconds.", user, seconds)
}

func cmdModUntimeout(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply(ctx, "Usage: -t <user>")
	}

	user = cleanUsername(user)

	if err := s.SendCommand(ctx, "untimeout", user); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s is no longer timed out.", user)
}

func cmdChangeMode(command, message string) func(ctx context.Context, s *session, cmd string, args string) error {
	return func(ctx context.Context, s *session, cmd string, args string) error {
		if err := s.SendCommand(ctx, command); err != nil {
			return err
		}
		return s.Reply(ctx, message)
	}
}

func cmdModPurge(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.Reply(ctx, "Usage: +p <user>")
	}

	user, _ := splitSpace(args)

	// TODO: Accept an argument for the last N messages,
	// don't use a timeout and instead remove some large number

	if user == "" {
		return usage()
	}

	user = cleanUsername(user)

	if err := s.SendCommand(ctx, "timeout", user, "1"); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s's chat history has been purged.", user)
}

func cmdModClear(ctx context.Context, s *session, cmd string, args string) error {
	return s.SendCommand(ctx, "clear")
}

const permitDur = 5 * time.Minute

var permitDurReadable = durafmt.Parse(permitDur).String()

func cmdPermit(ctx context.Context, s *session, cmd string, args string) error {
	if !s.Channel.EnableFilters || !s.Channel.FilterLinks {
		return nil
	}

	user, _ := splitSpace(args)
	if user == "" {
		return s.ReplyUsage(ctx, "<user>")
	}
	user = cleanUsername(user)

	if err := s.LinkPermit(ctx, user, permitDur); err != nil {
		return err
	}

	return s.Replyf(ctx, "%s may now post one link within "+permitDurReadable+".", user)
}
