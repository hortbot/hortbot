package bot

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
)

var moderationCommands handlerMap = map[string]handlerFunc{
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
}

func cmdModBan(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply("usage: +b <user>")
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("ban", user); err != nil {
		return err
	}

	return s.Replyf("%s has been banned.", user)
}

func cmdModUnban(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply("usage: -b <user>")
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("unban", user); err != nil {
		return err
	}

	return s.Replyf("%s has been unbanned.", user)
}

func cmdModTimeout(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.Reply("usage: +t <user> [seconds]")
	}

	user, args := splitSpace(args)
	seconds, _ := splitSpace(args)

	if user == "" {
		return usage()
	}

	user = strings.ToLower(user)

	if seconds == "" {
		if err := s.SendCommand("timeout", user); err != nil {
			return err
		}

		return s.Replyf("%s has been timed out.", user)
	}

	if _, err := strconv.Atoi(seconds); err != nil {
		return usage()
	}

	if err := s.SendCommand("timeout", user, seconds); err != nil {
		return err
	}

	return s.Replyf("%s has been timed out for %s seconds.", user, seconds)
}

func cmdModUntimeout(ctx context.Context, s *session, cmd string, args string) error {
	user, _ := splitSpace(args)

	if user == "" {
		return s.Reply("usage: -t <user>")
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("untimeout", user); err != nil {
		return err
	}

	return s.Replyf("%s is no longer timed out.", user)
}

func cmdChangeMode(command, message string) func(ctx context.Context, s *session, cmd string, args string) error {
	return func(ctx context.Context, s *session, cmd string, args string) error {
		if err := s.SendCommand(command); err != nil {
			return err
		}
		return s.Reply(message)
	}
}

func cmdModPurge(ctx context.Context, s *session, cmd string, args string) error {
	usage := func() error {
		return s.Reply("usage: +p <user>")
	}

	user, _ := splitSpace(args)

	// TODO: Accept an argument for the last N messages,
	// don't use a timeout and instead remove some large number

	if user == "" {
		return usage()
	}

	user = strings.ToLower(user)

	if err := s.SendCommand("timeout", user, "1"); err != nil {
		return err
	}

	return s.Replyf("%s's chat history has been purged.", user)
}

func cmdModClear(ctx context.Context, s *session, cmd string, args string) error {
	return s.SendCommand("clear")
}

const permitDur = 5 * time.Minute

var (
	permitSeconds     = int(permitDur.Seconds())
	permitDurReadable = durafmt.Parse(permitDur).String()
)

func cmdPermit(ctx context.Context, s *session, cmd string, args string) error {
	if !s.Channel.EnableFilters || !s.Channel.FilterLinks {
		return nil
	}

	user, _ := splitSpace(args)
	if user == "" {
		return s.ReplyUsage("<user>")
	}
	user = strings.ToLower(user)

	if err := s.LinkPermit(user, permitSeconds); err != nil {
		return err
	}

	return s.Replyf("%s may now post one link within "+permitDurReadable+".", user)
}
